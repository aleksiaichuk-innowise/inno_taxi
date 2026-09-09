# Order Service ↔ Wallet Service payment integration — design

## Context

`2026-08-27-order-service-design.md` (step 1) explicitly deferred two things: "Payment Integration with Wallet Service (which doesn't exist yet)" and "Order status transitions beyond `created`". Both now exist (`wallet_service` is live; `order_service` is in `docker-compose.yaml`). This spec is step 2: wire the two together per `README.md`'s Wallet Service section ("Processes payments via immediate deduction with compensatory rollback for cancellations") and Order Service section ("Payment Integration: Coordinates with Wallet Service for payment processing").

## Scope

In scope:
- `CreateOrder` charges the rider's wallet for the order's price, synchronously, before the order is considered created.
- A new `CancelOrder` RPC — the minimal, necessary counterpart to "compensatory rollback for cancellations": without a way to cancel, a refund path has nothing to trigger it. Only the order's own user can cancel their own order, and only while it's still `created`.
- A flat, deterministic price-per-taxi-type table.

Explicitly out of scope (separate follow-up, same as step 1's own deferrals):
- **Pricing Engine** (distance/time/surge-based cost) — README lists this as its own bullet under Order Service, independent of payment integration. The flat table here is a placeholder just large enough to make the wallet integration meaningful, not an attempt at the real thing.
- **Driver assignment / trip lifecycle** (`driver_assigned` → `in_progress` → `completed`) — still deferred from step 1. `CancelOrder` only needs to reason about `created`, since that's the only status any order can currently be in.
- **`GetOrder` / order history / trip rating** — README's "View order history and status" and "Rate completed trips" are separate features orthogonal to payment wiring; not added here.
- Refund on order *completion* going wrong, driver-initiated cancellation, admin overrides — none of these paths exist yet because the states they'd apply to don't exist yet.

## Charge flow (CreateOrder)

Order IDs are currently DB-generated (`DEFAULT uuidv7()`), which means the ID doesn't exist until after `INSERT`. But the wallet charge needs a `reference_id` up front for idempotency, and charging *after* insert would mean a row exists (in `created` status, implying "paid") before payment is confirmed. So order ID generation moves to the application:

1. Validate taxi type + location (unchanged).
2. Generate the order ID (`google/uuid`'s `NewV7()` — keeps the same time-ordered property the DB default had; the repository now inserts an explicit `id` instead of relying on `DEFAULT uuidv7()`).
3. Look up the flat price for the taxi type.
4. `wallet_service.Charge(userID, price, referenceID=orderID)` — blocking. `ErrInsufficientFunds` (wallet's 402) surfaces as-is; any other wallet-side failure is wrapped and returned. If this fails, no order row is ever written.
5. Insert the order (explicit `id`, `price_minor_units` set — no longer a nullable "not priced yet" column in practice, though the DB column stays nullable since old philosophy - explicit charge-then-insert order means a charge failure means the row is *never* written, so a partial/unpriced row can't happen going forward).
6. If the insert itself fails after a successful charge (DB down, etc.), refund with the same `referenceID` (best-effort, logged on failure) so the rider isn't left charged for a nonexistent order — this is the one place in the flow that's necessarily best-effort, since there's no way to make an HTTP call and a Postgres insert atomic across process boundaries. Same class of trade-off `ARCHITECTURE.md`'s Cons section already accepts for the Kafka publish.
7. Publish `order_created` to Kafka (unchanged, still best-effort/post-commit).

## Cancel flow (CancelOrder, new)

New RPC: `CancelOrder(CancelOrderRequest{order_id}) returns (CancelOrderResponse{order})`, `POST /v1/orders/{order_id}/cancel`, gRPC-auth-gated same as `CreateOrder` (user ID comes from the JWT, not the request body — an order's owner is never a client-supplied value).

1. Fetch the order by ID.
2. Not found, **or** found but `order.UserID != callerUserID` → the same `ErrOrderNotFound` either way (a 403-shaped "you can't touch this" response would confirm the order ID exists for someone else; collapsing the two avoids that, cheaply).
3. `order.Status != created` → `ErrOrderNotCancellable` (covers "already cancelled" and — once driver assignment exists — "already past the point of no return"; today `created` is the only other reachable status besides `cancelled`, so this check is simple but already correct for later).
4. Refund the wallet — **before** flipping the status, mirroring the charge flow's ordering: the financial operation gates the state transition, not the other way around. `wallet_service.Refund(userID, order.PriceMinorUnits, referenceID=order.ID)` is blocking; a failure here means `CancelOrder` fails and the order stays `created` (nothing lost, nothing double-refunded — `Refund` is idempotent by `reference_id`, so retrying `CancelOrder` after a transient wallet failure is always safe).
5. Update the order's status to `cancelled` and return it.

Gateway (`gateway_service`): `POST /orders/{order_id}/cancel` → rewritten to `/v1/orders/{order_id}/cancel`, same `auth_request`/`X-User-Id` pattern as the other user-facing locations.

## Wallet gateway (new, `order_service/gateway/wallet_service/`)

Plain HTTP client, same shape as `auth_service/gateway/user_service` (the existing service-to-service HTTP client pattern in this repo — no need to invent a second one): `Charge`/`Refund` POST to `wallet_service`'s existing `/internal/wallets/{user_id}/charge` and `/internal/wallets/{user_id}/refund`. `402 Payment Required` maps to `order_service`'s own `errorsx.ErrInsufficientFunds`; everything else 4xx/5xx wraps into a generic error. New config: `WalletService.BaseURL` (env `WALLET_SERVICE_BASE_URL`, default `http://localhost:8084`), mirroring `auth_service`'s `UserServiceConfig`.

## Transport error mapping note

gRPC has no status code that means "payment required" the way HTTP's 402 does. `ErrInsufficientFunds` and `ErrOrderNotCancellable` both map to `codes.FailedPrecondition` (gRPC-Gateway's default mapping puts that at HTTP 400); `ErrOrderNotFound` maps to `codes.NotFound` (404). This is a real difference from `wallet_service`'s own plain-REST `402`/`422` responses for the same underlying conditions — an accepted consequence of the two services using different transports, not something to paper over.

## Non-goals

- No outbox/2PC between the wallet charge and the order insert (or the refund and the status update) — same accepted gap as the existing Kafka publish, not a new one.
- No idempotency key exposed to the *client* for `CreateOrder`/`CancelOrder` — retries at the gRPC/HTTP layer are the caller's problem, same as today's `CreateOrder`. The idempotency that exists (`reference_id` = order ID) protects the wallet from *this service's own* internal retries (e.g. the insert-failure refund), not from a client double-submitting a request.
