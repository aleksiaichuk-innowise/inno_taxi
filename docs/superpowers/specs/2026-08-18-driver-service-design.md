# Driver Service — Design (Step 1: HTTP skeleton)

## Context

InnoTaxi is a taxi-booking microservices monorepo (see root `README.md` and
`ARCHITECTURE.md`). `user_service` is the only service with real code so far,
built out over this project's earlier sessions: Register, credential
verification (internal HTTP), full profile management (view/update/change
password/soft-delete), Analyst role assignment, JWT auth middleware, unit +
Mongo integration tests, `app/run.go` graceful shutdown, Dockerfile +
docker-compose wiring.

Driver Service is the second microservice. This spec covers **step 1 only**:
an HTTP skeleton with no Kafka. Step 2 (Kafka: `user_registered` producer in
`user_service` + consumer in `driver_service`) is a separate follow-up spec,
scoped once step 1 is built and working.

## Scope (from `README.md` Driver Service section, lines 58-66)

In scope for step 1:
- **Profile Management**: drivers view/update their own profile.
- **Status Management**: toggle availability status (available/on-trip/offline).
- A temporary way to create a driver record without Kafka (see below).

Explicitly out of scope for step 1 (needs Order Service, which doesn't exist
yet — same reasoning as Rating/Wallet being deferred in `user_service`):
- Trip Management (accept/decline trip requests, update trip status) —
  `README.md:63-65`.

Also out of scope for step 1 (deferred to step 2):
- Kafka producer in `user_service` for `user_registered`.
- Kafka consumer in `driver_service`.

## Data model

Only fields the spec actually calls for — no vehicle/license data, since
`README.md` never mentions it for Driver Service:

```go
// entity/service/driver.go
type TaxiType string

const (
    TaxiTypeEconomy  TaxiType = "economy"
    TaxiTypeComfort  TaxiType = "comfort"
    TaxiTypeBusiness TaxiType = "business"
)

type Status string

const (
    StatusAvailable Status = "available"
    StatusOnTrip    Status = "on-trip"
    StatusOffline   Status = "offline"
)

type Driver struct {
    ID        string
    UserID    string    // FK to User Service's user id (role=driver)
    TaxiType  TaxiType
    Status    Status
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

`UserID` is unique (one driver record per user) — enforced via a Mongo unique
index, same pattern as `user_service`'s `uniq_email`/`uniq_phone`.

## Architecture

Mirrors `user_service`'s already-proven Clean Architecture layout
(`ARCHITECTURE.md`) — same layers, same conventions, independent module:

```
services/driver_service/
  go.mod                      — module ".../services/driver_service"
                                 require .../shared, replace ../../shared
  cmd/main.go                 — config.Load() + app.Run(cfg); nothing else
  app/
    run.go                    — wires deps, http.Server, graceful shutdown
                                 (SIGINT/SIGTERM), from the start — not added
                                 later as debt, the way user_service did it
    db/mongo/mongo.go         — Mongo client (same shape as user_service's)
  config/config.go            — Config/MongoConfig/HostConfig/JWTConfig
  entity/
    service/driver.go         — Driver, TaxiType, Status (no tags)
    repository/driver.go      — bson-tagged projection
    http/
      requests.go             — UpdateProfileReq, UpdateStatusReq, CreateDriverReq
      responses.go            — DriverResp
  handler/
    http/
      http.go                 — Handler struct + constructor
      driver_handler.go        — Profile / UpdateProfile / UpdateStatus / CreateDriver
      middleware/auth.go        — same JWT verification as user_service (own
                                   copy — services never import each other,
                                   `ARCHITECTURE.md:165-166`)
  service/
    service.go                — DriverRepository interface + DriverService + New()
    get_profile.go
    update_profile.go
    update_status.go
    create_driver.go
  repository/mongo/
    driver_repository.go      — struct, NewDriverRepository, interface assertion
    find_by_user_id.go
    create_driver.go
    update_profile.go
    update_status.go
  migrations/mongo/
    migration.go               — unique_index_user_id
    runner.go                  — same shape as user_service's
```

## Endpoints

Self-service endpoints follow the exact pattern already validated in
`user_service`'s `/profile`: identity comes from the JWT (`userID` set by the
`Auth` middleware), never from a path parameter.

| Method | Path | Auth | Notes |
|---|---|---|---|
| `GET` | `/profile` | JWT | Own driver profile by `userID` from token |
| `PATCH` | `/profile` | JWT | Update `taxi_type` |
| `PATCH` | `/profile/status` | JWT | Update `status` (available/on-trip/offline) |
| `POST` | `/internal/drivers` | none (temporary) | Create a driver record — see below |

`JWTConfig.Secret` is the same shared secret convention already established
for `user_service` (HS256, `sub` = user id, `roles` = `[]string`) — both
services verify tokens issued by the same future Auth Service, so the secret
must be configured identically (same `JWT_SECRET` value) wherever both
services are deployed.

### Temporary driver creation without Kafka

`POST /internal/drivers` accepts `{"user_id": "...", "taxi_type": "..."}` and
creates a driver record. This mirrors `user_service`'s existing
`/internal/verify-credentials` pattern: an endpoint trusted because of its
network position (internal-only), not because of a request-level auth check.
It exists solely to unblock testing/demoing Profile and Status management
before the Kafka producer/consumer exist. Step 2's spec will decide whether
to remove it once the Kafka consumer takes over, or keep it as an admin
escape hatch — not decided now, out of scope for this spec.

## Validation

Same library/pattern as `user_service`: `go-playground/validator`, one custom
validator for `taxi_type`/`status` enum membership (mirrors the existing
`role` validator in `user_service/validation/validation.go`).

## Testing

Same two-tier pattern already proven in `user_service`:
- `service/*_test.go` — unit tests against a fake `DriverRepository`, no Mongo.
- `repository/mongo/*_integration_test.go` (build tag `integration`) —
  `dockertest`-based, real Mongo container, `TestMain` wires connection +
  runs migrations (must return the exit code from a `run(m)` helper so
  `defer` cleanup actually executes — this exact bug was caught and fixed in
  `user_service`'s integration tests; don't repeat it).

## Deployment

- `Dockerfile` at `services/driver_service/Dockerfile` — same multi-stage
  shape as `user_service/Dockerfile` (build context = repo root, because of
  the `shared` module's relative `replace` directive).
- Add `driver_service` to the root `docker-compose.yaml`, same shape as the
  `user_service` entry (own Mongo database name, own host port — e.g. `8081`
  on the host, since `8080` is already `user_service`'s).

## Explicitly not decided here (deferred to step 2's spec)

- Kafka client library choice, topic/schema naming, whether Kafka events use
  proto (per `ARCHITECTURE.md`'s target) or plain JSON (the pragmatic
  deviation `user_service` already took for HTTP).
- Fate of `POST /internal/drivers` once the Kafka consumer exists.
