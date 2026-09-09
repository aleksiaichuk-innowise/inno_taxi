# Gateway Service — Design

## Context

`user_service`, `driver_service`, `order_service`, and `auth_service` all
exist now, each on its own port, each independently validating its own
JWTs. There is still no single entry point a client can talk to — and
`README.md` pins Gateway Service to a specific, non-Go stack: **NGINX**
("Gateway Platform: NGINX", "Routing: NGINX location blocks and upstreams",
"Rate Limiting: NGINX modules (limit_req)"). Nothing here is a Go service;
this spec is about `nginx.conf`, not a `service/`/`repository/` split.

`auth_service`'s own spec
([[2026-09-09-auth-service-design]]) already anticipated this: `/validate`
exists specifically because "a future Gateway Service will call it instead
of validating tokens itself," and its rate-limiting section was explicitly
deferred to Gateway Service. This spec is where both of those come due.

## Scope (from `README.md`'s Gateway Service section)

In scope:
- **Request routing**: reverse-proxy incoming requests to `user_service`,
  `driver_service`, `auth_service` by URL prefix. `order_service` gets a
  routing block too (it already has a stable proto-defined HTTP path), but
  see "Explicitly not decided here" — it isn't live-tested yet because
  `order_service` itself isn't in `docker-compose.yaml`.
- **Token validation**: gate protected routes behind an `auth_request` to
  Auth Service's `/validate`.
- **Role-based access control**: demonstrate a role-gated route (the one
  that already exists — `user_service`'s admin-only
  `POST /admin/users/:id/analyst-role`) using nginx's `auth_request` against
  a second internal location that asks Auth Service for a specific role.
- **Rate limiting**: `limit_req_zone` on `/auth/login`, closing the gap
  `auth_service`'s spec flagged.
- Infra: `gateway_service` container in `docker-compose.yaml`, in front of
  the services that are already there.

Explicitly out of scope for this step:
- **Load balancing across multiple replicas** of a backend — every
  upstream has exactly one `server` today. The `upstream {}` blocks are
  structured so adding replicas later is just adding more `server` lines,
  but there's nothing to balance across yet.
- **Wallet Service / Analytic Service routes** — neither service exists.
  Nothing to route to.
- **Request/response transformation** — README lists it as a capability,
  but nothing today needs a transformed request/response; not adding
  speculative rewrite rules with no concrete use case.
- **Wiring `order_service`/`driver_service` fully into
  `docker-compose.yaml` with their own infra** — `driver_service` gets a
  `Dockerfile` and a compose entry here (it only needs Mongo, which is
  already running) so at least two real backends are reachable through the
  gateway end-to-end. `order_service` additionally needs Kafka + Elasticsearch
  in compose, which its own design doc already flagged as "a pre-existing
  gap, not this spec's concern" — still true, still deferred.

## Decisions made during design (with rationale)

- **Plain NGINX, no Lua/OpenResty.** README's tech-stack line says "NGINX"
  without qualification, and everything this step needs — reverse proxy,
  rate limiting, and even role-based gating — is reachable with stock
  `ngx_http_auth_request_module` plus `ngx_http_limit_req_module`, both
  compiled into the standard `nginx:alpine` image. No reason to pull in a
  scripting runtime for what config directives already do.
- **`auth_request` + per-role internal locations, not a single
  `/validate` call inspected ad hoc.** `auth_request` is the standard NGINX
  pattern for "ask another service whether this request may proceed": a
  non-2xx response from the subrequest becomes the response to the
  original request automatically. To get role-specific gating out of a
  single boolean subrequest, the *required role* has to be supplied by the
  subrequest itself — so each role gets its own tiny `internal` location
  (`/_gateway/validate`, `/_gateway/validate/admin`, ...) that sets an
  `X-Required-Role` header before proxying to Auth Service. The route ↔
  role mapping — the actual access policy — lives in `nginx.conf`, which is
  what "Gateway Service... enforces access policies" means concretely here.
- **`auth_service`'s `/validate` contract changed from a JSON body to the
  `Authorization` header** (see `auth_service/handler/http/validate.go`).
  An `auth_request` subrequest does not carry the original request body by
  default, and forwarding it through would need extra `nginx.conf`
  plumbing for no benefit — the token is naturally already in a header on
  every real request. This is a deliberate, tested contract change made
  as part of this spec, not a leftover from before Gateway existed to
  consume it.
- **Defense in depth kept, not replaced.** README's Request Authentication
  flow describes the target service processing a gateway-forwarded request
  "without additional authentication checks" — but `user_service`,
  `driver_service`, and `order_service` already independently validate
  JWTs (established in the stabilization pass before Auth Service existed).
  Ripping that out just to match the README's idealized single-checkpoint
  model would be a regression for zero benefit: every request still carries
  its own bearer token, so backend-side validation costs nothing extra and
  survives a gateway misconfiguration. Gateway's `auth_request` is an
  additional coarse-grained + rate-limiting layer, not the only one.
- **URL prefixing to resolve a real path collision.** Both `user_service`
  and `driver_service` expose `/profile` for semantically different things
  (own account vs. driver status/taxi-type). Routing through the gateway
  needs unambiguous prefixes: `/auth/*`, `/users/*`, `/drivers/*`,
  `/orders/*` map to each service with the prefix stripped before
  `proxy_pass`. This is an edge-only rewrite — no backend route changed.
- **`/internal/*` paths are never routed.** `user_service`'s
  `/internal/verify-credentials` and `driver_service`'s
  `POST /internal/drivers` / `GET /internal/drivers` are trusted because of
  their network position (only reachable service-to-service inside the
  compose network), exactly as their own specs already state. Gateway's
  location blocks are an explicit allowlist of public paths — there is no
  catch-all proxy, so nothing internal is reachable through it by
  construction, not by a rule that has to be remembered to keep excluding
  it.

## Route map

| Public path | Auth | Backend |
|---|---|---|
| `POST /auth/login` | none (rate-limited) | `auth_service` `POST /login` |
| `POST /auth/refresh` | none | `auth_service` `POST /refresh` |
| `POST /auth/logout` | none | `auth_service` `POST /logout` |
| `POST /users/register` | none | `user_service` `POST /register` |
| `GET/PATCH/DELETE /users/profile`, `POST /users/profile/password` | `auth_request` (any authenticated user) | `user_service` `/profile*` |
| `POST /users/admin/users/:id/analyst-role` | `auth_request` (role=admin) | `user_service` `POST /admin/users/:id/analyst-role` |
| `GET/PATCH /drivers/profile`, `PATCH /drivers/profile/status` | `auth_request` (any authenticated user) | `driver_service` `/profile*` |
| `POST /orders` | `auth_request` (any authenticated user) | `order_service` `POST /v1/orders` (config present, not live — see scope) |

## Architecture

```
services/gateway_service/
  Dockerfile              — FROM nginx:alpine, COPY nginx.conf
  nginx.conf               — the whole gateway: upstreams, auth_request
                              locations, rate limiting, route map above
  smoketest.sh              — README's "Testing: Integration tests using
                               HTTP clients" for a non-Go service: a
                               curl-based script exercising the route map
                               against a live docker-compose stack
```

No Go code, no `entity/`/`service/`/`repository/` split — there's no
business logic here to layer, just declarative routing config, which is
exactly what `ARCHITECTURE.md`'s Clean Architecture template doesn't apply
to (that template is for the six Go services).

## Deployment

`docker-compose.yaml` gets:
- `driver_service` — new `Dockerfile` (mirrors `user_service`'s), compose
  entry depending on `mongo`, so `/drivers/*` routes are live-testable.
- `gateway_service` — depends on `auth_service`, `user_service`,
  `driver_service`; publishes the single public port (`8000`) a client
  would actually hit.

`order_service` stays out of compose (see "Explicitly not decided here" in
its own spec) — its `nginx.conf` block is present and correct against its
proto contract, but exercising it end-to-end waits for that service's own
Kafka/Elasticsearch compose wiring.

## Explicitly not decided here (deferred to later work)

- Wiring `order_service` (and its Kafka/ES dependencies) into
  `docker-compose.yaml` — still `order_service`'s own follow-up, not
  Gateway's.
- Wallet Service / Analytic Service routes — added when those services
  exist.
- TLS termination at the gateway (README's "TLS for all communications" is
  a broader technical requirement, not scoped to this step; local dev
  stays plain HTTP like every other service today).
