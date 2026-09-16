# Kubernetes Deployment & Real-Host Rollout — Design

**Status:** Approved (design walked through section-by-section with the user; each section confirmed before moving on).

## Purpose

InnoTaxi currently only runs via `docker-compose.yaml` — single host, dev-only,
no Kubernetes, no CI/CD, no real-host deploy story. README's non-functional
requirements name Kubernetes + Helm charts for production, GitHub Actions
CI/CD, and Kubernetes Secrets for config. This spec designs the full path
from "code in the repo" to "running on a real host, reachable over HTTPS on
a real domain, updated automatically on every push to `main`."

**This is a learning project, not a production system.** It will not run
under real load. Every choice below optimizes for "correctly satisfies the
README's stated NFRs with the least new infrastructure," not for
production-grade resilience, scale, or security hardening. Single replicas
everywhere, no autoscaling, no multi-node HA, no Prometheus/Jaeger/GraphQL —
those are out of scope (see Non-Goals).

**Process note:** this project is being built in mentor mode for this phase.
Claude designs (this spec, the plan) and reviews; the user writes every
manifest, workflow file, and script by hand. The implementation plan that
follows this spec is written as a sequence of study-material-then-task
checkpoints, not code the user pastes in.

## Target Environment

One VPS (Ubuntu 22.04/24.04, **minimum 4 vCPU / 8 GB RAM** — the stack runs
Elasticsearch, Kafka, ClickHouse, 2×Postgres, MongoDB, Redis, and 7 Go
services on a single node simultaneously; a 1-2 GB "cheapest tier" VPS will
not hold this) running **k3s**, a lightweight but fully standard Kubernetes
distribution. A single k3s node satisfies both "deploy to Kubernetes" and
"deploy to a real host" at once — no separate local kind/minikube cluster is
needed, since that would just be a second environment to maintain for no
benefit at this scope.

k3s ships with two components pre-installed that this design deliberately
reuses instead of installing alternatives:
- **Traefik** as the ingress controller.
- **`local-path-provisioner`** as the default `StorageClass` (dynamically
  provisions a PVC as a hostPath directory on the node — correct for a
  single-node cluster, no NFS/cloud-disk/Longhorn needed).

The user already has a domain/subdomain they control (exact hostname to be
decided; see "Domain" below).

## Architecture

```
GitHub push to main
   -> CI: lint/vet, unit+integration tests, govulncheck, trivy scan
   -> CI: docker build + push (7 images) to ghcr.io
   -> CI: helm upgrade --install (kubeconfig secret) against the k3s API
          on the VPS
   -> k3s: Traefik routes HTTPS traffic for the domain to gateway-service
   -> gateway-service (nginx, existing role-gating logic, unchanged)
          -> routes to the other 6 services exactly as it does today
   -> each service's StatefulSet dependency (its own Postgres/Mongo/etc.)
```

One Helm chart, `deploy/helm/inno-taxi/`, describes the entire stack: the 7
existing services (as `Deployment` + `Service`) and the 7 stateful
dependencies they need (as `StatefulSet` + `PersistentVolumeClaim`), plus one
`Ingress` and one `Secret`. Nothing here changes any application code or
`gateway_service`'s `nginx.conf` routing logic — this is packaging and
infrastructure only.

## Chart Structure

```
deploy/helm/inno-taxi/
  Chart.yaml
  values.yaml                    # domain, image repo/tag, PVC sizes, credentials defaults
  templates/
    _helpers.tpl                 # name/label helpers
    secrets.yaml                 # one Secret, keys sourced from values
    ingress.yaml                 # one Ingress -> gateway-service:8080, TLS via cert-manager annotation
    services/
      user-service.yaml
      driver-service.yaml
      order-service.yaml
      auth-service.yaml
      wallet-service.yaml
      analytic-service.yaml
      gateway-service.yaml
    infra/
      postgres-order.yaml        # StatefulSet + ClusterIP Service + volumeClaimTemplate
      postgres-wallet.yaml
      mongo.yaml
      redis.yaml
      kafka.yaml
      clickhouse.yaml
      elasticsearch.yaml
```

Each file under `services/` follows one pattern: a `Deployment` with a single
container (image `ghcr.io/<owner>/<repo>-<service>:{{ .Values.image.tag }}`),
env sourced from the shared `Secret` (`envFrom`/`secretKeyRef`) matching that
service's existing `config/config.go` fields, and a `ClusterIP` `Service`
exposing its container port. `gateway-service`'s `Deployment` is the same
shape but built from `services/gateway_service`'s own Dockerfile/context
(it has no Go module, no `shared` dependency).

**`gateway_service`'s nginx config needs two variants, not one shared
file — discovered during the real deploy (2026-09-16), initially got this
wrong by editing the single existing `nginx.conf` in place.** The original
plan mutated `services/gateway_service/nginx.conf` directly for
Kubernetes, which silently broke the Docker Compose dev deployment (the
same file backs both `docker-compose.yaml`'s `gateway_service` build and
this chart's `gateway-service` image) — README requires both "Development:
Docker Compose" and "Production: Kubernetes" to work, so breaking one to
fix the other isn't acceptable. The chosen fix: **two config files and two
Dockerfiles**, not templating (an `envsubst`-based single-template
approach was considered and rejected — nginx configs are already full of
`$variables` of their own, e.g. `$remote_addr`, `$upstream_http_x_user_id`,
and a blanket `envsubst` pass risks colliding with them; two explicit
files are simpler to reason about for this project's scale):

- `services/gateway_service/nginx.conf` — unchanged, still backs
  `docker-compose.yaml` via the existing `Dockerfile`. Keeps Docker
  Compose's own DNS (`resolver 127.0.0.11`) and underscored container
  names (`auth_service:8082`, etc.) exactly as they were.
- `services/gateway_service/nginx.k8s.conf` (new) + `Dockerfile.k8s` (new,
  otherwise identical to `Dockerfile`, just `COPY`ing the k8s config) —
  used only when building `gateway-service`'s image for this chart. Three
  differences from the Compose version:
  1. `resolver 127.0.0.11` → `resolver 10.43.0.10` — `127.0.0.11` is
     Docker Compose's embedded per-network DNS server, which doesn't exist
     inside a Kubernetes pod. `10.43.0.10` is CoreDNS's ClusterIP, which
     k3s assigns deterministically from its default `10.43.0.0/16` service
     CIDR — since this deploy targets k3s specifically, that constant is
     knowable ahead of time.
  2. Every `set $<name>_service "<name>_service:<port>";` literal's
     hostname changes from underscored (`auth_service`) to hyphenated
     (`auth-service`) — a Kubernetes `Service` name is a DNS-1035 label
     (`[a-z]([-a-z0-9]*[a-z0-9])?`) and **cannot contain underscores**, so
     it must match the `Service` names this chart actually creates.
  3. Each of those hostnames must additionally be a **fully-qualified**
     in-cluster DNS name (`user-service.default.svc.cluster.local:8080`,
     not `user-service:8080`) — `nginx`'s `resolver` directive doesn't
     consult `/etc/resolv.conf` at all, not just for the nameserver address
     (point 1) but also the pod's DNS `search` list
     (`default.svc.cluster.local`, `svc.cluster.local`, `cluster.local`)
     that lets ordinary clients (Go's resolver, `curl`, etc.) resolve a
     bare name like `mongo`. `nginx`'s resolver sends exactly the hostname
     it's given, so an unqualified `user-service` gets NXDOMAIN from
     CoreDNS. `default` is the namespace this chart deploys into.

CI (Task 16) must build `gateway-service`'s image with
`-f services/gateway_service/Dockerfile.k8s` — every other service still
builds from its regular `Dockerfile`, unchanged.

## Stateful Dependencies

Each of the 7 infra dependencies becomes a `StatefulSet` (stable pod
identity + a `PersistentVolumeClaim` that survives pod restarts, unlike a
bare `Deployment` where replacement pods get fresh, unrelated storage) with
one replica, plus a normal `ClusterIP` `Service` in front of it. A headless
`Service` (`clusterIP: None`, giving per-pod DNS like `kafka-0.kafka`) is
the idiomatic choice for a *multi-replica* StatefulSet where clients need to
address a specific ordinal — with exactly one replica everywhere here, a
plain `ClusterIP` `Service` is just as stable and lets every dependency keep
the same hostname `docker-compose.yaml` already uses (`kafka:9092`,
`postgres-order:5432`, ...), so env vars need no k8s-specific rewriting.
Configuration mirrors `docker-compose.yaml` 1:1:

| Dependency | Source of truth today | k8s-specific adjustment |
|---|---|---|
| `postgres-order` | `docker-compose.yaml`'s `postgres` | Separate StatefulSet from wallet's; same image (`postgres:18`), env from Secret |
| `postgres-wallet` | `docker-compose.yaml`'s `wallet_postgres` | Same as above, separate PVC |
| `mongo` | `docker-compose.yaml`'s `mongo` | `mongo:7`, root creds from Secret |
| `redis` | `docker-compose.yaml`'s `redis` | `redis:7-alpine`, no changes needed |
| `kafka` | `docker-compose.yaml`'s `kafka` (KRaft mode, broker+controller combined, node ID 1) | `KAFKA_ADVERTISED_LISTENERS` stays `PLAINTEXT://kafka:9092` unchanged — the `Service` name `kafka` resolves the same way a compose container name does |
| `clickhouse` | `docker-compose.yaml`'s `clickhouse` + `services/analytic_service/clickhouse-listen-ipv4.xml` bind mount | The IPv4-only `config.d` override becomes a `ConfigMap` mounted at `/etc/clickhouse-server/config.d/listen-ipv4.xml` instead of a bind mount |
| `elasticsearch` | `docker-compose.yaml`'s `elasticsearch` (`discovery.type=single-node`, `xpack.security.enabled=false`) | Same env; additionally needs the host kernel's `vm.max_map_count >= 262144` (see VPS Bootstrap) or the pod crash-loops on startup |

**Migrations stay manual.** `order_service`/`wallet_service` migrations are
already a manual `make migrate-<service>-up` step today (CLAUDE.md: "applied
manually... not run automatically on startup"), not automated even in
`docker-compose`. This design keeps that convention rather than adding a
Helm post-install Job: `kubectl port-forward` the target Postgres
StatefulSet's Service to a local port, then run the same `goose` Makefile
target against it. Automating this would be new behavior beyond what the
project already does locally, not something this deploy effort needs to
add.

## Ingress & TLS

One `Ingress` resource, single host rule pointing to the `gateway-service`
`Service` (port 8080) — no path-based routing at the k8s layer, since all
role-based routing logic already lives inside `gateway_service`'s
`nginx.conf` and stays there unchanged. Using k3s's built-in Traefik avoids
installing and configuring a second ingress controller (e.g. ingress-nginx)
that would duplicate what Traefik already does out of the box.

**cert-manager** is installed once into the cluster as a platform add-on
(`helm install cert-manager jetstack/cert-manager`), separate from this
app's own chart. A `ClusterIssuer` targets Let's Encrypt via the HTTP-01
challenge (works once the domain's A record points at the VPS's public IP
and port 80 is reachable). The chart's `Ingress` carries the
`cert-manager.io/cluster-issuer: letsencrypt-prod` annotation and a `tls:`
block; cert-manager watches the Ingress, completes the challenge, and
populates the referenced `Secret` with the certificate — including
automatic renewal before expiry, no manual intervention after initial setup.

**Domain:** not yet decided by the user. Modeled as `values.yaml`'s
`domain` field (a normal Helm configuration value, not a placeholder to fill
in later inside the design) — set via `--set domain=<chosen-hostname>` at
install time, or in the `values-prod.yaml` overlay used by CI.

## Secrets

One `Secret` (`inno-taxi-secrets`), templated from `values.yaml` — the same
credentials (Postgres passwords, Mongo root user/pass, `auth_service`'s JWT
signing key) currently sitting in `docker-compose.yaml`'s `environment:`
blocks / a local `.env`. Every `Deployment`/`StatefulSet` reads what it needs
via `secretKeyRef`. README names plain "Kubernetes Secrets" explicitly as
the NFR — no Vault, no sealed-secrets, no external secret manager; those
would be new infrastructure this project doesn't ask for.

## CI/CD (GitHub Actions)

README names the exact pipeline stages this follows:

1. **Code quality** — per module (all 7 services): `gofmt -l .` (fails the
   job if any file is unformatted) + `go vet ./...`.
2. **Unit + integration tests** — `go test ./...` everywhere; additionally
   `go test -tags=integration ./...` for the services that actually have
   such a suite today (`user_service`, `wallet_service`, `analytic_service`,
   per CLAUDE.md — the other services simply report "no test files" and
   exit 0, no per-service branching needed in the workflow). GitHub-hosted
   runners have a normal Docker socket, so the local machine's
   `DOCKER_HOST=unix:///home/user/.docker/desktop/docker.sock` workaround
   documented in this repo's local dev notes does not apply in CI.
3. **Security scanning** — `govulncheck` per module (official Go tooling,
   no new infrastructure) plus a `trivy image` scan of each freshly built
   image before it's pushed.
4. **Docker build & push (`main` branch only)** — one image per service,
   built with the same Docker context each already uses in
   `docker-compose.yaml` (repo root for the 6 Go services, since they
   `replace` the `shared` module via a relative path; `services/gateway_service`
   for the gateway, which has no Go module). Tagged with `${{ github.sha }}`
   and `latest`, pushed to `ghcr.io/<owner>/<repo>-<service>`.
5. **Deploy (`main` branch only, after all 7 images are pushed)** — a single
   job installs `kubectl`/`helm`, reconstructs the kubeconfig from the
   `KUBE_CONFIG_B64` GitHub Secret, and runs
   `helm upgrade --install inno-taxi ./deploy/helm/inno-taxi -f values-prod.yaml --set image.tag=${{ github.sha }} --wait`.
   Deploying all 7 services together (one Helm release per push) keeps the
   running stack at one consistent version instead of services drifting out
   of sync with each other.

Stages 1-4 run as a matrix across the 7 services in parallel; stage 5 is a
single job gated on every matrix job succeeding.

## VPS Bootstrap (runbook, not automation)

This is a one-time, manual setup — not Terraform/Ansible, which would be new
infrastructure disproportionate to a test assignment. Documented as a
runbook the user follows once when the VPS is provisioned:

1. Provision an Ubuntu 22.04/24.04 VPS, ≥4 vCPU / 8 GB RAM.
2. Install k3s with the node's public IP in the API server's TLS SAN (required
   because k3s only signs the API server cert for localhost/private IPs by
   default, and the kubeconfig used by CI — which runs on GitHub-hosted
   runners, not on the VPS itself — connects via the public IP):
   ```bash
   curl -sfL https://get.k3s.io | sh -s - --tls-san <VPS-public-IP>
   ```
   This also installs Traefik and `local-path-provisioner`, so no separate
   ingress-controller or storage-provisioner install is needed.
3. Fix Elasticsearch's kernel prerequisite (a host-level setting, not a k8s
   object — must happen on the node itself):
   ```bash
   sysctl -w vm.max_map_count=262144
   echo 'vm.max_map_count=262144' > /etc/sysctl.d/99-elasticsearch.conf
   ```
4. Firewall: open 22 (SSH), 80/443 (HTTP/HTTPS — cert-manager's HTTP-01
   challenge and all user traffic), and 6443 (k8s API — needed by CI, which
   runs from GitHub-hosted runners with no fixed IP range, so this port stays
   open to the internet; the API server's TLS client-certificate
   authentication is the only protection here, an accepted trade-off for a
   learning project, not something to carry into a real production
   deployment).
5. Extract the kubeconfig from `/etc/rancher/k3s/k3s.yaml`, replace
   `server: https://127.0.0.1:6443` with `server: https://<VPS-public-IP>:6443`,
   base64-encode it, and store it as the `KUBE_CONFIG_B64` GitHub Actions
   secret.
6. Once the VPS's public IP is known, point the chosen domain's A record at
   it (required before cert-manager's HTTP-01 challenge can succeed).
7. Install cert-manager into the cluster (one-time platform add-on, not part
   of this app's chart) and create the `ClusterIssuer` for Let's Encrypt.

## Non-Goals

- Multi-node clusters, autoscaling (HPA/VPA/cluster autoscaler), pod
  anti-affinity, or any other high-availability concern — single replica of
  everything is intentional for a learning project that will not run under
  real load.
- Prometheus/Grafana, Jaeger tracing, GraphQL, Swagger/OpenAPI generation,
  Postman collections — all named as NFRs elsewhere in the README, but not
  part of "get the app running on a real k8s host," and not requested for
  this phase.
- `develop` branch / GitHub Flow branch protection rules — README names a
  branching strategy, but this spec is scoped to deployment mechanics, not
  repository/process policy.
- A secrets manager beyond plain Kubernetes `Secret` objects (Vault, sealed-
  secrets, cloud KMS) — README asks for Kubernetes Secrets specifically.
- Automating database migrations as part of the Helm release — stays a
  manual `goose` step via `kubectl port-forward`, consistent with how this
  project already runs migrations today.
- Terraform/Ansible/any VPS-provisioning-as-code — the VPS bootstrap is a
  manual runbook; introducing an IaC tool would be new infrastructure this
  effort doesn't need.

## Testing / Verification

- Each Helm template change: `helm template deploy/helm/inno-taxi` renders
  without error, and the user reviews the rendered YAML for the specific
  resource just added.
- `helm install --dry-run` against a real cluster context before merging.
- After a real `helm install`: `kubectl get pods` all `Running`/`Ready`,
  then an end-to-end smoke check (register a user, create an order) through
  the public domain over HTTPS — the same shape of manual check already used
  to validate this project's earlier features.
- CI pipeline itself is validated by pushing a small, low-risk change (e.g.
  a comment or README edit) to `main` first and watching it flow end-to-end
  before relying on it for a real change.
