# Trip lifecycle: start / complete — design

## Context

Driver assignment (previous step) gets an order to `driver_assigned` but nothing ever moves it further — `in_progress` and `completed` are defined in the `Status` enum but unreachable. README's Driver Service section: "Trip Management: ... Update trip status (started, completed)." That's the piece this step adds.

Driver assignment was deliberately server-side auto-claim, not an accept/decline offer (no notification transport exists to offer a driver anything). Trip progress reporting doesn't have that problem — it's the driver's own app telling the system "I started" / "I finished," an action *they* take, not something pushed to them. So this fits the same "driver calls an endpoint" shape `CancelOrder` already established for riders, just gated to the *assigned driver* instead of the order's owner.

## Scope

In scope:
- Two new `order_service` RPCs, gated to the order's assigned driver (not the rider):
  - `StartTrip`: `driver_assigned` → `in_progress`.
  - `CompleteTrip`: `in_progress` → `completed`, and releases the driver back to `available` (the actual, real end of the "driver comes back available" loop — cancellation releasing early was a special case of this, not the main path).
- Gateway routes `POST /orders/{order_id}/start` and `POST /orders/{order_id}/complete`, same `auth_request` shape as `/orders/{order_id}/cancel`.

Explicitly out of scope:
- **Driver payout / trip settlement** — README doesn't ask for a *second* money movement on completion (the rider was already charged at creation); nothing here changes wallet balances beyond the driver-release side effect.
- **Cancelling an `in_progress` trip** — `CancelOrder`'s cancellable set stays `{created, driver_assigned}`; a trip that's already started isn't cancelled the same way a request is (matches real taxi apps: once you're in the car, "cancel" isn't really cancel anymore, it'd be a different, unbuilt feature like a fare dispute).
- **Ratings on completion** — README's Rating System is still its own separate, unimplemented feature; `CompleteTrip` doesn't ask for or record one.
- **GPS/location tracking during the trip** — no location data exists anywhere in this codebase (same gap noted in the driver-assignment spec).

## Ownership and error shape

Mirrors `CancelOrder`'s existing pattern exactly, just checking the *driver* instead of the rider: caller's user ID comes from the JWT (gRPC context), not a request field. "Order doesn't exist" and "order exists but isn't assigned to you" collapse to the same `ErrOrderNotFound` — same reasoning as before, a distinct response would confirm someone else's order ID.

Wrong-state attempts get their own errors rather than reusing `ErrOrderNotCancellable`, since they're a different condition with a different fix (nothing to retry until the trip reaches the right state, not "this can never be cancelled now"): `ErrOrderNotStartable` (not `driver_assigned`), `ErrOrderNotCompletable` (not `in_progress`). Both map to gRPC `FailedPrecondition` (HTTP 400 via gRPC-Gateway), same transport-mismatch trade-off already accepted for `ErrOrderNotCancellable`/`ErrInsufficientFunds`.

## Non-goals

Same posture as every prior step in this chain: no outbox/2PC between the driver-release call and the status-update write in `CompleteTrip` — best-effort release, logged on failure, exactly like `CancelOrder`'s existing release-on-cancel.
