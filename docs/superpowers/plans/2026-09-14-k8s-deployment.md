# Kubernetes Deployment & Real-Host Rollout Implementation Plan

> **Execution mode: mentor, not agentic.** This plan is NOT executed via
> `subagent-driven-development` or `executing-plans`. The user writes every
> file by hand. For each task below: Claude points to study material only
> when a genuinely new concept shows up, states the task's exact
> requirements (files, values, what the resulting resource must do), the
> user implements it, and Claude reviews the result — findings, not fixes.
> Tasks below intentionally do NOT contain ready-to-paste YAML/workflow
> code; they contain the exact values and constraints the user's own
> manifests/scripts must satisfy.

**Goal:** Get the InnoTaxi stack running on a real k3s host, reachable over
HTTPS on a real domain, redeployed automatically on every push to `main`.

**Architecture:** One Helm chart (`deploy/helm/inno-taxi/`) describing all
7 services (Deployments) and 7 stateful dependencies (StatefulSets), an
Ingress with cert-manager-issued TLS, a single Secret for credentials, and
a GitHub Actions workflow that tests, scans, builds/pushes 7 images to
`ghcr.io`, and runs `helm upgrade` against the k3s cluster.

**Tech Stack:** Helm 3, Kubernetes (k3s), Traefik (bundled with k3s),
cert-manager + Let's Encrypt, GitHub Actions, `ghcr.io`.

**Spec:** `docs/superpowers/specs/2026-09-14-k8s-deployment-design.md`

## Global Constraints

- Chart lives at `deploy/helm/inno-taxi/`.
- Every k8s object name uses hyphens, never underscores (k8s `Service`
  names are DNS-1035 labels — underscores are invalid there even though
  Docker Compose's container names use them).
- Every `Deployment`/`StatefulSet`/`Service` name is the **bare literal**
  name (`auth-service`, `redis`, `postgres-order`, ...) — never run through
  the `inno-taxi.fullname` helper. A `Service`'s `metadata.name` *is* its
  DNS hostname, and every env var and `gateway_service/nginx.conf` literal
  in this plan already hardcodes these bare names with no release prefix —
  fullname-prefixing would silently break every one of them. (This is
  fine because the chart only ever runs as a single instance — no
  multi-release-per-namespace scenario to avoid name collisions for.) The
  fullname helper is still correct for the one resource nothing else
  addresses by name: the `Secret` from Task 2.
- All 14 workloads (7 services + 7 stateful deps) run **1 replica** — no
  HA, no autoscaling (Non-Goals in the spec).
- Stateful deps use a plain `ClusterIP` `Service` (not headless) — single
  replica means no per-pod DNS is needed, and this keeps the exact
  hostnames `docker-compose.yaml` already uses.
- Credentials go in one `Secret` (`inno-taxi-secrets`), sourced from
  `values.yaml` — no Vault/sealed-secrets.
- Registry: `ghcr.io/<owner>/<repo>-<service>`, tag = `${{ github.sha }}`.
- Target cluster: k3s (bundled Traefik ingress + `local-path-provisioner`
  storage — never install ingress-nginx or a separate storage provisioner).
- Migrations (`goose`) stay a manual step via `kubectl port-forward` —
  never automated into a Helm hook/Job.
- Nothing in this plan touches business logic — only new manifests/CI, plus
  the two `gateway_service/nginx.conf` fixes named in Tasks 4 and part of
  the chart work (resolver IP, underscore hostnames).

---

### Task 1: Chart skeleton

**Study first:** [Helm Quickstart (RU)](https://helm.sh/ru/docs/intro/quickstart/) —
just enough to know what `Chart.yaml`, `values.yaml`, and `templates/`
each do, and that `{{ .Values.x.y }}` in a template pulls from `values.yaml`.

**Files to create:**
- `deploy/helm/inno-taxi/Chart.yaml`
- `deploy/helm/inno-taxi/values.yaml`
- `deploy/helm/inno-taxi/templates/_helpers.tpl`

**Requirements:**
- `Chart.yaml`: `apiVersion: v2`, `name: inno-taxi`, `type: application`,
  a `version` (chart version, e.g. `0.1.0`) and `appVersion` field.
- `values.yaml` must declare (values can be placeholders like empty string
  for anything task 1 itself doesn't consume yet — later tasks fill in
  their own sections):
  - `domain` (string, empty default — set at install time via `--set`)
  - `image.registry` (default `ghcr.io/<your-github-username-or-org>/inno-taxi`)
  - `image.tag` (default `latest`)
  - A `credentials:` section with these exact keys and these exact default
    values (mirroring `docker-compose.yaml`'s current fallbacks):
    `mongoRootUser: taxi`, `mongoRootPassword: taxi`, `jwtSecret: taxi`,
    `pgOrderUser: postgres`, `pgOrderPass: postgres`,
    `pgWalletUser: postgres`, `pgWalletPass: postgres`,
    `clickhouseUsername: default`, `clickhousePassword: analytics`
- `_helpers.tpl`: at minimum a `inno-taxi.fullname` template (returns
  `<release-name>-<name>`) and a `inno-taxi.labels` template (standard
  `app.kubernetes.io/name`, `app.kubernetes.io/instance`,
  `app.kubernetes.io/managed-by: Helm`) — every resource in every later
  task includes these labels.

**Verify:**
```bash
helm lint deploy/helm/inno-taxi
helm template deploy/helm/inno-taxi
```
Both must succeed with zero resources rendered yet (no `templates/*.yaml`
besides `_helpers.tpl`, which produces no standalone resource on its own).

**Review checkpoint:** show the 3 files. Claude checks `Chart.yaml`
validity, that the helper templates are actually reusable (not hardcoding
a resource name), and that the credential keys match the list above
exactly (later tasks depend on these exact names).

---

### Task 2: Secret template

**Files to create:**
- `deploy/helm/inno-taxi/templates/secrets.yaml`

**Requirements:**
- One `Secret` named via the `inno-taxi.fullname` helper (e.g.
  `<release>-secrets`), `type: Opaque`.
- `stringData` (not `data` — `stringData` takes plain text and lets Helm/k8s
  base64-encode it for you, so you don't have to encode by hand) with keys
  named exactly like the `values.yaml` credential keys from Task 1:
  `mongoRootUser`, `mongoRootPassword`, `jwtSecret`, `pgOrderUser`,
  `pgOrderPass`, `pgWalletUser`, `pgWalletPass`, `clickhouseUsername`,
  `clickhousePassword` — each templated from `.Values.credentials.<key>`.

**Verify:**
```bash
helm template deploy/helm/inno-taxi | grep -A15 "kind: Secret"
```
Confirm every key from `values.yaml`'s `credentials:` section appears, and
none are missing or misspelled.

**Review checkpoint:** show `secrets.yaml` and the rendered output. Claude
checks key names match Task 1 exactly (a mismatch here breaks every later
task's `secretKeyRef`).

---

### Task 3: Worked example — `auth_service` Deployment + Service

**Study first:** what a `Deployment` and a `Service` are and how they
relate (a Deployment manages pods running your container; a Service gives
those pods a stable DNS name + ClusterIP other pods can call) — the 5-minute
video linked earlier, or the "Pods, Deployments, Services" chapter of
whichever Kubernetes intro material you're using.

**Files to create:**
- `deploy/helm/inno-taxi/templates/services/auth-service.yaml`

**Requirements — this is the pattern every later service task repeats:**
- `Deployment` named `auth-service` — the bare literal name, **not** run
  through the `inno-taxi.fullname` helper (see Global Constraints: only
  the Task 2 `Secret` uses that helper) — 1 replica, one container:
  - image: `{{ .Values.image.registry }}/auth-service:{{ .Values.image.tag }}`
  - container port: `8082`
  - env vars (name → value; `<from secret:X>` means `valueFrom.secretKeyRef`
    with `name: <the Task 2 Secret's fullname>` and `key: X`):
    - `HTTP_AUTH_HOST` = `0.0.0.0`
    - `HTTP_AUTH_PORT` = `8082`
    - `REDIS_ADDR` = `redis:6379`
    - `USER_SERVICE_BASE_URL` = `http://user-service:8080`
    - `JWT_SECRET` = `<from secret: jwtSecret>`
- `Service` named `auth-service`, type `ClusterIP`, port `8082` ->
  `targetPort: 8082`, selector matching the Deployment's pod labels.

**Verify:**
```bash
helm template deploy/helm/inno-taxi | grep -A30 "name: .*auth-service"
```
Confirm both a `Deployment` and a `Service` render, env vars match the list
above exactly, and the Secret reference resolves to Task 2's Secret name.

**Review checkpoint:** show `auth-service.yaml`. This is the task Claude
reviews most carefully — every later stateless-service task is "the same
shape as this one," so a mistake here (wrong `secretKeyRef` shape, missing
label, wrong selector) would otherwise get copied 6 more times.

---

### Task 4: Fix `gateway_service/nginx.conf` for Kubernetes

**No study material needed — this is a direct, small edit to an existing
file, not new infrastructure.**

**Files to modify:**
- `services/gateway_service/nginx.conf`

**Requirements (exact, from the spec's findings):**
1. Line 18: `resolver 127.0.0.11 valid=10s;` → `resolver 10.43.0.10 valid=10s;`
   (`10.43.0.10` is k3s's default CoreDNS ClusterIP, deterministic for
   k3s's default `10.43.0.0/16` service CIDR).
2. Every `set $<x>_service "<x>_service:<port>";` line must have its
   **hostname** changed from underscore to hyphen form — the port stays
   the same. Exact replacements, by line content (there are 18 occurrences
   across the file, several repeating):
   - `"auth_service:8082"` → `"auth-service:8082"` (6 occurrences)
   - `"user_service:8080"` → `"user-service:8080"` (3 occurrences)
   - `"driver_service:8081"` → `"driver-service:8081"` (1 occurrence)
   - `"order_service:8080"` → `"order-service:8080"` (7 occurrences)
   - `"analytic_service:8085"` → `"analytic-service:8085"` (1 occurrence)

**Verify:**
```bash
grep -n "127.0.0.11\|_service:" services/gateway_service/nginx.conf
```
Expected: no output at all (every occurrence replaced).

Then confirm nothing else in the repo depends on the old underscore
hostnames for this file specifically:
```bash
grep -rn "auth_service:8082\|user_service:8080\|driver_service:8081\|order_service:8080\|analytic_service:8085" services/gateway_service/
```
Expected: no output.

**Review checkpoint:** show the diff. Claude checks every one of the 18
occurrences was caught (miscounting here silently breaks one specific
route, not all of them, which is easy to miss in manual testing).

---

### Task 5: Remaining 6 stateless services

**Files to create:**
- `deploy/helm/inno-taxi/templates/services/user-service.yaml`
- `deploy/helm/inno-taxi/templates/services/driver-service.yaml`
- `deploy/helm/inno-taxi/templates/services/order-service.yaml`
- `deploy/helm/inno-taxi/templates/services/wallet-service.yaml`
- `deploy/helm/inno-taxi/templates/services/analytic-service.yaml`
- `deploy/helm/inno-taxi/templates/services/gateway-service.yaml`

**Requirements:** same `Deployment` + `Service` pattern as Task 3
(`auth-service.yaml`) — copy the *shape*, not literal content. Exact
per-service values:

**`user-service`** (image `user-service`, port `8080`):
- `HTTP_US_HOST=0.0.0.0`, `HTTP_US_PORT=8080`
- `MONGO_US_HOST=mongo`, `MONGO_US_PORT=27017`, `MONGO_US_DB=taxi`
- `MONGO_US_USERNAME=<from secret: mongoRootUser>`
- `MONGO_US_PASSWORD=<from secret: mongoRootPassword>`
- `JWT_SECRET=<from secret: jwtSecret>`
- `KAFKA_BROKERS=kafka:9092`

**`driver-service`** (image `driver-service`, port `8081`):
- `HTTP_DS_HOST=0.0.0.0`, `HTTP_DS_PORT=8081`
- `MONGO_DS_HOST=mongo`, `MONGO_DS_PORT=27017`, `MONGO_DS_DB=driver_taxi`
- `MONGO_DS_USERNAME=<from secret: mongoRootUser>`
- `MONGO_DS_PASSWORD=<from secret: mongoRootPassword>`
- `JWT_SECRET=<from secret: jwtSecret>`
- `KAFKA_BROKERS=kafka:9092`
- `ANALYTIC_SERVICE_BASE_URL=http://analytic-service:8085`

**`order-service`** (image `order-service`; **two** container ports:
`8080` HTTP + `9082` gRPC — the Service needs two port entries, `http` and
`grpc`, both pointing at this one Deployment):
- `HTTP_ORDER_HOST=0.0.0.0`, `HTTP_ORDER_PORT=8080`
- `GRPC_ORDER_HOST=0.0.0.0`, `GRPC_ORDER_PORT=9082`
- `PG_ORDER_HOST=postgres-order`, `PG_ORDER_PORT=5432`
- `PG_ORDER_USER=<from secret: pgOrderUser>`
- `PG_ORDER_PASS=<from secret: pgOrderPass>`
- `PG_ORDER_DATABASE=order`
- `ES_ORDER_HOST=http://elasticsearch:9200`
- `KAFKA_BROKERS=kafka:9092`
- `JWT_SECRET=<from secret: jwtSecret>`
- `WALLET_SERVICE_BASE_URL=http://wallet-service:8084`
- `DRIVER_SERVICE_BASE_URL=http://driver-service:8081`

**`wallet-service`** (image `wallet-service`, port `8084`):
- `HTTP_WALLET_HOST=0.0.0.0`, `HTTP_WALLET_PORT=8084`
- `PG_WALLET_HOST=postgres-wallet`, `PG_WALLET_PORT=5432`
- `PG_WALLET_USER=<from secret: pgWalletUser>`
- `PG_WALLET_PASS=<from secret: pgWalletPass>`
- `PG_WALLET_DATABASE=wallet`
- `REDIS_ADDR=redis:6379`
- `KAFKA_BROKERS=kafka:9092`

**`analytic-service`** (image `analytic-service`, port `8085`):
- `HTTP_ANALYTIC_HOST=0.0.0.0`, `HTTP_ANALYTIC_PORT=8085`
- `KAFKA_BROKERS=kafka:9092`
- `CLICKHOUSE_ADDR=clickhouse:9000`
- `CLICKHOUSE_DATABASE=analytics`
- `CLICKHOUSE_USERNAME=<from secret: clickhouseUsername>`
- `CLICKHOUSE_PASSWORD=<from secret: clickhousePassword>`

**`gateway-service`** (image built from `services/gateway_service`'s own
Dockerfile — no Go module, so no env vars at all; port `8000`): just a
`Deployment` (image `gateway-service`, container port `8000`, no `env:`
block) + `Service` (port `8000`). This is the only service with zero env
vars — don't add a Secret reference here, there's nothing for it to read.

**Verify (per file, as you finish each one):**
```bash
helm template deploy/helm/inno-taxi | grep -c "^kind: Deployment"
helm template deploy/helm/inno-taxi | grep -c "^kind: Service"
```
After all 6 plus Task 3's `auth-service`, expect `7` Deployments and `7`
Services (the stateful ones aren't templated yet at this point).

**Review checkpoint:** show all 6 files at once (they're mechanically
similar — one review pass across all of them makes more sense than 6
separate ones). Claude checks each service's env-var list against the
table above exactly, and that `order-service`'s two-port Service is
correct (`http`/`grpc` named ports, not just one).

---

### Task 6: Worked example — `redis` StatefulSet + Service + PVC

**Study first:** what a `StatefulSet` and a `PersistentVolumeClaim` are and
why they differ from a `Deployment` (per the spec's Stateful Dependencies
section) — the Slurm evening-school video on Kubernetes storage, or
equivalent, if you want visuals; the core idea (`volumeClaimTemplates` gives
each pod its own disk that survives a restart) is short enough to read from
the k8s docs directly too.

**Files to create:**
- `deploy/helm/inno-taxi/templates/infra/redis.yaml`

**Requirements — this is the pattern every later stateful task repeats:**
- `StatefulSet` named `redis`, 1 replica, `serviceName: redis` (a
  `StatefulSet` must name the governing Service, even a non-headless one),
  container image `redis:7-alpine`, container port `6379`, no env vars.
- `volumeClaimTemplates`: one PVC template, `accessModes: [ReadWriteOnce]`,
  storage request `512Mi` (no `storageClassName` needed — k3s's
  `local-path-provisioner` is already the cluster default).
- Mount the PVC at `/data` (matches `redis-data:/data` in
  `docker-compose.yaml`).
- `Service` named `redis`, `ClusterIP` (not headless — see Global
  Constraints), port `6379`.

**Verify:**
```bash
helm template deploy/helm/inno-taxi | grep -A20 "kind: StatefulSet"
```
Confirm the `volumeClaimTemplates` block is present with the exact size
above, and the container mounts it at `/data`.

**Review checkpoint:** show `redis.yaml`. Same reasoning as Task 3 — every
later `StatefulSet` task copies this shape.

---

### Task 7: Postgres ×2 StatefulSets

**Files to create:**
- `deploy/helm/inno-taxi/templates/infra/postgres-order.yaml`
- `deploy/helm/inno-taxi/templates/infra/postgres-wallet.yaml`

**Requirements:** same shape as Task 6, `image: postgres:18` for both.

**`postgres-order`:**
- env: `POSTGRES_USER=<from secret: pgOrderUser>`,
  `POSTGRES_PASSWORD=<from secret: pgOrderPass>`, `POSTGRES_DB=order`
- container port `5432`
- PVC: `1Gi`, mounted at `/var/lib/postgresql` (matches
  `postgres-data:/var/lib/postgresql` in `docker-compose.yaml` — note this
  is the parent directory, not `/var/lib/postgresql/data`, matching the
  existing compose volume exactly)
- `Service` named `postgres-order`, port `5432`

**`postgres-wallet`:** identical shape, `POSTGRES_USER=<from secret:
pgWalletUser>`, `POSTGRES_PASSWORD=<from secret: pgWalletPass>`,
`POSTGRES_DB=wallet`, `Service`/`StatefulSet` named `postgres-wallet`, same
PVC size and mount path.

**Verify:**
```bash
helm template deploy/helm/inno-taxi | grep -B2 -A10 "POSTGRES_DB"
```
Confirm both StatefulSets render with distinct DBs/credentials and distinct
PVCs (two separate `volumeClaimTemplates`, not one shared between them).

**Review checkpoint:** show both files. Claude checks the two don't
accidentally share a PVC name or Secret key (a copy-paste-and-rename task
like this is exactly where that kind of mistake creeps in).

---

### Task 8: `mongo` StatefulSet

**Files to create:**
- `deploy/helm/inno-taxi/templates/infra/mongo.yaml`

**Requirements:** same shape as Task 6/7. `image: mongo:7`, env
`MONGO_INITDB_ROOT_USERNAME=<from secret: mongoRootUser>`,
`MONGO_INITDB_ROOT_PASSWORD=<from secret: mongoRootPassword>`, container
port `27017`, PVC `1Gi` mounted at `/data/db` (matches `mongo-data:/data/db`),
`Service` named `mongo`, port `27017`.

**Verify:**
```bash
helm template deploy/helm/inno-taxi | grep -A15 "name: .*mongo"
```

**Review checkpoint:** show `mongo.yaml`.

---

### Task 9: `kafka` StatefulSet

**Requirements:** same shape, `image: apache/kafka:3.9.0`, env — copy
these verbatim from `docker-compose.yaml`'s `kafka` service, unchanged
(single-node KRaft mode, broker+controller combined):
```
KAFKA_NODE_ID=1
KAFKA_PROCESS_ROLES=broker,controller
KAFKA_LISTENERS=PLAINTEXT://:9092,CONTROLLER://:9093
KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://kafka:9092
KAFKA_CONTROLLER_LISTENER_NAMES=CONTROLLER
KAFKA_LISTENER_SECURITY_PROTOCOL_MAP=CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT
KAFKA_CONTROLLER_QUORUM_VOTERS=1@kafka:9093
KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=1
KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS=0
KAFKA_TRANSACTION_STATE_LOG_MIN_ISR=1
KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR=1
```
No PVC is strictly required for correctness at this scope (a learning
project, restart just recreates the single-node log from scratch) — but to
match `docker-compose.yaml`'s behavior (which doesn't bind-mount a volume
for kafka today, unlike every other stateful dependency) include one
anyway: PVC `1Gi`, mounted at `/var/lib/kafka/data` (the `apache/kafka`
image's documented default log directory). `Service` named `kafka`, port
`9092` (client) — the controller port `9093` does not need a Service
(nothing outside this single pod ever dials it).

**Verify:**
```bash
helm template deploy/helm/inno-taxi | grep -A20 "name: .*kafka"
```

**Review checkpoint:** show `kafka.yaml`, including whatever data-directory
research you did for the PVC mount path.

---

### Task 10: `clickhouse` StatefulSet + ConfigMap

**Study first:** what a `ConfigMap` is (a Secret's non-sensitive cousin —
also injected as env vars or mounted as files; here it replaces the bind
mount `docker-compose.yaml` currently uses for
`services/analytic_service/clickhouse-listen-ipv4.xml`).

**Files to create:**
- `deploy/helm/inno-taxi/templates/infra/clickhouse-configmap.yaml`
- `deploy/helm/inno-taxi/templates/infra/clickhouse.yaml`

**Requirements:**
- `clickhouse-configmap.yaml`: a `ConfigMap` named `clickhouse-listen-ipv4`
  whose data key `listen-ipv4.xml` holds the **exact current contents** of
  `services/analytic_service/clickhouse-listen-ipv4.xml` (read that file,
  copy its content verbatim into the ConfigMap's data).
- `clickhouse.yaml`: `StatefulSet`, `image: clickhouse/clickhouse-server:24.8-alpine`,
  env `CLICKHOUSE_DB=analytics` (a literal value, not a secret — it's a
  database name, not a credential), `CLICKHOUSE_USER=<from secret:
  clickhouseUsername>`, `CLICKHOUSE_PASSWORD=<from secret:
  clickhousePassword>`. Container ports `8123` (HTTP) and `9000` (native).
  Mount the ConfigMap's `listen-ipv4.xml` key at
  `/etc/clickhouse-server/config.d/listen-ipv4.xml` (read-only) — this is
  the k8s equivalent of `docker-compose.yaml`'s bind mount doing the same
  job. PVC `1Gi`, mounted at `/var/lib/clickhouse`. `Service` named
  `clickhouse`, ports `8123` and `9000`.

**Verify:**
```bash
helm template deploy/helm/inno-taxi | grep -A5 "ConfigMap"
helm template deploy/helm/inno-taxi | grep -B2 -A25 "name: .*clickhouse\b" | grep -v "clickhouse-listen"
```
Confirm the ConfigMap's content matches
`services/analytic_service/clickhouse-listen-ipv4.xml` byte-for-byte, and
the StatefulSet mounts it at the right path.

**Review checkpoint:** show both files, plus confirm you diffed the
ConfigMap content against the source XML file (a transcription error here
reproduces the exact IPv6-listener crash this file exists to prevent).

---

### Task 11: `elasticsearch` StatefulSet

**Requirements:** same shape, `image: docker.elastic.co/elasticsearch/elasticsearch:8.15.3`,
env (copy verbatim from `docker-compose.yaml`):
```
discovery.type=single-node
xpack.security.enabled=false
xpack.security.enrollment.enabled=false
ES_JAVA_OPTS=-Xms512m -Xmx512m
```
Container port `9200`. PVC `2Gi`, mount at
`/usr/share/elasticsearch/data`. `Service` named `elasticsearch`, port
`9200`.

**One prerequisite this pod needs from the cluster, not from this
manifest:** Elasticsearch requires the host kernel's
`vm.max_map_count >= 262144` or it crash-loops on startup — this is set at
the VPS-bootstrap stage (Task 14), not here; note it in your own memory as
a reason a later task exists, no action needed in this task.

**Verify:**
```bash
helm template deploy/helm/inno-taxi | grep -A20 "name: .*elasticsearch"
```

**Review checkpoint:** show `elasticsearch.yaml`.

---

### Task 12: Ingress + TLS

**Study first:** what an `Ingress` object is (an HTTP(S) router in front of
your cluster — one hostname rule here, since all the interesting routing
already lives inside `gateway-service`'s own nginx) and what a
`ClusterIssuer` is (a cert-manager object describing how to get
certificates — installed cluster-wide, not part of this chart). The 5-
minute Helm video and the [Helm Quickstart](https://helm.sh/ru/docs/intro/quickstart/)
already covered templating; for Ingress/cert-manager specifically,
cert-manager's own "ACME HTTP01" tutorial page is the right reference when
you get there — hold off reading it in depth until Task 14, when you
actually install cert-manager into a real cluster.

**Files to create:**
- `deploy/helm/inno-taxi/templates/ingress.yaml`

**Requirements:**
- One `Ingress`, annotation `cert-manager.io/cluster-issuer:
  letsencrypt-prod` (this ClusterIssuer doesn't exist yet — it's created in
  Task 14, but the Ingress can reference it now).
- One rule: host `{{ .Values.domain }}`, path `/` (`pathType: Prefix`),
  backend = the `gateway-service` Service, port `8000`.
- `tls:` block: `hosts: [{{ .Values.domain }}]`, `secretName:
  inno-taxi-tls` (cert-manager creates and populates this Secret itself
  once the challenge succeeds — you don't create it).

**Verify:**
```bash
helm template deploy/helm/inno-taxi --set domain=example.com | grep -A20 "kind: Ingress"
```
Confirm the host and backend render correctly with a test domain value.

**Review checkpoint:** show `ingress.yaml`.

---

### Task 13: Full local chart validation

**No new files — this task is entirely verification, no real cluster or
built images needed yet.**

**Steps:**
1. `helm lint deploy/helm/inno-taxi` — must report zero errors.
2. `helm template deploy/helm/inno-taxi --set domain=example.com > /tmp/rendered.yaml`
   then read through the whole rendered output once, end to end. Confirm:
   - Exactly 7 `Deployment`s and 7 stateless `Service`s (Tasks 3-5).
   - Exactly 7 `StatefulSet`s and 7 stateful `Service`s (Tasks 6-11).
   - Exactly 1 `Secret`, 1 `ConfigMap`, 1 `Ingress`.
   - No two resources of the same `kind` share a `name`.
   - Every `secretKeyRef`/`configMapKeyRef` in the file points at a key
     that actually exists in this same rendered `Secret`/`ConfigMap`.
3. `helm install inno-taxi deploy/helm/inno-taxi --dry-run --set domain=example.com`
   against any reachable cluster context you have (even a throwaway one) —
   this validates the manifests against the real Kubernetes API schema,
   which `helm template` alone does not do.

**Review checkpoint:** report back the counts from step 2 and the dry-run
result. This is the gate before touching the real VPS — Claude reviews the
full picture once, rather than 14 separate times.

---

### Task 14: VPS provisioning + k3s bootstrap

**Study first:** skim the [k3s quickstart](https://docs.k3s.io/quick-start)
and cert-manager's ACME HTTP01 tutorial before starting — this task is a
manual runbook (SSH into a real machine and run commands), not a file you
write in this repo.

**Steps (from the spec's VPS Bootstrap section, exact values):**
1. Provision an Ubuntu 22.04/24.04 VPS, ≥4 vCPU / 8 GB RAM. Note its
   public IP.
2. `curl -sfL https://get.k3s.io | sh -s - --tls-san <that public IP>`
3. `sysctl -w vm.max_map_count=262144` and persist it:
   `echo 'vm.max_map_count=262144' > /etc/sysctl.d/99-elasticsearch.conf`
4. Firewall: open `22`, `80`, `443`, `6443`.
5. Copy `/etc/rancher/k3s/k3s.yaml` off the VPS, replace
   `server: https://127.0.0.1:6443` with
   `server: https://<public IP>:6443`, `base64 -w0` it, and save the result
   somewhere safe locally (this becomes the `KUBE_CONFIG_B64` GitHub
   secret in Task 16 — don't create the GitHub secret yet, just have the
   value ready).
6. Point your chosen domain's A record at the VPS's public IP.
7. `helm install cert-manager jetstack/cert-manager --namespace cert-manager --create-namespace --set installCRDs=true`
   (this is the one Helm chart in this whole project that is NOT `inno-taxi`
   — a platform add-on, installed once, separately). Then create a
   `ClusterIssuer` named `letsencrypt-prod` for Let's Encrypt via HTTP-01 —
   follow cert-manager's own ACME tutorial for the exact object shape,
   since it's a one-time cluster object this plan doesn't need to own.

**Verify:**
```bash
kubectl get nodes          # one Ready node
kubectl get pods -A        # kube-system pods (traefik, coredns, local-path-provisioner) all Running
kubectl get clusterissuer  # letsencrypt-prod present
```

**Review checkpoint:** report the 3 command outputs above and the domain
you chose. Claude reviews the bootstrap is complete before Task 15 does a
real install.

---

### Task 15: First real `helm install` + migrations + smoke test

**Requirements:**
1. Build and push the 7 images by hand once (before CI exists) so there's
   something for the chart to actually run:
   ```bash
   docker build -f services/<service>/Dockerfile -t ghcr.io/<owner>/inno-taxi-<service>:manual-test .
   docker push ghcr.io/<owner>/inno-taxi-<service>:manual-test
   ```
   (repeat for all 7 — `gateway_service`'s context is
   `services/gateway_service`, not repo root, per the spec).
2. `helm install inno-taxi deploy/helm/inno-taxi --set domain=<your real domain> --set image.tag=manual-test`
3. `kubectl get pods` — every pod reaches `Running`/`1/1 Ready`. If any
   Postgres/Mongo/etc. pod isn't ready, `kubectl logs`/`kubectl describe
   pod` it before moving on — don't proceed with broken infra underneath.
4. Migrations: `kubectl port-forward svc/postgres-order 5433:5432` in one
   terminal, then in another, the same `make migrate-order-up` Makefile
   target you already use locally, pointed at `localhost:5433`. Repeat the
   pattern for `postgres-wallet`.
5. Smoke test through the real domain over HTTPS: register a user, then
   create an order (same manual check already used for earlier features in
   this project) — confirms Ingress, TLS, and the gateway routing fixes
   from Task 4 all actually work end-to-end, not just render correctly.

**Review checkpoint:** report `kubectl get pods` output and the smoke-test
result (what you did, what came back).

---

### Task 16: GitHub Actions CI/CD pipeline

**Study first:** GitHub Actions' own "Quickstart" doc for jobs/steps/matrix
syntax if this is your first workflow file — the concepts (jobs run in
parallel unless `needs:` says otherwise, a matrix multiplies one job
definition across a list of values) matter more here than any specific
action's options.

**Files to create:**
- `.github/workflows/deploy.yml`

**Requirements (stages from the spec, in this order):**
1. `test` job, matrix over the 7 service directories: checkout, setup-go,
   `cd services/<service>`, `gofmt -l .` (fail if non-empty output),
   `go vet ./...`, `go build ./...`, `go test ./...`, then
   `go test -tags=integration ./...` (safe to always run — services
   without such tests just report "no test files").
2. `vulncheck` job, matrix over the 7 services: `govulncheck ./...` per
   module.
3. `build-and-push` job, matrix over the 7 services, `needs: [test,
   vulncheck]`, only on push to `main`: `docker build` with the context
   rule from the spec (repo root for the 6 Go services, since they
   `replace` the `shared` module via a relative path;
   `services/gateway_service` for gateway), `trivy image` scan on the
   built image (fail on `HIGH`/`CRITICAL`), then push to
   `ghcr.io/<owner>/inno-taxi-<service>:${{ github.sha }}` and `:latest`,
   authenticating with the built-in `secrets.GITHUB_TOKEN` (needs
   `permissions: packages: write` at the workflow or job level — no extra
   PAT required).
4. `deploy` job, `needs: build-and-push`, only on push to `main`: install
   `kubectl`+`helm`, reconstruct the kubeconfig from the
   `KUBE_CONFIG_B64` GitHub secret you saved in Task 14 (base64-decode it
   to a temp file, `export KUBECONFIG=<that file>`), then
   `helm upgrade --install inno-taxi deploy/helm/inno-taxi -f deploy/helm/inno-taxi/values-prod.yaml --set image.tag=${{ github.sha }} --wait`.
   `values-prod.yaml` is a new small file (create it in this task) holding
   just your real `domain` value — everything else stays the chart's
   defaults from Task 1.
5. Before wiring the `deploy` job for real: add the `KUBE_CONFIG_B64`
   secret to the GitHub repo (Settings → Secrets and variables → Actions),
   using the value you saved in Task 14.

**Verify:**
1. Push a trivial, low-risk change (e.g. a comment) on a feature branch,
   open a PR, and confirm the `test`/`vulncheck` jobs run and pass — `build-
   and-push`/`deploy` should NOT run on a PR (only on `main`).
2. Merge to `main` and watch the full pipeline, including `deploy`, run
   end to end.
3. `kubectl get pods` on the VPS afterward — confirm pods restarted with
   the new image tag (`kubectl get pods -o jsonpath='{.items[*].spec.containers[*].image}'`).
4. Re-run the Task 15 smoke test through the real domain.

**Review checkpoint:** show the workflow file and the Actions run URL/logs
for both the PR run and the `main` run. This is the last task in the plan —
once this is green, the deployment is live and self-updating.
