# Auth Service — Design

## Context

InnoTaxi is a taxi-booking microservices monorepo (see root `README.md` and
`ARCHITECTURE.md`). `user_service`, `driver_service`, and `order_service`
exist. None of them can currently obtain a real JWT — every existing
`middleware.Auth`/`interceptor.AuthInterceptor` only *validates* a token
signed with a shared static secret (`JWT_SECRET`); nothing *issues* one.
Auth Service is the piece that closes that loop.

Gateway Service doesn't exist yet, so there is no component in front of Auth
Service today. Its HTTP endpoints (`/login`, `/refresh`, `/logout`,
`/validate`) are directly client-facing for now, exactly like
`user_service`'s `/register` is today. When Gateway Service is built, it
will call `/validate` (or a gRPC equivalent — not decided, see "Explicitly
not decided here") instead of validating tokens itself.

## Scope (from `README.md`'s Auth Service section + Authentication Flow)

In scope:
- **Login**: `POST /login` — delegate credential verification to
  `user_service`'s existing `POST /internal/verify-credentials`, issue an
  access token (short TTL) + refresh token (long TTL), store both as active
  sessions in Redis.
- **Token validation**: `POST /validate` — check JWT signature/expiry *and*
  that the session is still active in Redis (this is what makes logout and
  refresh-rotation actually revoke a token instead of just discarding a
  client-side reference to it).
- **Token refresh**: `POST /refresh` — exchange a valid, still-active
  refresh token for a brand new access+refresh pair (rotation: the old
  refresh token is invalidated in the same call, so it can't be replayed).
- **Logout**: `POST /logout` — invalidate both tokens' sessions in Redis
  immediately.
- Infra: stand up Redis in docker-compose, wire `auth_service` in
  alongside it.

Explicitly out of scope for this step:
- **Gateway-facing gRPC token validation** (`README.md`'s "Synchronous
  (gRPC): Gateway to Auth Service for token validation") — there is no
  Gateway Service yet to call it. `/validate` covers the same need over
  HTTP now; a gRPC surface is added if/when Gateway Service needs one,
  mirroring how `order_service`'s driver-assignment gRPC calls were
  deferred until they had a caller.
- **Role changes taking effect without re-login.** Roles are captured at
  login time and carried forward through `/refresh` (see "Known
  trade-off" below) rather than re-fetched from `user_service` on every
  refresh.
- Rate limiting / brute-force lockout on `/login` (README assigns rate
  limiting to Gateway Service).

## Decisions made during design (with rationale)

- **HTTP (Gin), not gRPC**, matching `README.md`'s Auth Service tech-stack
  line ("HTTP Framework: Gin with middleware for token validation") and the
  same reasoning already used for `user_service`
  ([[project_innotaxi_user_service_context]]): the README pins this
  service's transport explicitly, so there's no ambiguity to resolve the
  way Order Service had to.
- **Redis only, no Postgres/Mongo.** Auth Service owns no durable user data
  — its only state is active sessions, which are inherently ephemeral
  (bounded by token TTL) and exactly what Redis is for. This matches
  README's "Database: Redis for session storage and refresh token
  management" line for Auth Service specifically.
- **Session revocation via a `sid` (session ID) claim, not by storing the
  JWT itself in Redis.** A JWT is self-verifying (signature + `exp`), so
  Redis's only job is answering "has this session been revoked before
  natural expiry?" — a UUID `sid` claim shared by an access/refresh pair,
  used as a Redis key, is smaller and simpler than storing/parsing whole
  tokens back out of Redis. Two keys per session:
  - `auth:session:<sid>:access` — exists with TTL = access-token TTL
  - `auth:session:<sid>:refresh` — exists with TTL = refresh-token TTL

  `/validate` and `/refresh` check existence, not content — the value is a
  placeholder (`"1"`); nothing downstream reads it back out of Redis
  (`user_id`/`roles` come from the JWT claims themselves).
- **Refresh rotates the session ID.** Reusing the same `sid` across
  refreshes would let a stolen refresh token keep working indefinitely
  through repeated `/refresh` calls with no way to distinguish the
  legitimate client from an attacker. Rotating (`/refresh` deletes the old
  `sid`'s keys and mints a new `sid` for the new pair) means a refresh
  token is single-use — the standard refresh-rotation pattern, and cheap
  to add now versus retrofitting later.
- **Known trade-off: roles come from the refresh token's own claims, not a
  fresh `user_service` lookup.** At `/refresh` time, Auth Service has no
  cheap way to re-verify current roles without calling `user_service` (no
  "get roles by user ID" endpoint exists, and adding one is out of scope
  here). The refresh token carries `roles` alongside `sub`, and `/refresh`
  copies them into the new access token unchanged. Consequence: an admin
  revoking/granting a role between logins won't take effect until the
  user's refresh token also expires and they log in again. Acceptable for
  now — flagged here so it isn't mistaken for an oversight later.
- **`/logout` accepts and invalidates both tokens by design**, per
  README's Logout flow — a client might call it with only one token still
  unexpired; both `sid` keys are deleted if present, and a missing key is
  not an error (idempotent logout).
- **JWT signing stays HS256 with the existing shared `JWT_SECRET`**, same
  convention as every other service, so tokens Auth Service issues are
  immediately valid against `user_service`/`driver_service`/`order_service`'s
  existing verify-only middleware without any of them changing.

## Data model

```go
// entity/service/auth.go
type TokenPair struct {
    AccessToken  string
    RefreshToken string
}

type AccessClaims struct {
    UserID string
    Roles  []string
    jwt.RegisteredClaims // Subject = UserID, ExpiresAt, ID = sid
}

type RefreshClaims struct {
    UserID string
    Roles  []string
    jwt.RegisteredClaims // Subject = UserID, ExpiresAt, ID = sid
}
```

```go
// entity/gateway/user.go — what user_service's VerifyCredentials returns,
// trimmed to what Auth Service actually needs
type UserInfo struct {
    ID    string
    Roles []string
}
```

No repository-layer entity (`entity/repository/`) — Redis keys are plain
strings, there's no row/document shape to project.

## Architecture

```
services/auth_service/
  go.mod                          — require .../shared, replace ../../shared
  cmd/main.go
  config/config.go                — Redis, HTTP, JWT (secret + access/refresh
                                     TTL), UserService (base URL) config

  app/
    run.go                        — wires deps, starts HTTP server,
                                     graceful shutdown (SIGINT/SIGTERM)
    db/
      redis/redis.go              — Redis client setup (github.com/redis/go-redis/v9)

  entity/
    service/auth.go               — TokenPair, AccessClaims, RefreshClaims
    gateway/user.go                — UserInfo (trimmed user_service response)

  handler/
    http/
      http.go                     — Handler struct + constructor
      login.go                    — POST /login
      refresh.go                  — POST /refresh
      logout.go                   — POST /logout
      validate.go                 — POST /validate

  gateway/
    user_service/
      user_service.go              — UserServiceGateway interface + HTTP
                                      client: VerifyCredentials(ctx, login,
                                      password) (entity/gateway.UserInfo, error)

  service/
    service.go                    — SessionStore interface, UserServiceGateway
                                     interface, AuthService struct, New()
    login.go                      — Login use case
    refresh.go                    — Refresh use case (rotation)
    logout.go                     — Logout use case
    validate.go                   — Validate use case
    token.go                      — issueTokenPair(userID, roles) helper:
                                     mint sid, sign both JWTs

  repository/
    redis_repo/
      redis_repo.go                — SessionRepository: implements SessionStore
                                      (SaveAccess/SaveRefresh/Exists.../Delete...)
                                      against *redis.Client

  errorsx/errors.go                — ErrInvalidCredentials, ErrInvalidToken,
                                      ErrSessionNotFound
```

`service.SessionStore` interface (what `repository/redis_repo` implements):

```go
type SessionStore interface {
    SaveAccessSession(ctx context.Context, sid string, ttl time.Duration) error
    SaveRefreshSession(ctx context.Context, sid string, ttl time.Duration) error
    AccessSessionExists(ctx context.Context, sid string) (bool, error)
    RefreshSessionExists(ctx context.Context, sid string) (bool, error)
    DeleteAccessSession(ctx context.Context, sid string) error
    DeleteRefreshSession(ctx context.Context, sid string) error
}
```

`service.UserServiceGateway` interface (what `gateway/user_service`
implements):

```go
type UserServiceGateway interface {
    VerifyCredentials(ctx context.Context, login, password string) (gateway_dto.UserInfo, error)
}
```

Both interfaces live in `service/service.go` and are satisfied by concrete
types in `repository/`/`gateway/`, consistent with the interface-at-the-
service-boundary pattern already applied to `order_service` and
`driver_service` in the stabilization pass before this spec.

## Endpoints

| Method | Path | Auth | Request | Response |
|---|---|---|---|---|
| `POST` | `/login` | none | `{login, password}` | `200 {access_token, refresh_token}` / `401` invalid credentials |
| `POST` | `/refresh` | none (refresh token is the credential) | `{refresh_token}` | `200 {access_token, refresh_token}` (new pair) / `401` invalid/expired/revoked |
| `POST` | `/logout` | none (tokens are the credential) | `{access_token, refresh_token}` | `204` (idempotent — always succeeds if tokens parse) |
| `POST` | `/validate` | none (token is the credential) | `{access_token}` | `200 {user_id, roles}` / `401` invalid/expired/revoked |

None of these sit behind `middleware.Auth` themselves — the token *is* the
credential being presented to each of them, same as how `/login` itself
can't require a token.

## Deployment

`docker-compose.yaml` currently has `mongo`/`mongo-ui`/`postgres`/`user_service`.
Add:

- `redis` — single instance, no auth needed for local dev, healthcheck via
  `redis-cli ping`.
- `auth_service` — HTTP port, `depends_on: redis (healthy), user_service
  (started)`, needs `USER_SERVICE_BASE_URL` pointing at `user_service`'s
  container DNS name.

`services/auth_service/Dockerfile` mirrors `services/user_service/Dockerfile`
(repo-root build context, for the same `shared` replace-directive reason).

## Explicitly not decided here (deferred to later specs)

- Whether Gateway Service calls `/validate` over HTTP or gets a gRPC
  `ValidateToken` RPC instead — decide when Gateway Service is actually
  built and its own transport choices are made.
- Rate limiting on `/login` (assigned to Gateway Service by README, which
  doesn't exist yet).
- Any mechanism to push a role change into an *active* session before its
  refresh token naturally expires (see "Known trade-off" above).
