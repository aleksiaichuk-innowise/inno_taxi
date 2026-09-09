# Analytic Service — Design

## Context

The last of the seven planned services. `order_service` publishes exactly
one real Kafka event today (`order_created`, per its own step-1 spec) and
nothing else in the repo publishes to Kafka at all — `user_service` has no
producer, and there is no rating system anywhere. Analytic Service is a
pure Kafka consumer + read API, so its step 1 is scoped to the one event
that actually exists, same discipline every other service's spec has
followed.

This is also the first service to actually need a running Kafka broker.
`order_service`'s own design doc specified `kafka` in `docker-compose.yaml`
but explicitly deferred adding it ("a pre-existing gap, not this spec's
concern"), and nothing since has needed it live. This spec is where Kafka
(and, for symmetry, ClickHouse) finally join `docker-compose.yaml` — not
because Analytic Service's own scope demands the whole event-driven
architecture, just because a Kafka consumer with no broker to connect to
can't be demonstrated at all.

## Scope (from `README.md`'s Analytic Service section)

In scope:
- **Data collection**: consume `order_created` from Kafka, write one row
  per event to ClickHouse.
- **Statistics**: `GET /analytics/orders/stats` — total order count, plus
  counts broken down by status and by taxi type, over a date range.
- **Real-time dashboards**: `GET /analytics/orders/daily` — per-day order
  counts over a date range, the shape a dashboard's time-series chart
  would consume.
- **Authentication**: gated at `gateway_service` by the `analyst` role,
  the same `auth_request`/`X-Required-Role` mechanism already built for
  `user_service`'s admin-only route.
- Infra: Kafka (single-broker, KRaft) and ClickHouse join
  `docker-compose.yaml`.

Explicitly out of scope for this step:
- **Consuming `user_registered` or rating events.** Neither producer
  exists (`user_service` publishes no Kafka events at all; there is no
  rating feature anywhere in the app yet). Nothing to consume.
- **"User behavior analytics" and "Ratings Analytics."** Both are
  functional-requirement bullets in README, but both need data sources
  (`user_registered`, ratings) that don't exist yet — same reasoning as
  above.
- **"Reports" for stakeholders.** README names this as a capability but
  gives no concrete shape (format, cadence, recipients) to build against
  — there's nothing to scope narrowly here yet, unlike "Statistics" and
  "Real-time Dashboards," which map onto concrete, buildable endpoints.
- **Self-validating JWTs the way `user_service`/`driver_service`/
  `order_service` do.** See "Decisions" below — a deliberate gap, not an
  oversight.

## Decisions made during design (with rationale)

- **Fiber, not Gin.** README pins "HTTP Framework: Fiber for high-
  performance analytics API" specifically for this service — a
  deliberate per-service choice, same category as `order_service`'s
  proto/gRPC-Gateway pattern or `wallet_service`'s `sqlx`. This is the
  first Fiber-based service in the repo.
- **No self-validated JWT/role check inside Analytic Service itself** —
  every other service so far (`user_service`, `driver_service`,
  `order_service`) validates its own JWT even though `gateway_service`
  also gates it, as defense in depth. Analytic Service skips this
  deliberately: both of its endpoints are read-only aggregate queries
  (low blast radius compared to, say, `driver_service`'s status-mutating
  routes), and `shared/transport/http/middleware` is Gin-specific —
  standing up a parallel Fiber JWT middleware for two GET endpoints isn't
  proportionate to what it buys. Flagged here as an intentional trade-off
  to revisit if Analytic Service ever grows a mutating or more sensitive
  endpoint, not silently left out.
- **`ReplacingMergeTree`, not plain `MergeTree`, keyed on `order_id`.**
  Kafka's at-least-once delivery means the consumer group can redeliver
  a message after a restart before its offset commit lands. ClickHouse's
  usual answer to "the same row might arrive twice" is `ReplacingMergeTree`
  — duplicate `order_id` rows collapse into one during background merges.
  **Known trade-off**: that collapsing is eventual, not immediate, so a
  query right after a redelivery could still double-count briefly. Both
  read endpoints query with `FINAL` (forces dedup at query time) to make
  the numbers correct now rather than "eventually correct" — acceptable
  cost at this data volume; revisit with pre-aggregated materialized
  views if `FINAL` ever becomes a bottleneck.
- **No goose migrations for ClickHouse.** `order_service` and
  `wallet_service` both use goose-formatted SQL files for Postgres, but
  neither this repo nor goose itself has an established ClickHouse
  convention to extend — and ClickHouse's own DDL culture leans on
  idempotent `CREATE TABLE IF NOT EXISTS` rather than versioned
  up/down migrations. `app/db/clickhouse` embeds the one schema file
  (`go:embed`) and runs it on startup — one source of truth, no drift
  between a checked-in `.sql` file and a hand-duplicated Go string, and
  no new migration tooling invented for one table.
- **Kafka: single-broker, KRaft, `apache/kafka` image.** Matches
  `order_service`'s own deployment section ("single-broker, KRaft mode —
  simpler for local dev, no separate Zookeeper container"). The official
  Apache image needs less docker-compose-specific env-var tuning for a
  single-node KRaft setup than the common community alternatives.
- **Consumer group id `analytic_service`, auto-commit.** A named,
  stable group means a restart resumes from the last committed offset
  instead of reprocessing the whole topic; auto-commit is fine precisely
  because `ReplacingMergeTree` already absorbs the redelivery this
  implies.

## Data model

```go
// entity/service/order_event.go
type OrderEvent struct {
    OrderID   string
    UserID    string
    TaxiType  string
    Status    string
    CreatedAt time.Time
}

// entity/service/stats.go
type OrderStats struct {
    TotalOrders      int64
    CountsByStatus   map[string]int64
    CountsByTaxiType map[string]int64
}

type DailyOrderCount struct {
    Date  string // YYYY-MM-DD
    Count int64
}
```

ClickHouse table (`app/db/clickhouse/schema/create_order_events.sql`,
co-located with the code that `go:embed`s it — `go:embed` patterns can't
traverse into a parent directory, so this can't live in a separate
top-level `migrations/` folder the way `order_service`/`wallet_service`'s
goose files do):

```sql
CREATE TABLE IF NOT EXISTS order_events (
    order_id String,
    user_id String,
    taxi_type LowCardinality(String),
    status LowCardinality(String),
    order_created_at DateTime64(3),
    ingested_at DateTime64(3) DEFAULT now64(3)
) ENGINE = ReplacingMergeTree(ingested_at)
ORDER BY (order_id);
```

## Architecture

```
services/analytic_service/
  go.mod                          — require .../shared, replace ../../shared
  cmd/main.go
  config/config.go                — Kafka, ClickHouse, HTTP config

  app/
    run.go                        — wires deps, starts the Kafka consumer
                                     group (background goroutine) + Fiber
                                     HTTP server, graceful shutdown
    db/
      clickhouse/
        clickhouse.go              — client + go:embed schema, runs it on connect
        schema/create_order_events.sql
    kafka/
      consumer.go                  — sarama.NewConsumerGroup setup

  entity/service/
    order_event.go                — OrderEvent (no tags)
    stats.go                       — OrderStats, DailyOrderCount

  handler/
    kafka/
      order_created_consumer.go   — sarama.ConsumerGroupHandler: unmarshal
                                     proto OrderCreatedEvent, map to
                                     OrderEvent, call service
    http/
      http.go                     — Handler struct + constructor (Fiber)
      order_stats.go               — GET /orders/stats
      order_daily.go                — GET /orders/daily

  service/
    service.go                    — OrderEventRepository interface,
                                     AnalyticService struct, New()
    ingest_order_event.go
    get_order_stats.go
    get_daily_order_counts.go

  repository/
    ch_repo/
      ch_repo.go                  — ClickHouseRepository struct
      insert_order_event.go
      get_order_stats.go
      get_daily_order_counts.go

  errorsx/errors.go                — ErrInvalidDateRange
```

`service.OrderEventRepository`:

```go
type OrderEventRepository interface {
    InsertOrderEvent(ctx context.Context, evt service_dto.OrderEvent) error
    GetOrderStats(ctx context.Context, from, to time.Time) (service_dto.OrderStats, error)
    GetDailyOrderCounts(ctx context.Context, from, to time.Time) ([]service_dto.DailyOrderCount, error)
}
```

## Endpoints (as exposed through `gateway_service`)

| Method | Path | Auth | Query params | Response |
|---|---|---|---|---|
| `GET` | `/analytics/orders/stats` | `analyst` role | `from`, `to` (YYYY-MM-DD, default last 30 days) | `200 {total_orders, counts_by_status, counts_by_taxi_type}` |
| `GET` | `/analytics/orders/daily` | `analyst` role | `from`, `to` | `200 [{date, count}, ...]` |

`gateway_service/nginx.conf` gets a third role-gated internal location
(`/_gateway/validate/analyst`, mirroring the existing `/admin` one) and a
`/analytics/` route stripping that prefix before proxying, same pattern as
every other routed service.

## Testing

Service-layer unit tests with a fake `OrderEventRepository`, same
convention as every other service. `repository/ch_repo` gets a
`//go:build integration` test against a real ClickHouse container
(`ory/dockertest`, the same tool used for `user_service`'s Mongo suite and
`wallet_service`'s Postgres suite) — this is the one thing a fake
structurally can't prove: that `ReplacingMergeTree` + `FINAL` actually
dedups a redelivered event instead of double-counting it.

## Deployment

`docker-compose.yaml` gets:
- `kafka` — `apache/kafka`, KRaft mode, single broker.
- `clickhouse` — official `clickhouse/clickhouse-server` image, own
  database (per `ARCHITECTURE.md`: each service owns its data).
- `analytic_service` — depends on both, plus `gateway_service`'s
  `nginx.conf` gets the new route.

`order_service` itself still isn't added to `docker-compose.yaml` (still
needs its own Postgres/Elasticsearch wiring, `order_service`'s own
unfinished business, not this spec's). Live verification that the
consumer actually works publishes a test `OrderCreatedEvent` directly
onto the `order_created` topic using `shared/proto/order_service`'s
existing generated types, without needing a running `order_service`.

## Explicitly not decided here (deferred to later work)

- Wiring `order_service` into `docker-compose.yaml` (its own follow-up;
  once it's there, this topic actually gets produced to for real instead
  of by a test script).
- The `user_registered` event and its consumers, `driver_service`'s and
  Analytic Service's alike (flagged as ready-to-spec in
  [[2026-09-09-wallet-service-design]] already).
- A rating feature anywhere in the app, and therefore Ratings Analytics.
- The "Reports" functional requirement, once it has a concrete shape.
