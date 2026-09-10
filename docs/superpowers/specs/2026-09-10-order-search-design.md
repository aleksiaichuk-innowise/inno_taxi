# Order search & indexing (Elasticsearch) — design

## Context

`order_service` has held an Elasticsearch client connection since the
docker-compose/ES-container work (`app/db/elastic`, wired in `app/run.go`),
but nothing indexes into it or queries it — `CLAUDE.md` calls this out as
"connection only, no indices/queries yet." `README.md`'s Order Service
requirements include:

> **Search & Analytics**: Provides search functionality for users with
> "Analyst" role.

This design wires that connection up: orders get indexed into ES as they're
created and as their status changes, and a new `SearchOrders` endpoint lets
an Analyst filter across all orders (not scoped to a single rider/driver,
unlike `ListOrders`).

Orders carry no free text (no addresses, no rider-facing comments beyond the
optional trip-rating comment, which isn't a search requirement here), so
this is a **structured filter**, not full-text search. Scope, per the
test-assignment constraint this project is being built under: filter by
`taxi_type`, `status`, a `created_at` date range, and a `price_minor_units`
range. No `user_id`/`driver_id` filter (out of scope — decided during
brainstorming), no fuzzy/full-text matching, no new infrastructure beyond
the ES connection that already exists.

## Non-goals

- Full-text search of any kind.
- Filtering by `user_id`/`driver_id` (an analyst tool, not an ownership
  lookup — `ListOrders` already covers the caller-scoped case).
- Geo queries over `start`/`destination` (they're indexed as plain `double`
  fields for document fidelity, not `geo_point` — nothing here searches by
  location).
- Async/Kafka-driven indexing (see "Alternatives considered").
- Backfilling existing Postgres rows into the index (none exist in this
  environment yet outside manual testing; if it's ever needed it's a
  one-off script, not part of this design).

## Alternatives considered

1. **Synchronous best-effort indexing after each Postgres write (chosen).**
   `order_service` itself calls Elasticsearch right after each successful
   state-changing write, the same way it already calls
   `OrderGateway.PublishOrderCreated`/`PublishOrderRated` right after a
   Postgres write — best-effort, logged on failure, never fails the RPC.
   No new infrastructure, no new Kafka event types, immediate consistency
   for search (no lag window).
2. **Kafka-driven indexing.** A consumer (in `order_service` or a new
   component) indexes off `order_created`/`order_rated` and new events that
   don't exist yet for `driver_assigned`/`cancelled`/`in_progress`/
   `completed`. Rejected: needs three new Kafka event types for a
   single-service concern, adds an eventual-consistency window right after
   a status change (the moment an analyst is most likely to be looking),
   and is more moving parts than a same-process HTTP call to ES.
3. **No Elasticsearch — query Postgres directly with dynamic SQL.** Least
   new code, but leaves the ES connection permanently dead and doesn't use
   the dependency `ARCHITECTURE.md` already documents for exactly this
   purpose. Rejected.

## Data flow: indexing

New package `repository/es_repo` (mirrors `repository/pg_repo`), exposing
one interface at the service boundary (`service/order_service.go`, next to
`OrderRepository`/`OrderGateway`/`WalletGateway`/`DriverGateway`):

```go
type SearchRepository interface {
	IndexOrder(ctx context.Context, order service_dto.Order) error
	SearchOrders(ctx context.Context, filter service_dto.OrderSearchFilter) ([]service_dto.Order, int64, error)
}
```

`OrderService` gets a new `search SearchRepository` field and constructor
parameter.

**Index:** `orders` (configurable via `ES_ORDER_INDEX`, default `orders`).
Created at startup if missing (`es_repo.EnsureIndex`, idempotent — checks
existence first, creates with an explicit mapping otherwise; not an error if
it already exists). Document = the full `service_dto.Order` snapshot,
one-to-one with the row `ListOrders`/`GetOrder` already return, keyed by
order ID (an ES upsert — `PUT orders/_doc/{id}` — so every reindex of the
same order overwrites cleanly, no separate insert/update path needed).
Mapping:

| Field | ES type | Note |
|---|---|---|
| `id`, `user_id`, `driver_id` | `keyword` | exact match / not used in filters today but kept for fidelity |
| `taxi_type`, `status` | `keyword` | filtered on |
| `price_minor_units` | `long` | range-filtered |
| `rating` | `integer` | stored, not filtered |
| `comment` | `text` | stored, not filtered |
| `start_lat`, `start_lng`, `destination_lat`, `destination_lng` | `double` | stored, not filtered (see Non-goals) |
| `created_at`, `updated_at` | `date` | `created_at` range-filtered |

**Write points** — after each successful Postgres write that changes order
state, `order_service` calls `s.search.IndexOrder(ctx, order)` with the
just-persisted order, **best-effort**: log on failure via `slog.ErrorContext`,
never fail the RPC. This mirrors the existing
`PublishOrderCreated`/`PublishOrderRated` best-effort-after-persist
convention exactly:

- `CreateOrder` (`service/create_order.go`), after `s.repo.CreateOrder`.
- `tryAssignDriver` (`service/create_order.go`), after `s.repo.AssignDriver`
  succeeds.
- `CancelOrder` (`service/cancel_order.go`), after `s.repo.UpdateOrderStatus`.
- `StartTrip` (`service/start_trip.go`), after `s.repo.UpdateOrderStatus`.
- `CompleteTrip` (`service/complete_trip.go`), after
  `s.repo.UpdateOrderStatus`.
- `RateTrip` (`service/rate_trip.go`), after `s.repo.RateOrder`.

## Data flow: search

New proto messages/RPC in `shared/proto/order_service/order_service.proto`:

```proto
message SearchOrdersRequest {
  TaxiType taxi_type = 1;              // TAXI_TYPE_UNSPECIFIED = no filter
  Status status = 2;                   // STATUS_UNSPECIFIED = no filter
  string created_after = 3;            // RFC3339, empty = unbounded
  string created_before = 4;           // RFC3339, empty = unbounded
  int64 min_price_minor_units = 5;     // 0 = unbounded
  int64 max_price_minor_units = 6;     // 0 = unbounded
  int32 limit = 7;
  int32 offset = 8;
}

message SearchOrdersResponse {
  repeated Order orders = 1;
  int64 total = 2;
}
```

bound to `rpc SearchOrders(SearchOrdersRequest) returns (SearchOrdersResponse)`
with `option (google.api.http) = { get: "/v1/orders/search" }`.

`0`/empty/`_UNSPECIFIED` as "no filter" mirrors this file's existing
convention (`ListOrders`' `limit <= 0` already means "use the default" in
`service/list_orders.go`) — no proto3 `optional` fields, which nothing in
this codebase currently uses.

`service_dto.OrderSearchFilter` holds `CreatedAfter`/`CreatedBefore` as
`time.Time` (zero value = unset), not raw strings — parsing the wire-format
RFC3339 string is a conversion, not a business-rule validation, so it
belongs in `handler/grpc/mapping.go` next to `locationFromProto`/
`taxiTypeFromProto`, not in the service. Unlike those two conversions
(which are infallible — a proto field extraction always produces *some*
`Location`/`TaxiType` value), a malformed date string is a genuine parse
failure, so `handler/grpc/server.go`'s `SearchOrders` method parses both
fields itself and returns `codes.InvalidArgument` directly on failure —
the same way it already returns `InvalidArgument` directly for an empty
`order_id` on every other endpoint, without a dedicated `errorsx` sentinel
for that either.

`service.SearchOrders(ctx, filter service_dto.OrderSearchFilter)` clamps
`limit`/`offset` exactly like `ListOrders` (default 20, cap 100, negative
offset → 0) and delegates to `s.search.SearchOrders`. **No ownership
scoping and no role check** — an analyst sees every order, and role gating
is `gateway_service`'s job only, the same convention `analytic_service`'s
handlers already follow (zero role/identity logic server-side).

`handler/grpc/server.go`'s `SearchOrders` method does not call
`interceptor.UserIDFromContext` — nothing here needs the caller's identity.
The service still requires a valid JWT to reach this RPC at all (the global
`AuthInterceptor` in `app/run.go` applies to every RPC), it just doesn't use
its contents.

Elasticsearch query: a `bool` query with `term` clauses for `taxi_type`/
`status` (only included when the request sets them) and `range` clauses for
`created_at` and `price_minor_units` (only included when their bounds are
set), sorted `created_at: desc` — same order as `ListOrdersByUser`.

## Gateway routing

`services/gateway_service/nginx.conf` gets one new location, placed near the
other `order_service` blocks:

```nginx
# -- order_service: /orders/search (Analyst only) ---------------------
location = /orders/search {
    set $order_service "order_service:8080";
    auth_request /_gateway/validate/analyst;
    auth_request_set $auth_user_id $upstream_http_x_user_id;
    proxy_set_header X-User-Id $auth_user_id;

    proxy_pass http://$order_service/v1/orders/search$is_args$args;
}
```

`= /orders/search` is an **exact-match** location, so nginx selects it ahead
of the existing `location ~ ^/orders/(?<order_id>[^/]+)$` regardless of
which appears first in the file — exact matches are always checked first.
Without this, `/orders/search` would otherwise be swallowed by that regex
route with `search` misread as an `order_id`. `$is_args$args` is required
for the same reason it was added to `/orders` — a literal `proxy_pass`
rewrite URI doesn't forward the query string on its own, and every filter
here rides in the query string.

## Config

`config/config.go`: new `EsIndex string` field, env `ES_ORDER_INDEX`,
default `orders`.

`app/run.go`: after `elastic.NewESClient`, call `es_repo.EnsureIndex(ctx, es,
cfg.EsIndex)`, then `search := es_repo.NewSearchRepository(es, cfg.EsIndex)`,
passed into `service.NewOrderService(repo, kafkaGateway, walletGateway,
driverGateway, search)`.

`docker-compose.yaml`: no change needed — `ES_ORDER_HOST` is already set for
`order_service`; `ES_ORDER_INDEX` only needs setting if the default
(`orders`) isn't wanted.

## Error handling

- `IndexOrder` failures: logged, never propagated — matches every other
  best-effort call in this codebase (Kafka publish, driver release).
- `EnsureIndex` failure at startup: propagated, fails service startup — same
  treatment as the existing `NewESClient` connectivity check right above it
  in `app/run.go` (an ES that's unreachable at boot is already fatal here,
  this doesn't change that).
- `SearchOrders` with an invalid `created_after`/`created_before`: caught in
  `handler/grpc/server.go` at parse time, returned as `codes.InvalidArgument`
  directly (no `errorsx` sentinel — see "Data flow: search").
- Elasticsearch itself unreachable during a search: `SearchOrders` returns
  an error, mapped to `codes.Internal` — there's no meaningful degraded
  result to fall back to for a search request (unlike indexing, where the
  order is already safely in Postgres regardless).

## Testing

- `repository/es_repo`: unit tests via `httptest.Server` standing in for
  Elasticsearch — assert the request body sent for `IndexOrder` and for
  `SearchOrders`' query DSL (each filter field present/absent), and that a
  response is parsed into the right `[]service_dto.Order`/`total`. Mirrors
  how `gateway/driver_service`'s and `gateway/wallet_service`'s tests are
  already written in this service. No integration/dockertest suite — none
  exists for `order_service` yet (per `CLAUDE.md`), and this design doesn't
  introduce one.
- `service/search_orders_test.go`: fake `SearchRepository` (added to the
  shared fakes in `service/create_order_test.go`, same convention as the
  other fakes there) — limit/offset clamping, filter pass-through.
- Each of the six write points (`create_order_test.go`,
  `cancel_order_test.go`, `start_trip_test.go`, `complete_trip_test.go`,
  `rate_trip_test.go`): one assertion that `IndexOrder` is called with the
  persisted order, and one that an `IndexOrder` error doesn't fail the RPC —
  mirroring the existing "publish failure doesn't fail the RPC" tests
  already present for `PublishOrderCreated`/`PublishOrderRated`.
