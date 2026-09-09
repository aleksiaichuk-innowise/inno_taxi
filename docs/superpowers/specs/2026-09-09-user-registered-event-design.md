# `user_registered` Kafka event — design

## Problem

Per `README.md`:
- "Wallet Creation: Automatically creates personal wallets upon user/driver registration"
- "Driver Service Registration: ... invoked by the User service when a user with the 'Driver' registration role initiates the process"
- "Analytic Service Data Collection: Receives events from other services via Kafka (user registrations, completed orders, ratings)"

Today `user_service.CreateUser` does none of this: it inserts a Mongo row and returns. `driver_service`'s `POST /internal/drivers` and `wallet_service`'s `POST /internal/wallets` exist and work, but nothing calls them. `analytic_service` only consumes `order_created`.

## Decision: Kafka fan-out, not synchronous HTTP calls

`order_service` → `analytic_service` already established the pattern (sarama producer/consumer, single-broker KRaft `apache/kafka:3.9.0`, protobuf payloads, at-least-once, mark-processed-even-on-error). Reuse it instead of having `user_service` call three other services' internal HTTP endpoints synchronously:
- Registration doesn't need to wait on driver-profile/wallet/analytics provisioning — matches "automatically creates" in the README (a background effect of registration, not a precondition for the HTTP response succeeding).
- `driver_service` and `wallet_service` consume the event and call their **own** `CreateDriver`/`CreateWallet` service methods in-process — not their own `/internal/*` HTTP endpoint via loopback. Those endpoints stay as-is for direct/manual/admin use; the Kafka path is now the actual provisioning path.
- Consistent with the "services own their side effects" shape the codebase already has.

Publish happens **after** the Mongo insert commits, best-effort: on publish failure, `user_service` logs and returns the created user anyway (HTTP 201) rather than failing an already-committed registration. There's no outbox/transactional-publish anywhere else in this repo either (order_service has the same gap) — not introducing it here is consistent, not a new corner cut.

## Event shape

New proto package `shared/proto/user_service/` (this service had none before — no gRPC surface, Kafka-only, so a single `kafka.proto` with a plain `--go_out` is enough; no `--go-grpc_out`/`--grpc-gateway_out`/`--openapiv2_out`, unlike `order_service`'s Makefile target).

```proto
syntax = "proto3";
package user.kafka.v1;
option go_package = ".../shared/proto/user_service;user_service";

message UserRegisteredEvent {
  string user_id = 1;
  string name = 2;
  string email = 3;
  string phone = 4;
  string role = 5;         // "user" or "driver" — the single role chosen at registration
  string created_at = 6;   // RFC3339, matches order_service's convention
}
```

`role` is a plain string, not a shared proto enum, deliberately breaking from `order_service`'s `TaxiType`/`Status` enum convention: those enums are reused across multiple messages/services already, which is what justifies the shared-enum machinery. `role` here has exactly two possible values at registration time (`user`, `driver` — `admin`/`analyst` are assigned later, out of band, never at registration) and is consumed by three different services each with their own local `Role`/string conventions. A shared enum would add cross-module coupling for no real benefit; a string is simpler and sufficient.

Topic: `user_registered`, one topic for both roles — consumers filter by `role` rather than getting separate topics, matching how `order_created` carries all order states rather than being split per-state.

## Per-service changes

**`user_service` (producer)**
- `config.Config` gains `Kafka shared.KafkaConfig` (same `KAFKA_BROKERS`/`KAFKA_USERNAME`/`KAFKA_PASSWORD` env vars as `order_service`/`analytic_service`).
- `app/kafka/publisher.go`: copy of `order_service`'s (`RequiredAcks: WaitForAll`, `Idempotent: true`, `Retry.Max: 5`). Not extracted to `shared` — `order_service` doesn't share it either; not doing an unrelated cross-cutting refactor as a side effect of this feature.
- `gateway/kafka.go`: `KafkaGateway` interface, `PublishUserRegistered(ctx, user serviceEntity.User) error` — mirrors `order_service.KafkaGateway.PublishOrderCreated`'s shape (takes the already-created domain entity, not a separate DTO). `user.Roles[0]` is used as `role` — registration always creates exactly one role (see `service/create_user.go`), so this is safe here even though `User.Roles` is a slice in general (roles can be added later via `AddRole`).
- `service.UserService` gets a second dependency, `KafkaGateway`; `NewUserService(userRepo, kafkaGateway)`. `CreateUser` publishes post-insert, logs-and-continues on publish error (see above).

**`driver_service` (consumer)**
- `config.Config` gains `Kafka shared.KafkaConfig`.
- `app/kafka/consumer.go`: `NewConsumerGroup(brokers []string) (sarama.ConsumerGroup, error)`, group ID `"driver_service"`.
- `handler/kafka/user_registered_consumer.go`: mirrors `analytic_service`'s `OrderCreatedConsumer` shape (`Setup`/`Cleanup`/`ConsumeClaim`, mark-message-always). On a message where `role != "driver"`, no-op (not every registration is a driver). Otherwise calls `DriverService.CreateDriver` directly with `TaxiType: TaxiTypeEconomy` as the default.
  - **Default taxi type, explicit scope note:** registration (`POST /register`) collects only name/email/phone/role/password (per README's User Service section) — no taxi type. Auto-created driver profiles default to `economy`; the existing `PATCH /profile` (`UpdateType`) endpoint is how a driver changes it afterward. Not adding a taxi-type field to registration — that would change `user_service`'s public contract for a detail that's driver-specific and already has its own update path.
  - `errorsx.ErrDriverAlreadyExists` from a duplicate/redelivered event is expected under at-least-once delivery (the Mongo unique index on `user_id` makes `CreateDriver` naturally idempotent) — logged at `Info`, not `Error`.

**`wallet_service` (consumer)**
- `config.Config` gains `Kafka shared.KafkaConfig`.
- `app/kafka/consumer.go`: group ID `"wallet_service"`.
- `handler/kafka/user_registered_consumer.go`: calls `WalletService.CreateWallet(ctx, event.GetUserId())` for **every** role (both `user` and `driver` get a wallet, per README). `errorsx.ErrWalletAlreadyExists` on redelivery logged at `Info` (Postgres unique constraint on `wallets.user_id` makes this idempotent too).

**`analytic_service` (second consumer + new table)**
- `app/kafka.NewConsumerGroup` gets a `groupID` parameter (was hardcoded `"analytic_service"`) so a second, independent group can read `user_registered` without interfering with the existing `order_created` consumption — two separate `sarama.ConsumerGroup` instances/goroutines, not one group subscribed to two topics, to avoid touching the working `order_created` consumer's behavior at all. Existing call site keeps its literal group ID (`"analytic_service"`, unchanged, so its committed offsets aren't affected); new call uses `"analytic_service_user_registered"`.
- `handler/kafka/user_registered_consumer.go`: same mark-always/log-and-skip shape as `order_created_consumer.go`.
- `service.OrderEventRepository` renamed to `EventRepository` (it now backs two event kinds; low blast radius — one interface, one implementing struct, two test files) and gains `InsertUserRegisteredEvent(ctx, evt) error`.
- New ClickHouse table `user_registration_events` (own schema file, second `go:embed` + `Exec` in `app/db/clickhouse/clickhouse.go`, `ReplacingMergeTree` keyed the same way as `order_events` for the same at-least-once-dedup-at-query-time reason):
  ```sql
  CREATE TABLE IF NOT EXISTS user_registration_events (
      user_id String,
      name String,
      email String,
      phone String,
      role LowCardinality(String),
      registered_at DateTime64(3),
      ingested_at DateTime64(3) DEFAULT now64(3)
  ) ENGINE = ReplacingMergeTree(ingested_at)
  ORDER BY (user_id);
  ```
- **Explicitly out of scope:** no new HTTP endpoint for registration stats (e.g. daily signups). This spec only wires the ingestion pipeline; querying it is a separate, later concern, same as how `order_events` existed before `GetOrderStats`/`GetDailyOrderCounts` were added.

## Wiring

- `docker-compose.yaml`: `user_service`, `driver_service`, `wallet_service` each gain `KAFKA_BROKERS: kafka:9092` and `depends_on: kafka: condition: service_healthy`.
- `Makefile`: new `proto-user` target (single `--go_out`, no gateway/grpc/openapi plugins), added to `.PHONY`.

## Explicit non-goals

- No outbox pattern / transactional publish for `user_service` — same at-least-once, best-effort posture the rest of the repo already has.
- No dead-letter queue for unprocessable messages — same posture as `analytic_service`'s existing consumer.
- No change to `user_service`'s registration request/response contract (no taxi type field, no synchronous "wallet created" confirmation).
- No admin/manual re-trigger endpoint for a failed auto-provisioning — if `driver_service`/`wallet_service` are down when the event is published, they'll never see it (no replay from `OffsetOldest` beyond what Kafka retention keeps, and `OffsetOldest` is only relevant for a *new* consumer group that hasn't committed offsets yet — a pre-existing group that was briefly down just resumes from its last committed offset on restart, which is on-topic here and not something extra to build).
