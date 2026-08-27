# Order Service — Design (Step 1: CreateOrder via proto/gRPC-Gateway/Kafka)

## Context

InnoTaxi is a taxi-booking microservices monorepo (see root `README.md` and
`ARCHITECTURE.md`). `user_service` and `driver_service` both exist and both
use the pragmatic plain-Gin pattern (`entity/http` hand-written DTOs, no
proto, no gRPC) rather than ARCHITECTURE.md's canonical proto-driven
gRPC-Gateway template.

Order Service is the third microservice, and the first to adopt
ARCHITECTURE.md's full canonical pattern: proto as the single source of
truth for HTTP, gRPC, and Kafka contracts, via gRPC-Gateway. This is a
deliberate fork from the established `user_service`/`driver_service`
pattern, not a retrofit of those two services — they stay as they are.

This spec covers **step 1 only**: a single use case, `CreateOrder`, exposed
over gRPC-Gateway (HTTP) and gRPC, publishing an `order.created` Kafka event.
No driver assignment, no pricing, no Elasticsearch indexing/search — those
are separate follow-up specs, same way Driver Service's Kafka step was
split off from its HTTP skeleton step.

## Scope (from `README.md` Order Service section)

In scope for step 1:
- **Order Management**: "Initiate taxi orders through Order Service (User
  only)" — create an order in `created` status.
- Kafka: publish `order.created` after successful creation.
- Infra: stand up Postgres (primary store) and Elasticsearch (connection
  only, no indices/mappings yet) and Kafka (KRaft single-broker) in
  docker-compose.

Explicitly out of scope for step 1 (separate follow-up specs):
- **Driver Assignment** (matching algorithm, driver queue, accept/reject) —
  `README.md`'s Order Service section.
- **Pricing Engine** (cost calculation, surge pricing).
- **Trip History** view / order status view / rating completed trips.
- **Search & Analytics** (Elasticsearch indexing + queries, GraphQL).
- **Payment Integration** with Wallet Service (which doesn't exist yet).
- Order status transitions beyond `created` (no `driver_assigned` /
  `in_progress` / `completed` / `cancelled` logic — the enum is defined in
  full now as the proto contract, but only `created` is reachable).

## Decisions made during brainstorming (with rationale)

- **Postgres, not Mongo, not Elasticsearch-as-primary.** README lists
  Elasticsearch as Order Service's "Database", but ES doesn't offer
  transactions and isn't a good fit for a strict lifecycle state machine
  (`created → driver_assigned → in_progress → completed/cancelled`) or for
  coordinating with Wallet Service's payment deduction/rollback. Postgres
  matches ARCHITECTURE.md's `PgAtomicRepository`/`transactionalOperation`
  pattern, built exactly for atomic multi-step writes. ES stays in scope as
  a secondary search index (README calls it a "Search Engine"), stood up
  now infrastructurally but not written to or queried until a later spec.
- **CreateOrder only, no driver assignment in step 1.** Keeps the first
  slice to one use case, consistent with how Driver Service's first step
  was Register-only. Driver Assignment depends on this existing first, and
  is a large enough feature (matching algorithm, driver queue) to deserve
  its own spec.
- **Kafka `order.created` publish included in step 1**, even though no
  consumer exists yet (Analytic Service isn't built) — the user explicitly
  chose to adopt gRPC+Kafka now rather than defer, so this step exercises
  the full publish path end-to-end rather than standing up unused
  infrastructure.
- **JWT auth via gRPC unary interceptor**, not a Gin middleware (the
  transport is gRPC-Gateway now, so the old HTTP-middleware pattern from
  `user_service`/`driver_service` doesn't apply directly).

## Data model

```go
// entity/service/order.go
type Status string

const (
    StatusCreated        Status = "created"
    StatusDriverAssigned Status = "driver_assigned"
    StatusInProgress     Status = "in_progress"
    StatusCompleted      Status = "completed"
    StatusCancelled      Status = "cancelled"
)

type TaxiType string // economy/comfort/business — mirrors driver_service's enum

type Location struct {
    Lat float64
    Lng float64
}

type Order struct {
    ID          string
    UserID      string
    TaxiType    TaxiType
    Pickup      Location
    Destination Location
    Status      Status
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

The full `Status` enum is defined now, even though step 1 only ever produces
`created` — it's part of the proto contract, and adding enum values later
is a breaking-change-prone proto edit best avoided.

## Proto contract

New package `shared/proto/order_service/`:

- `order_service.proto` — `OrderService` with RPC `CreateOrder
  (CreateOrderRequest) returns (CreateOrderResponse)`, annotated
  `google.api.http = { post: "/v1/orders" body: "*" }`. Messages carry the
  `Status`/`TaxiType` enums and a `Location` message (`lat`/`lng`).
- `kafka.proto` — `OrderCreatedEvent` (`order_id`, `user_id`, `taxi_type`,
  `pickup`, `destination`, `status`, `created_at` as unix millis `int64`).

## Architecture

```
services/order_service/
  go.mod                        — require .../shared, replace ../../shared
  cmd/main.go
  config/config.go              — PG, ES, Kafka, gRPC/HTTP ports, JWT secret

  app/
    run.go                      — wires deps, starts gRPC server + gRPC-Gateway
                                   HTTP server, graceful shutdown (SIGINT/SIGTERM)
    db/
      pg/pg.go                  — Postgres client, runs migrations in New()
      pg/migrations/             — SQL: orders table
      es/es.go                  — ES client (connection only, no indices yet)
    grpc/
      server.go                 — gRPC server setup, UnaryInterceptor(auth),
                                   handler registration
    kafka/
      publisher.go               — Kafka producer setup

  entity/
    service/order.go            — Order, Status, TaxiType, Location (no tags)
    repository/order.go         — Postgres row projection (db tags)
    gateway/order_created.go     — outbound Kafka event DTO

  handler/
    grpc/
      grpc.go                   — GrpcHandler struct + New()
      create_order.go           — CreateOrder RPC handler: validate pbentity
                                   request, map to entity/service, call
                                   service, map result back to pbentity
    http/
      http.go                   — gRPC-Gateway registration only (proto owns
                                   HTTP — no manual routes/DTOs)

  gateway/
    gateway.go                  — Gateway aggregate (Kafka only for now)
    kafka/
      kafka.go                  — KafkaGateway interface + struct
      order_created.go          — publishes "order_created" topic

  service/
    service.go                  — OrderRepository interface, OrderGateway
                                   interface, OrderService struct, New()
    create_order.go              — CreateOrder use case: validate, persist,
                                   publish event (post-commit)

  repository/
    repository.go               — Repository aggregate (Pg only for now)
    pg/
      pg.go                     — PgRepository interface
      create_order.go
```

`shared/transport/grpc/middleware/auth.go` — new, parallel to the existing
`shared/transport/http/middleware/auth.go`. Same JWT verification (HS256,
`sub` = user id, `roles` = `[]string`), but:
- reads the token from **gRPC metadata** (`authorization` key) instead of an
  HTTP header — this is what gRPC-Gateway translates the HTTP `Authorization`
  header into when proxying into gRPC;
- on success, stores `userID`/`roles` in `context.Context` via a typed key +
  `middleware.UserIDFromContext(ctx)` getter, which `create_order.go` uses
  instead of trusting a client-supplied `user_id` field.
- registered once in `app/grpc/server.go` via `grpc.UnaryInterceptor(...)`
  (or `ChainUnaryInterceptor` if more interceptors are added later, e.g.
  logging) — applies to every RPC automatically.

**Must be verified during implementation, not assumed:** gRPC-Gateway does
not forward every HTTP header into gRPC metadata by default — only headers
prefixed `Grpc-Metadata-`, though `Authorization` is commonly special-cased.
Write an explicit end-to-end test (HTTP request with `Authorization` header
→ interceptor receives it) rather than trusting this from memory. If it
isn't forwarded automatically, `handler/http/http.go` must set
`runtime.WithIncomingHeaderMatcher` explicitly when constructing the
`ServeMux`.

## Kafka event

Topic: `order_created` (snake_case, matching the naming already used for
`driver_service`'s not-yet-built `user_registered` consumer in its spec).

`gateway/kafka/order_created.go` marshals `OrderCreatedEvent` (proto) and
publishes after the Postgres insert commits — never inside the DB
transaction, per ARCHITECTURE.md's rule ("Writes to Postgres + publish to
Kafka: Yes for PG writes; Kafka publish goes in `gateway/kafka/` after
commit").

**Known trade-off, not addressed in this step:** if the Postgres write
succeeds but the Kafka publish fails, the order is still considered
successfully created (client gets `200`); the publish failure is only
logged. No outbox pattern / retry / idempotency key — ARCHITECTURE.md
already flags this as Cons #6 ("Kafka handlers lack idempotency contract...
must be added before production use"). Not worth solving now since there is
no real consumer yet (Analytic Service doesn't exist).

## Deployment

`docker-compose.yaml` currently only has `mongo`/`mongo-ui`/`user_service`
(`driver_service` isn't wired in yet either — a pre-existing gap, not this
spec's concern). Add:

- `postgres` — own container, own database, not shared with a future
  Wallet Service (ARCHITECTURE.md: "Each service owns its data").
- `elasticsearch` — single-node, security disabled (local dev only),
  connection wired up, no indices/mappings written yet.
- `kafka` — single-broker, KRaft mode (no separate Zookeeper container —
  simpler for local dev).
- `order_service` — the service itself, HTTP port (gRPC-Gateway) and a
  separate gRPC port, `depends_on` all three infra services with health
  checks.

## Explicitly not decided here (deferred to later specs)

- Driver Assignment algorithm and its transport to `driver_service` (HTTP,
  same as today's `GET /internal/drivers?status=...`, or gRPC now that
  Order Service has the toolchain).
- Elasticsearch index mapping and what triggers indexing (dual-write in
  `CreateOrder`, or a Kafka consumer reading `order_created` back into ES).
- Pricing Engine.
- Whether `driver_service` and `user_service` ever migrate to the
  proto/gRPC pattern, or permanently stay on plain Gin.
