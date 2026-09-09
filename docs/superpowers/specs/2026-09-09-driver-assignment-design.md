# Driver assignment — design

## Context

README's Order Service section: "Driver Assignment: Implements driver matching algorithm based on location, taxi type, and rating. Manages driver queue and availability. Handles driver acceptance/rejection logic." Driver Service section: "Trip Management: Accept/decline trip requests from Order Service. Update trip status (started, completed)."

Neither **location** nor **rating** exist anywhere in this codebase — `Driver` has no lat/lng field, and the Rating System is its own separate, unimplemented README bullet. A real matching algorithm needs both. Building either is a much bigger feature than "assign a driver," so, same as the flat price table stood in for the real Pricing Engine, this step ships the smallest *honest* version: match on `taxi_type` + availability only, first match wins (whichever document Mongo's atomic claim finds first — no ranking).

## Scope

In scope:
- `driver_service` gains an atomic "claim an available driver of this taxi type" operation and an internal endpoint for it.
- `order_service.CreateOrder` tries to claim a driver right after the order is persisted; if one's found, the order comes back `driver_assigned` with `driver_id` set; if not, it comes back `created` with no driver — a normal, expected outcome (matches "no drivers nearby right now" in a real app), not an error.
- `order_service.CancelOrder` now also accepts `driver_assigned` orders (not just `created`), and releases the claimed driver back to `available` as part of cancelling.
- Incidental fix: `driver_service.FindByStatus`'s Mongo query built an empty filter (`bson.M{}`) and ignored the `status` argument entirely — `GET /internal/drivers?status=X` silently returned every driver regardless of status. One-line fix, directly adjacent to the code this feature touches.

Explicitly out of scope:
- **Location-based matching** — no driver location field exists; adding one, plus geospatial querying, is its own feature.
- **Rating-based matching** — Rating System doesn't exist (README lists it separately, under both User and Driver Service, as "View calculated rating based on last 20 trips" — nothing to compute from yet).
- **Accept/decline** — README frames this as the driver being *offered* a trip and choosing to accept or decline it. That needs a way to notify a specific driver in real time (push, websocket, or at minimum a poll-and-claim endpoint the driver's app calls) — none of that transport exists here. What's built instead is server-side auto-assignment: the system claims a driver on the rider's behalf, atomically, with no offer/response round-trip. This is a materially different (simpler, non-interactive) model, not a partial implementation of accept/decline.
- **Trip status progression past `driver_assigned`** (`in_progress` → `completed`) — "started"/"completed" trip updates are a separate feature (needs a driver-facing endpoint to report them); `driver_assigned` → `cancelled` is the only new transition this step adds. An order that's actually driven to completion has no way to reach `completed` yet, same gap as before this change.
- **Driver queueing / retry if the first claim attempt finds nobody** — the claim is attempted exactly once, synchronously, during `CreateOrder`. No background retry, no "wait for the next driver to go available."

## Claim mechanics (driver_service)

New internal endpoint `POST /internal/drivers/claim`, body `{"taxi_type": "..."}`. Repository method uses Mongo's `FindOneAndUpdate` (filter `{taxi_type, status: "available"}`, update `{$set: {status: "on-trip"}}`, `ReturnDocument: After`) — single-document atomicity is enough here (no multi-document transaction needed) to guarantee two concurrent `CreateOrder`s can never claim the same driver twice. No document found → `404`, which the `order_service` gateway client maps to `(ok=false, err=nil)` — "no driver available" is a normal outcome, not an error, and callers shouldn't have to `errors.Is`-check for it.

Release (on cancel) reuses the *existing* `UpdateStatusByUser` service method — no new business logic needed — behind a new internal route, `PATCH /internal/drivers/:user_id/status`, since the existing `PATCH /profile/status` is JWT-gated (driver updates their own status) and `order_service` calling on a rider's behalf has no JWT to present. Same `/internal/*` "trusted by network position" convention as every other internal endpoint in this repo.

## `order_service` wiring

New `gateway/driver_service/` HTTP client (same shape as the `wallet_service` gateway from the previous step): `ClaimAvailableDriver(ctx, taxiType) (driverID string, ok bool, err error)` and `ReleaseDriver(ctx, userID) error`.

**`CreateOrder`**, after the order is persisted: attempt a claim. Best-effort, unlike the wallet charge — a driver being unavailable (or `driver_service` being briefly unreachable) is not a reason to fail an order that's already been correctly charged. Three outcomes:
- Claimed + the order's driver-assignment DB update succeeds → return the `driver_assigned` order.
- Claimed but the DB update then fails → release the driver back (compensating action, same shape as the existing charge-insert-refund compensation) and return the order as originally created (unassigned); log the failure.
- No driver available, or the claim call itself errors → return the order as `created`/unassigned; log a claim error (not a "no driver" outcome, which isn't an error at all).

**`CancelOrder`**: cancellable statuses become `{created, driver_assigned}`. Order of operations: refund (blocking, as before — money stays the correctness-critical, blocking step) → release the driver if one was assigned (best-effort, logged on failure — an operational detail, not something the rider's response should fail over) → update status to `cancelled`.

## Non-goals

Same posture as the wallet-integration step: no outbox/2PC anywhere in this flow (the claim-then-assign and refund-then-release sequences each have a window where a crash mid-sequence leaves things temporarily inconsistent — a driver claimed but not yet assigned to an order, or released but the order not yet marked cancelled — accepted for the same reasons already documented for the payment flow, not a new gap this step introduces).
