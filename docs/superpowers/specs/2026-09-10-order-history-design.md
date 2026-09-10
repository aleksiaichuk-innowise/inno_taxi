# Order history for the rider — design

## Context

Per `docs/PROJECT-STATUS.md`, "View order history and status" (Order Service) was the only ❌ left with no dependency on unbuilt infrastructure (unlike Pricing Engine, geo-matching, or Elasticsearch search, which each need something that doesn't exist yet). `order_service` has stored every field a history view needs since the payment and rating steps landed - it just never exposed a way to read more than one order back.

## Scope

In scope:
- `GetOrder`: fetch a single order by ID.
- `ListOrders`: the caller's own order history, paginated, most recent first.

Both are read paths, gated by the same ownership shape every mutating endpoint here already uses (JWT-derived caller, not a request field) - but broadened to **either side of the trip**: the rider who placed the order, or the driver who was assigned to it. Every existing write (`CancelOrder`, `StartTrip`, `CompleteTrip`, `RateTrip`) is one-sided (only the rider, or only the driver, can act), because each of those is a specific action naturally owned by one party. Viewing is different - both parties have an obvious, legitimate reason to see the same order (a driver needs their own trip history exactly as much as a rider does), and README's Driver Service section separately implies drivers need visibility into their own trip activity. One `GetOrder`/`ListOrders` pair serving both, rather than a second rider-only endpoint plus a speculative driver-only one nobody asked for concretely, is the smaller surface for the same coverage.

Explicitly out of scope:
- **Search/filtering** (by date range, status, taxi type) - README lists "Search & Analytics" as its own bullet, tied to Elasticsearch, which is a separate, larger, not-yet-built piece (ES today is a connection only, no indices). `ListOrders` here is a plain chronological feed, not a search.
- **Rating/comment visible to the *driver* being rated in this same call** - the returned `Order` already carries `rating`/`comment` (added for `RateTrip`), so a driver listing their own trips sees how they were rated as a side effect of the existing field, not new work.
- **Cursor/keyset pagination.** Plain `limit`/`offset`, capped, is the whole pagination story here - nothing else in this repo has established a pagination convention to be consistent with, and keyset pagination's only real advantage (stable results under concurrent inserts at the boundary) isn't worth the added complexity for a personal order-history feed at this scale. `offset`-based pagination's well-known correctness gap (a row inserted ahead of the window shifts everything by one) is an accepted, ordinary trade-off here, not a new one this step introduces.

## Ownership check

`GetOrder`: caller must be `order.UserID` **or** `order.DriverID` (if assigned) - anyone else gets `ErrOrderNotFound`, same non-disclosure reasoning as every other order endpoint (a distinct "forbidden" response would confirm the ID belongs to someone).

`ListOrders`: no such check needed by construction - the query itself is scoped to `WHERE user_id = @caller OR driver_id = @caller`, so there's nothing to leak; a caller only ever sees rows they already have a legitimate side in.

## Pagination

`limit` defaults to 20, clamped to [1, 100] server-side (a caller passing 0 or a huge number gets the default/cap silently, not an error - this is a convenience default, not a contract worth failing requests over). `offset` defaults to 0, clamped to ≥ 0. Response includes `total` (a plain `COUNT(*)` over the same `WHERE` clause) so a client can tell whether there's more without an extra round-trip.

## Routing

`GetOrder`: `GET /v1/orders/{order_id}` - new `gateway_service` location (regex, no trailing action segment, so it doesn't collide with `/cancel`, `/start`, `/complete`, `/rate`).

`ListOrders`: `GET /v1/orders` - reuses the *existing* `location = /orders` block used for `CreateOrder`'s `POST` today. No `gateway_service` change needed: nginx location matching in this repo has never been method-aware (the same block already just proxies whatever method the client used to `order_service`, which is where grpc-gateway itself dispatches `POST` → `CreateOrder` vs `GET` → `ListOrders` on the same path).

## Non-goals

Same posture as every step in this chain: no new outbox/consistency mechanism - this step only adds reads.
