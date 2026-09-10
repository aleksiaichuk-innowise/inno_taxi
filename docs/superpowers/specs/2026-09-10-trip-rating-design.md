# Trip ratings — design

## Context

Per `docs/PROJECT-STATUS.md`, ratings are the single biggest gap touching three services at once: Order Service ("Rate completed trips"), User Service ("Rating System: View calculated rating based on last 20 trips. For both users and drivers."), and Analytic Service ("Data Collection: ... ratings", "Ratings Analytics: Aggregated ratings for drivers"). This step builds the part that's tractable without new infrastructure: the rating itself, and its analytics pipeline. It follows directly from the trip-lifecycle step — a trip can only be rated once it's `completed`.

## Scope

In scope:
- A new `order_service` RPC, `RateTrip`: the rider rates the driver on the just-completed trip (1–5, optional comment). One rating per order, enforced at the DB level.
- `order_service` publishes an `order_rated` Kafka event (mirrors `order_created`'s existing shape).
- `analytic_service` consumes it into a new ClickHouse table and exposes an aggregation endpoint: a driver's average rating and rating count over their last 20 rated trips (Analyst-gated, same as the service's other endpoints) — this is the "Ratings Analytics" bullet.

Explicitly out of scope:
- **Driver rates rider.** README's "for both users and drivers" implies both directions exist, but nothing in the current codebase establishes what a *user's* rating would even mean operationally (drivers don't have a symmetric "was this rider good" workflow anywhere in the spec beyond that one clause), and building it doubles every piece of this step for a direction with no other spec detail behind it. Riders rating drivers is the direction actually exercised by "Rate completed trips" under Order Service and is the one every other taxi-app requirement in this README assumes (driver availability, driver matching "based on ... rating"). One direction, done properly, beats two done by half.
- **Surfacing the computed rating on the driver's own profile** (`driver_service`'s `GET /profile`, or `user_service`'s). That's a `driver_service` → `analytic_service` call, a dependency direction that doesn't exist anywhere else in this codebase (every existing cross-service call in this system flows toward wallet/driver/analytic, never *from* driver/analytic outward to a peer). It's a real, separate follow-up - profile enrichment - not part of "the rating system" itself, which is: capturing a rating and being able to aggregate it. The aggregation endpoint this step adds is exactly where that follow-up would read from.
- **Using rating in driver matching.** `CreateOrder`'s matching is still taxi_type + availability only (see the driver-assignment spec) - this step makes rating data exist and queryable, it doesn't change matching to consume it.
- **Editing or deleting a rating** once submitted.

## RateTrip

`RateTrip(RateTripRequest{order_id, rating, comment}) returns (RateTripResponse{order})`, `POST /v1/orders/{order_id}/rate`. Ownership check mirrors `CancelOrder`'s existing shape exactly: caller ID from the JWT, `order.UserID != callerID` → `ErrOrderNotFound` (not found and not-yours collapse, same reasoning as every other order endpoint in this codebase). `order.Status != completed` → `ErrOrderNotRatable`. `rating` outside 1–5 → `ErrInvalidRating`.

Already-rated is checked twice, not once, because "check then write" always has a race window between two requests for the same order:
1. Service layer checks `order.Rating != nil` after the fetch (fast path, no DB round-trip needed for the common case).
2. The `UPDATE ... WHERE id = @id AND rating IS NULL` guard at the repository layer is what's actually load-bearing - if the row wasn't already rated when this statement ran, it will affect exactly one row; if it was (a concurrent second `RateTrip` call won the race), it affects zero, and that's how a genuine double-submit gets caught instead of silently overwriting the first rating.

Publishing the `order_rated` event is best-effort/post-commit, the same trade-off already accepted for `order_created` - a Kafka hiccup doesn't undo an already-recorded rating.

## Analytics ingestion

New table `driver_ratings` (mirrors `user_registration_events`'s shape: `ReplacingMergeTree` keyed for at-least-once dedup, own schema file, second `go:embed`). New consumer group `analytic_service_order_rated` (own group ID, independent of the two consumer groups that already exist, same reasoning as the driver-assignment step's `analytic_service_user_registered`: each topic gets its own group so one's wiring can't affect another's).

New endpoint `GET /analytics/ratings/drivers/{driver_id}` returns `{average, count}` over that driver's most recent 20 ratings (`ORDER BY rated_at DESC LIMIT 20` inside the aggregate, not a plain `AVG()` over the whole table - README specifically says "last 20 trips", not "all trips"). Analyst-gated via the existing `gateway_service` pattern (`/_gateway/validate/analyst`), same as every other `/analytics/*` route.

## Non-goals

Same posture as every step in this chain: no outbox/2PC for the Kafka publish (best-effort, logged on failure); no dead-letter queue for an unparsable `order_rated` message (logged and skipped, same as every other consumer here).
