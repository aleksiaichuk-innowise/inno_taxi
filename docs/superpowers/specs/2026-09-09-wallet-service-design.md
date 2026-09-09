# Wallet Service — Design

## Context

`user_service`, `driver_service`, `order_service`, `auth_service`, and
`gateway_service` all exist now. Wallet Service is the sixth: a
financial ledger, called only by other services, never directly by a
client. README pins its stack explicitly and it differs from every
service built so far in two ways worth calling out up front:

- **`sqlx`, not raw `pgx`.** `order_service` talks to Postgres directly
  through `pgx/v5`. README pins Wallet Service to "sqlx with transaction
  support" specifically — a deliberate, per-service choice, not an
  oversight to "fix" into consistency with `order_service`.
- **This is the first service whose core operation is genuinely a
  multi-step atomic write** (check balance, deduct, record the
  transaction) — exactly the shape `ARCHITECTURE.md`'s
  `transactionalOperation`/`PgAtomicRepository` pattern was written for.
  `order_service`'s `CreateOrder` was a single INSERT and never needed it
  (`ARCHITECTURE.md`'s own "when to use vs. skip" table: "Single
  INSERT/UPDATE → No"). This spec is where that pattern gets its first
  real implementation.

## Scope (from `README.md`'s Wallet Service section)

In scope:
- **Wallet creation**: `POST /internal/wallets` — one wallet per user,
  balance starts at 0.
- **Charge** (immediate deduction): `POST /internal/wallets/{user_id}/charge`
  — atomically checks sufficient balance, deducts, records a `debit`
  transaction.
- **Refund** (compensatory rollback): `POST /internal/wallets/{user_id}/refund`
  — atomically credits back, records a `credit` transaction.
- **Transaction history**: `GET /internal/wallets/{user_id}/transactions`
  — recent transactions, Redis-cached per README's "Caching: Redis for
  storing recent transactions."
- **Balance read**: `GET /internal/wallets/{user_id}`.
- Infra: Postgres (own database, own container) + Redis (shared
  container — Auth Service already stood one up, and README doesn't ask
  for per-service Redis isolation the way it does for each service's own
  Postgres/Mongo database).

Explicitly out of scope for this step:
- **Anyone actually calling these endpoints.** Neither `user_service`'s
  registration flow nor `order_service`'s (nonexistent) payment step
  calls Wallet Service yet. This mirrors `driver_service`'s own
  `POST /internal/drivers`, which nothing calls either — both are
  real gaps, but wiring them is naturally *one* future piece of work
  (a `user_registered` Kafka event fanning out to both
  `driver_service` and `wallet_service` consumers, now that both exist),
  not something to bolt onto Wallet Service alone.
- **Field-level encryption for "sensitive financial data."** README's
  Security bullet names this, but the only data actually stored here is
  an integer balance and a ledger of amounts/references — no card
  numbers, no bank details, nothing that maps to a concrete secret to
  encrypt. Standing up KMS/envelope-encryption infrastructure to protect
  an `int64` column with nothing sensitive in it is security theater,
  not security. Revisit if a real payment-method field is ever added.
- **Currency.** README doesn't mention multi-currency, and neither does
  any other service (`order_service`'s `price_minor_units` has no
  currency column either). Single implicit currency, matching precedent.
- **Validating a refund against a prior charge.** "Handles refunds and
  cancellations" doesn't require proving the money was ever taken in the
  first place — that linkage belongs to whatever calls both endpoints
  (`order_service`, once it has payment integration), not to Wallet
  Service policing its caller's business logic.

## Decisions made during design (with rationale)

- **Every endpoint lives under `/internal/*` with no auth check at
  all**, exactly like `user_service`'s `/internal/verify-credentials` and
  `driver_service`'s `/internal/drivers`. README's "Internal service
  authentication for secure inter-service communication" is satisfied
  the same way it already is for those two: trust by network position
  (not reachable through `gateway_service`, which never routes
  `/internal/*` — see the gateway design doc), not a bearer token. No new
  auth mechanism invented for this one service.
- **`sqlx` over `pgx/v5/stdlib`**, not `lib/pq`. README says "sqlx",
  which needs a `database/sql` driver underneath — `pgx/v5/stdlib`
  registers itself as one. Reusing the same underlying driver
  `order_service` already depends on (just accessed through `sqlx`
  instead of directly) beats pulling in a second, unrelated Postgres
  driver for no functional benefit.
- **Migrations follow `order_service`'s established Postgres
  convention exactly: goose-formatted SQL files, applied manually via
  Makefile targets, not run automatically on startup.** `order_service`'s
  own `app/db/postgres/postgres.go` only connects and pings — it never
  migrates. Auto-migrating here (which the Mongo-based services *do* do
  on startup) would be a nicer default in isolation, but it would make
  Wallet Service the only Postgres service in the repo that behaves
  differently from its one precedent for no stated reason. Consistency
  with the existing Postgres service outweighs a marginal UX
  improvement, so the pattern doesn't get invented twice.
- **Idempotency by `(wallet_id, reference_id, type)`, enforced at both
  layers.** A charge/refund call carries a caller-supplied `reference_id`
  (e.g. an order ID). Inside the transaction, the code first checks for
  an existing transaction with the same `(wallet_id, reference_id,
  type)` and returns it unchanged (HTTP 200, not 201) if found — this
  covers the common case. A `UNIQUE (wallet_id, reference_id, type)`
  constraint backs it up for the race where two concurrent requests both
  pass that check before either commits; the loser's insert fails with a
  duplicate-key error, which the code catches and treats the same as
  finding it on the read (fetch and return the winner's row, not a
  hard error). Without this, a network retry from `order_service` in a
  future step could double-charge a rider.
- **Row-level locking on charge/refund** (`SELECT ... FOR UPDATE` on the
  wallet row, inside the transaction, before checking/deducting balance)
  — needed so two concurrent charges against the same wallet can't both
  read the same starting balance and both succeed when only one should.
- **Insufficient funds is a rejected request (`402 Payment Required`),
  not a recorded "failed" transaction.** Nothing happened to the wallet,
  so there's nothing to put in the ledger — recording a no-op as a
  "failed" row would just be noise an auditor has to filter back out.
- **No `repository/repository.go` aggregate struct.** `ARCHITECTURE.md`'s
  file-tree example shows one (`Repository{Pg, Mongo, Redis}`), but
  neither `order_service` nor `driver_service` actually built it — each
  just constructs its Postgres/Mongo repo directly and passes it to the
  service constructor. `WalletService` takes `WalletRepository` and
  `TransactionCache` as two separate constructor params instead, matching
  what's actually been built twice now rather than a documented pattern
  that's never once been realized.
- **Redis caches `ListTransactions`' recent-N read, invalidated on
  write.** Cache-aside: a miss reads Postgres and populates Redis with a
  short TTL; any successful charge/refund on that wallet deletes the
  cache key so the next read isn't stale. This is the one place README's
  "Caching: Redis for storing recent transactions" line actually applies
  — wallet creation and balance reads aren't in that line and don't get
  cached.

## Data model

```go
// entity/service/wallet.go
type Wallet struct {
    ID                string
    UserID            string
    BalanceMinorUnits int64
    CreatedAt         time.Time
    UpdatedAt         time.Time
}

type TransactionType string

const (
    TransactionTypeDebit  TransactionType = "debit"  // charge
    TransactionTypeCredit TransactionType = "credit" // refund
)

type Transaction struct {
    ID               string
    WalletID         string
    Type             TransactionType
    AmountMinorUnits int64
    ReferenceID      string
    CreatedAt        time.Time
}
```

## Architecture

```
services/wallet_service/
  go.mod                          — require .../shared, replace ../../shared
  cmd/main.go
  config/config.go                — Postgres, Redis, HTTP config

  app/
    run.go                        — wires deps, starts HTTP server,
                                     graceful shutdown (SIGINT/SIGTERM)
    db/
      postgres/postgres.go        — *sqlx.DB via pgx/v5/stdlib, connect + ping only
      redis/redis.go               — Redis client setup

  entity/
    service/wallet.go             — Wallet, Transaction, TransactionType (no tags)
    repository/
      wallet.go                    — Wallet row (db tags)
      transaction.go                — Transaction row (db tags)

  handler/
    http/
      http.go                     — Handler struct + constructor
      create_wallet.go             — POST /internal/wallets
      get_wallet.go                 — GET /internal/wallets/:user_id
      charge.go                      — POST /internal/wallets/:user_id/charge
      refund.go                       — POST /internal/wallets/:user_id/refund
      list_transactions.go             — GET /internal/wallets/:user_id/transactions

  service/
    service.go                    — WalletRepository, TransactionCache
                                     interfaces, WalletService struct, New()
    transactional_operation.go     — transactionalOperation helper
    create_wallet.go
    get_wallet.go
    charge.go
    refund.go
    list_transactions.go

  repository/
    pg_repo/
      pg_repo.go                   — PgRepository struct wrapping *sqlx.DB
      pg_atomic.go                  — PgAtomicRepository: Start/Finish/Abort
      create_wallet.go
      get_wallet_by_user_id.go
      get_wallet_for_update.go       — SELECT ... FOR UPDATE, tx-scoped
      update_balance.go               — tx-scoped
      find_transaction_by_reference.go — tx-scoped, idempotency check
      create_transaction.go            — tx-scoped
      list_recent_transactions.go
    redis_repo/
      redis_repo.go                 — recent-transactions cache (get/set/invalidate)

  migrations/postgres/
    ..._create_wallets_table.sql
    ..._create_transactions_table.sql

  errorsx/errors.go                — ErrWalletAlreadyExists, ErrWalletNotFound,
                                      ErrInvalidAmount, ErrInsufficientFunds
```

`service.WalletRepository` (satisfied by `repository/pg_repo`):

```go
type WalletRepository interface {
    CreateWallet(ctx context.Context, userID string) (service_dto.Wallet, error)
    GetWalletByUserID(ctx context.Context, userID string) (service_dto.Wallet, error)

    PgAtomicRepository // Start/Finish/Abort

    GetWalletForUpdate(ctx context.Context, tx *sqlx.Tx, walletID string) (service_dto.Wallet, error)
    UpdateBalance(ctx context.Context, tx *sqlx.Tx, walletID string, newBalance int64) error
    FindTransactionByReference(ctx context.Context, tx *sqlx.Tx, walletID, referenceID string, t service_dto.TransactionType) (service_dto.Transaction, bool, error)
    CreateTransaction(ctx context.Context, tx *sqlx.Tx, walletID string, t service_dto.TransactionType, amount int64, referenceID string) (service_dto.Transaction, error)

    ListRecentTransactions(ctx context.Context, walletID string, limit int) ([]service_dto.Transaction, error)
}

type TransactionCache interface {
    GetRecent(ctx context.Context, walletID string) ([]service_dto.Transaction, bool, error)
    SetRecent(ctx context.Context, walletID string, txs []service_dto.Transaction) error
    Invalidate(ctx context.Context, walletID string) error
}
```

## Endpoints

| Method | Path | Request | Response |
|---|---|---|---|
| `POST` | `/internal/wallets` | `{user_id}` | `201 {wallet}` / `422` already exists |
| `GET` | `/internal/wallets/{user_id}` | — | `200 {wallet}` / `404` |
| `POST` | `/internal/wallets/{user_id}/charge` | `{amount_minor_units, reference_id}` | `201 {transaction}` (new) / `200 {transaction}` (idempotent replay) / `402` insufficient funds / `404` wallet / `422` invalid amount |
| `POST` | `/internal/wallets/{user_id}/refund` | `{amount_minor_units, reference_id}` | `201 {transaction}` / `200 {transaction}` (replay) / `404` / `422` |
| `GET` | `/internal/wallets/{user_id}/transactions` | — | `200 [transaction...]` (last 20, Redis-cached) |

## Testing

README calls out "financial transaction testing suites" specifically for
this service. Beyond the fake-repository unit tests every service has
gotten, `repository/pg_repo` gets a `//go:build integration` dockertest
suite (same tool `user_service` already uses, against a real
`postgres:18` container) proving the two things a fake repository
structurally cannot: the `FOR UPDATE` lock actually serializes concurrent
charges against the same wallet, and the `UNIQUE (wallet_id,
reference_id, type)` constraint actually rejects the race it exists for.

## Deployment

`docker-compose.yaml` gets a `wallet_postgres` container (own database,
per `ARCHITECTURE.md`: "Each service owns its data" — not sharing
`order_service`'s `postgres` container/database) and a `wallet_service`
entry depending on it plus the existing `redis` container. Makefile gets
`build-wallet`/`run-wallet`/`test-wallet`/`test-wallet-integration` and a
`migrate-wallet-*` goose target block, mirroring `order_service`'s
exactly. Not routed through `gateway_service` — see "no auth check"
above; there's nothing here for `gateway_service`'s `nginx.conf` to
route to.

## Explicitly not decided here (deferred to later work)

- The `user_registered` Kafka event and its two consumers
  (`driver_service`, `wallet_service`) — now that both services exist,
  this is ripe to become its own spec.
- `order_service` actually calling `charge`/`refund` at the right points
  in the order lifecycle — `order_service`'s own follow-up.
- Whether a user-facing "check my balance" surface ever gets added to
  `user_service` (its own "Wallet Integration" functional requirement) —
  not this service's concern either way.
