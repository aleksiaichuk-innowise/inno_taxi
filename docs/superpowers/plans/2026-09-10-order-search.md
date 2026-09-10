# Order Search & Indexing (Elasticsearch) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire `order_service`'s existing (but unused) Elasticsearch connection into a real search feature — orders get indexed as they're created and as their status changes, and a new `SearchOrders` endpoint lets an Analyst filter across all orders by `taxi_type`, `status`, a `created_at` range, and a `price_minor_units` range.

**Architecture:** Synchronous best-effort indexing (`order_service` calls ES right after each Postgres write, logged-not-failed on error — the same convention already used for the Kafka publishes after `CreateOrder`/`RateTrip`), plus one new read-only gRPC-Gateway endpoint gated to the Analyst role entirely by `gateway_service` (no role check in `order_service` itself, matching `analytic_service`'s existing convention).

**Tech Stack:** Go, `github.com/elastic/go-elasticsearch/v8` (already a dependency), gRPC-Gateway/protoc, nginx (`gateway_service`).

**Spec:** `docs/superpowers/specs/2026-09-10-order-search-design.md`

## Global Constraints

- Structured filtering only — no full-text search, no `user_id`/`driver_id` filter, no geo queries. (Spec: "Context" / "Non-goals".)
- `0`/empty string/`_UNSPECIFIED` always means "no filter for this field" — the existing convention in this codebase (`ListOrders`' `limit <= 0`), not proto3 `optional` fields, which nothing here uses. (Spec: "Data flow: search".)
- Every ES write (`IndexOrder`) is best-effort: log via `slog.ErrorContext`, never fail the RPC it's attached to. (Spec: "Write points", "Error handling".)
- `SearchOrders` has no ownership scoping and no server-side role check — role gating is `gateway_service`'s job only. (Spec: "Data flow: search".)
- All new Go code in `services/order_service` — no changes to any other service.

---

### Task 1: Proto — add the `SearchOrders` RPC

**Files:**
- Modify: `shared/proto/order_service/order_service.proto`
- Generated (via `make proto-order`, do not hand-edit): `shared/proto/order_service/order_service.pb.go`, `order_service_grpc.pb.go`, `order_service.pb.gw.go`, `order_service.swagger.json`

**Interfaces:**
- Produces: `order_service.SearchOrdersRequest` (fields: `TaxiType`, `Status`, `CreatedAfter string`, `CreatedBefore string`, `MinPriceMinorUnits int64`, `MaxPriceMinorUnits int64`, `Limit int32`, `Offset int32`), `order_service.SearchOrdersResponse` (fields: `Orders []*Order`, `Total int64`), and `order_service.OrderServiceClient`/`OrderServiceServer`'s new `SearchOrders` method — all consumed by Task 8.

- [ ] **Step 1: Add the new messages and RPC to the proto file**

Open `shared/proto/order_service/order_service.proto`. Add these two messages right after `ListOrdersResponse` (before `message Location`):

```proto
message SearchOrdersRequest {
  TaxiType taxi_type = 1;
  Status status = 2;
  string created_after = 3;
  string created_before = 4;
  int64 min_price_minor_units = 5;
  int64 max_price_minor_units = 6;
  int32 limit = 7;
  int32 offset = 8;
}

message SearchOrdersResponse {
  repeated Order orders = 1;
  int64 total = 2;
}
```

Add the RPC to the `OrderService` service block, right after `ListOrders`:

```proto
  rpc SearchOrders(SearchOrdersRequest) returns (SearchOrdersResponse) {
    option (google.api.http) = {
      get: "/v1/orders/search"
    };
  }
```

- [ ] **Step 2: Regenerate the Go code**

Run: `cd /home/user/projects/InnoTaxi && make proto-order`
Expected: exits 0, and `git status` shows the four generated files under `shared/proto/order_service/` modified (no new files, no deletions).

- [ ] **Step 3: Verify order_service still builds**

Run: `cd services/order_service && go build ./...`
Expected: exits 0 (nothing consumes the new types yet, but generated code must compile standalone).

- [ ] **Step 4: Commit**

```bash
git add shared/proto/order_service/
git commit -m "proto: add SearchOrders RPC to order_service"
```

---

### Task 2: `entity/repository.OrderDocument` + `repository/es_repo` foundations (mapping, `EnsureIndex`)

**Files:**
- Create: `services/order_service/entity/repository/order_document.go`
- Create: `services/order_service/repository/es_repo/repository.go`
- Create: `services/order_service/repository/es_repo/mapping.go`
- Create: `services/order_service/repository/es_repo/ensure_index.go`
- Test: `services/order_service/repository/es_repo/ensure_index_test.go`

**Interfaces:**
- Consumes: `service_dto.Order` (`entity/service/order.go`, already exists).
- Produces: `repository.OrderDocument` struct + `repository.OrderDocumentFromDomain(o service.Order) OrderDocument` + `(OrderDocument) ToDomain() service.Order` — consumed by Task 3 and Task 4. `es_repo.EsRepository` struct + `es_repo.NewEsRepo(client *elasticsearch.Client, index string) EsRepository` — consumed by Task 3, 4, 7. `es_repo.EnsureIndex(ctx context.Context, client *elasticsearch.Client, index string) error` — consumed by Task 7.

- [ ] **Step 1: Write `OrderDocument` and its conversions**

Create `services/order_service/entity/repository/order_document.go`:

```go
package repository

import (
	"time"

	"github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
)

// OrderDocument is the Elasticsearch-facing shape of an Order - json-tagged,
// mirroring how Order (above, in order.go) is db-tagged for Postgres.
type OrderDocument struct {
	ID              string    `json:"id"`
	UserID          string    `json:"user_id"`
	DriverID        string    `json:"driver_id"`
	TaxiType        string    `json:"taxi_type"`
	StartLat        float64   `json:"start_lat"`
	StartLng        float64   `json:"start_lng"`
	DestinationLat  float64   `json:"destination_lat"`
	DestinationLng  float64   `json:"destination_lng"`
	Status          string    `json:"status"`
	PriceMinorUnits int64     `json:"price_minor_units"`
	Rating          int32     `json:"rating"`
	Comment         string    `json:"comment"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func OrderDocumentFromDomain(o service.Order) OrderDocument {
	doc := OrderDocument{
		ID:             o.ID,
		UserID:         o.UserID,
		TaxiType:       string(o.TaxiType),
		StartLat:       o.Start.Lat,
		StartLng:       o.Start.Lng,
		DestinationLat: o.Destination.Lat,
		DestinationLng: o.Destination.Lng,
		Status:         string(o.Status),
		CreatedAt:      o.CreatedAt,
		UpdatedAt:      o.UpdatedAt,
	}
	if o.DriverID != nil {
		doc.DriverID = *o.DriverID
	}
	if o.PriceMinorUnits != nil {
		doc.PriceMinorUnits = *o.PriceMinorUnits
	}
	if o.Rating != nil {
		doc.Rating = *o.Rating
	}
	if o.Comment != nil {
		doc.Comment = *o.Comment
	}
	return doc
}

func (d OrderDocument) ToDomain() service.Order {
	o := service.Order{
		ID:          d.ID,
		UserID:      d.UserID,
		TaxiType:    service.TaxiType(d.TaxiType),
		Start:       service.Location{Lat: d.StartLat, Lng: d.StartLng},
		Destination: service.Location{Lat: d.DestinationLat, Lng: d.DestinationLng},
		Status:      service.Status(d.Status),
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
	if d.DriverID != "" {
		driverID := d.DriverID
		o.DriverID = &driverID
	}
	if d.PriceMinorUnits != 0 {
		price := d.PriceMinorUnits
		o.PriceMinorUnits = &price
	}
	if d.Rating != 0 {
		rating := d.Rating
		o.Rating = &rating
	}
	if d.Comment != "" {
		comment := d.Comment
		o.Comment = &comment
	}
	return o
}
```

- [ ] **Step 2: Write the `EsRepository` type**

Create `services/order_service/repository/es_repo/repository.go`:

```go
package es_repo

import "github.com/elastic/go-elasticsearch/v8"

type EsRepository struct {
	client *elasticsearch.Client
	index  string
}

func NewEsRepo(client *elasticsearch.Client, index string) EsRepository {
	return EsRepository{client: client, index: index}
}
```

- [ ] **Step 3: Write the index mapping**

Create `services/order_service/repository/es_repo/mapping.go`:

```go
package es_repo

// indexMapping is the Elasticsearch index-creation body for the orders
// index - keyword fields are exact-match filtered on, date/long/integer
// fields support range filters, the rest are stored for document fidelity
// only (see the design doc's "Data flow: indexing" mapping table).
const indexMapping = `{
  "mappings": {
    "properties": {
      "id": {"type": "keyword"},
      "user_id": {"type": "keyword"},
      "driver_id": {"type": "keyword"},
      "taxi_type": {"type": "keyword"},
      "status": {"type": "keyword"},
      "price_minor_units": {"type": "long"},
      "rating": {"type": "integer"},
      "comment": {"type": "text"},
      "start_lat": {"type": "double"},
      "start_lng": {"type": "double"},
      "destination_lat": {"type": "double"},
      "destination_lng": {"type": "double"},
      "created_at": {"type": "date"},
      "updated_at": {"type": "date"}
    }
  }
}`
```

- [ ] **Step 4: Write the failing tests for `EnsureIndex`**

Create `services/order_service/repository/es_repo/ensure_index_test.go`:

```go
package es_repo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/elastic/go-elasticsearch/v8"
)

func TestEnsureIndex_CreatesWhenMissing(t *testing.T) {
	var createCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodHead && r.URL.Path == "/orders":
			w.WriteHeader(http.StatusNotFound)
		case r.Method == http.MethodPut && r.URL.Path == "/orders":
			createCalled = true
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	client, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{srv.URL}})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	if err := EnsureIndex(context.Background(), client, "orders"); err != nil {
		t.Fatalf("EnsureIndex() error = %v, want nil", err)
	}
	if !createCalled {
		t.Fatal("expected the index to be created when it doesn't exist")
	}
}

func TestEnsureIndex_SkipsWhenPresent(t *testing.T) {
	var createCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodHead && r.URL.Path == "/orders":
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPut && r.URL.Path == "/orders":
			createCalled = true
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	client, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{srv.URL}})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	if err := EnsureIndex(context.Background(), client, "orders"); err != nil {
		t.Fatalf("EnsureIndex() error = %v, want nil", err)
	}
	if createCalled {
		t.Fatal("expected no create call when the index already exists")
	}
}
```

- [ ] **Step 5: Run the tests to verify they fail**

Run: `cd services/order_service && go test ./repository/es_repo/... -run TestEnsureIndex -v`
Expected: FAIL with `undefined: EnsureIndex`.

- [ ] **Step 6: Implement `EnsureIndex`**

Create `services/order_service/repository/es_repo/ensure_index.go`:

```go
package es_repo

import (
	"context"
	"fmt"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
)

// EnsureIndex creates the given index with this package's mapping if it
// doesn't already exist. Idempotent - safe to call on every startup.
func EnsureIndex(ctx context.Context, client *elasticsearch.Client, index string) error {
	existsRes, err := client.Indices.Exists([]string{index}, client.Indices.Exists.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("check index exists: %w", err)
	}
	defer existsRes.Body.Close()

	if existsRes.StatusCode == 200 {
		return nil
	}
	if existsRes.StatusCode != 404 {
		return fmt.Errorf("check index exists returned unexpected status %d", existsRes.StatusCode)
	}

	createRes, err := client.Indices.Create(index,
		client.Indices.Create.WithContext(ctx),
		client.Indices.Create.WithBody(strings.NewReader(indexMapping)),
	)
	if err != nil {
		return fmt.Errorf("create index: %w", err)
	}
	defer createRes.Body.Close()

	if createRes.IsError() {
		return fmt.Errorf("create index returned unexpected status %d", createRes.StatusCode)
	}
	return nil
}
```

- [ ] **Step 7: Run the tests to verify they pass**

Run: `cd services/order_service && go test ./repository/es_repo/... -v`
Expected: PASS (both `TestEnsureIndex_*` tests).

- [ ] **Step 8: Build the whole service and commit**

Run: `cd services/order_service && go build ./... && go vet ./...`
Expected: exits 0.

```bash
git add services/order_service/entity/repository/order_document.go services/order_service/repository/es_repo/
git commit -m "order_service: add OrderDocument mapping and Elasticsearch index bootstrap"
```

---

### Task 3: `repository/es_repo` — `IndexOrder`

**Files:**
- Create: `services/order_service/repository/es_repo/index_order.go`
- Test: `services/order_service/repository/es_repo/index_order_test.go`

**Interfaces:**
- Consumes: `repository.OrderDocumentFromDomain` (Task 2), `es_repo.EsRepository` (Task 2), `service_dto.Order` (existing).
- Produces: `(EsRepository) IndexOrder(ctx context.Context, order service_dto.Order) error` — consumed by Task 5 (`SearchRepository` interface) and Task 6 (the write points).

- [ ] **Step 1: Write the failing tests**

Create `services/order_service/repository/es_repo/index_order_test.go`:

```go
package es_repo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
	"github.com/elastic/go-elasticsearch/v8"
)

func TestIndexOrder_Success(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result":"created"}`))
	}))
	defer srv.Close()

	client, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{srv.URL}})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	repo := NewEsRepo(client, "orders")

	price := int64(500)
	order := service_dto.Order{
		ID:              "order-1",
		UserID:          "user-1",
		TaxiType:        service_dto.TaxiTypeEconomy,
		Status:          service_dto.StatusCreated,
		PriceMinorUnits: &price,
	}
	if err := repo.IndexOrder(context.Background(), order); err != nil {
		t.Fatalf("IndexOrder() error = %v, want nil", err)
	}

	if gotMethod != http.MethodPut {
		t.Errorf("got method %q, want PUT", gotMethod)
	}
	if gotPath != "/orders/_doc/order-1" {
		t.Errorf("got path %q, want /orders/_doc/order-1", gotPath)
	}
	if gotBody["id"] != "order-1" {
		t.Errorf("got id %v, want order-1", gotBody["id"])
	}
	if gotBody["price_minor_units"] != float64(500) {
		t.Errorf("got price_minor_units %v, want 500", gotBody["price_minor_units"])
	}
}

func TestIndexOrder_UnexpectedStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{srv.URL}})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	repo := NewEsRepo(client, "orders")

	if err := repo.IndexOrder(context.Background(), service_dto.Order{ID: "order-1"}); err == nil {
		t.Fatal("expected an error for an unexpected status code")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd services/order_service && go test ./repository/es_repo/... -run TestIndexOrder -v`
Expected: FAIL with `repo.IndexOrder undefined (type EsRepository has no field or method IndexOrder)`.

- [ ] **Step 3: Implement `IndexOrder`**

Create `services/order_service/repository/es_repo/index_order.go`:

```go
package es_repo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	repo_entity "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/repository"
	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
)

func (r EsRepository) IndexOrder(ctx context.Context, order service_dto.Order) error {
	doc := repo_entity.OrderDocumentFromDomain(order)
	body, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal order document: %w", err)
	}

	res, err := r.client.Index(r.index, bytes.NewReader(body),
		r.client.Index.WithDocumentID(order.ID),
		r.client.Index.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("index order: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("index order returned unexpected status %d", res.StatusCode)
	}
	return nil
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd services/order_service && go test ./repository/es_repo/... -v`
Expected: PASS (all tests in the package, including Task 2's).

- [ ] **Step 5: Commit**

```bash
git add services/order_service/repository/es_repo/index_order.go services/order_service/repository/es_repo/index_order_test.go
git commit -m "order_service: index orders into Elasticsearch on write"
```

---

### Task 4: `entity/service.OrderSearchFilter` + `repository/es_repo` — `SearchOrders`

**Files:**
- Create: `services/order_service/entity/service/search.go`
- Create: `services/order_service/repository/es_repo/search_orders.go`
- Test: `services/order_service/repository/es_repo/search_orders_test.go`

**Interfaces:**
- Consumes: `repo_entity.OrderDocument`/`.ToDomain()` (Task 2), `es_repo.EsRepository` (Task 2).
- Produces: `service.OrderSearchFilter{TaxiType, Status, CreatedAfter, CreatedBefore time.Time, MinPriceMinorUnits, MaxPriceMinorUnits int64, Limit, Offset int32}` — consumed by Task 5 and Task 8. `(EsRepository) SearchOrders(ctx context.Context, filter service_dto.OrderSearchFilter) ([]service_dto.Order, int64, error)` — consumed by Task 5 (`SearchRepository` interface).

- [ ] **Step 1: Write `OrderSearchFilter`**

Create `services/order_service/entity/service/search.go`:

```go
package service

import "time"

type OrderSearchFilter struct {
	TaxiType           TaxiType
	Status             Status
	CreatedAfter       time.Time
	CreatedBefore      time.Time
	MinPriceMinorUnits int64
	MaxPriceMinorUnits int64
	Limit              int32
	Offset             int32
}
```

- [ ] **Step 2: Write the failing tests for `SearchOrders`**

Create `services/order_service/repository/es_repo/search_orders_test.go`:

```go
package es_repo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
	"github.com/elastic/go-elasticsearch/v8"
)

func TestSearchOrders_BuildsFilterClausesFromFilter(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"hits":{"total":{"value":0},"hits":[]}}`))
	}))
	defer srv.Close()

	client, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{srv.URL}})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	repo := NewEsRepo(client, "orders")

	filter := service_dto.OrderSearchFilter{
		TaxiType:           service_dto.TaxiTypeComfort,
		Status:             service_dto.StatusCompleted,
		CreatedAfter:       time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		MinPriceMinorUnits: 100,
		Limit:              20,
	}
	if _, _, err := repo.SearchOrders(context.Background(), filter); err != nil {
		t.Fatalf("SearchOrders() error = %v, want nil", err)
	}

	query, ok := gotBody["query"].(map[string]any)
	if !ok {
		t.Fatalf("expected a query object, got %+v", gotBody)
	}
	boolQuery, ok := query["bool"].(map[string]any)
	if !ok {
		t.Fatalf("expected a bool query, got %+v", query)
	}
	clauses, ok := boolQuery["filter"].([]any)
	if !ok || len(clauses) != 3 {
		t.Fatalf("expected 3 filter clauses (taxi_type, status, price range), got %+v", boolQuery["filter"])
	}
}

func TestSearchOrders_NoFiltersUsesMatchAll(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"hits":{"total":{"value":0},"hits":[]}}`))
	}))
	defer srv.Close()

	client, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{srv.URL}})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	repo := NewEsRepo(client, "orders")

	if _, _, err := repo.SearchOrders(context.Background(), service_dto.OrderSearchFilter{Limit: 20}); err != nil {
		t.Fatalf("SearchOrders() error = %v, want nil", err)
	}

	query, ok := gotBody["query"].(map[string]any)
	if !ok {
		t.Fatalf("expected a query object, got %+v", gotBody)
	}
	if _, ok := query["match_all"]; !ok {
		t.Fatalf("expected match_all with no filters set, got %+v", query)
	}
}

func TestSearchOrders_ParsesResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"hits": {
				"total": {"value": 2},
				"hits": [
					{"_source": {"id":"order-1","user_id":"user-1","taxi_type":"economy","status":"completed","price_minor_units":500,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}}
				]
			}
		}`))
	}))
	defer srv.Close()

	client, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{srv.URL}})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	repo := NewEsRepo(client, "orders")

	orders, total, err := repo.SearchOrders(context.Background(), service_dto.OrderSearchFilter{Limit: 20})
	if err != nil {
		t.Fatalf("SearchOrders() error = %v, want nil", err)
	}
	if total != 2 {
		t.Errorf("got total %d, want 2", total)
	}
	if len(orders) != 1 || orders[0].ID != "order-1" {
		t.Fatalf("unexpected orders: %+v", orders)
	}
}

func TestSearchOrders_UnexpectedStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{srv.URL}})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	repo := NewEsRepo(client, "orders")

	if _, _, err := repo.SearchOrders(context.Background(), service_dto.OrderSearchFilter{Limit: 20}); err == nil {
		t.Fatal("expected an error for an unexpected status code")
	}
}
```

- [ ] **Step 3: Run the tests to verify they fail**

Run: `cd services/order_service && go test ./repository/es_repo/... -run TestSearchOrders -v`
Expected: FAIL with `repo.SearchOrders undefined`.

- [ ] **Step 4: Implement the query builder and `SearchOrders`**

Create `services/order_service/repository/es_repo/search_orders.go`:

```go
package es_repo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	repo_entity "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/repository"
	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
)

// buildSearchQuery builds a filter-only bool query (no scoring needed) from
// the given filter, falling back to match_all when nothing is set - see the
// design doc's "Data flow: search" section.
func buildSearchQuery(filter service_dto.OrderSearchFilter) map[string]any {
	var clauses []map[string]any

	if filter.TaxiType != "" {
		clauses = append(clauses, map[string]any{"term": map[string]any{"taxi_type": string(filter.TaxiType)}})
	}
	if filter.Status != "" {
		clauses = append(clauses, map[string]any{"term": map[string]any{"status": string(filter.Status)}})
	}
	if !filter.CreatedAfter.IsZero() || !filter.CreatedBefore.IsZero() {
		r := map[string]any{}
		if !filter.CreatedAfter.IsZero() {
			r["gte"] = filter.CreatedAfter.Format(time.RFC3339)
		}
		if !filter.CreatedBefore.IsZero() {
			r["lte"] = filter.CreatedBefore.Format(time.RFC3339)
		}
		clauses = append(clauses, map[string]any{"range": map[string]any{"created_at": r}})
	}
	if filter.MinPriceMinorUnits != 0 || filter.MaxPriceMinorUnits != 0 {
		r := map[string]any{}
		if filter.MinPriceMinorUnits != 0 {
			r["gte"] = filter.MinPriceMinorUnits
		}
		if filter.MaxPriceMinorUnits != 0 {
			r["lte"] = filter.MaxPriceMinorUnits
		}
		clauses = append(clauses, map[string]any{"range": map[string]any{"price_minor_units": r}})
	}

	query := map[string]any{"match_all": map[string]any{}}
	if len(clauses) > 0 {
		query = map[string]any{"bool": map[string]any{"filter": clauses}}
	}

	return map[string]any{
		"query": query,
		"sort":  []map[string]any{{"created_at": map[string]any{"order": "desc"}}},
	}
}

type searchResponseBody struct {
	Hits struct {
		Total struct {
			Value int64 `json:"value"`
		} `json:"total"`
		Hits []struct {
			Source repo_entity.OrderDocument `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

func (r EsRepository) SearchOrders(ctx context.Context, filter service_dto.OrderSearchFilter) ([]service_dto.Order, int64, error) {
	body, err := json.Marshal(buildSearchQuery(filter))
	if err != nil {
		return nil, 0, fmt.Errorf("marshal search query: %w", err)
	}

	res, err := r.client.Search(
		r.client.Search.WithContext(ctx),
		r.client.Search.WithIndex(r.index),
		r.client.Search.WithBody(bytes.NewReader(body)),
		r.client.Search.WithFrom(int(filter.Offset)),
		r.client.Search.WithSize(int(filter.Limit)),
		r.client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("search orders: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, 0, fmt.Errorf("search orders returned unexpected status %d", res.StatusCode)
	}

	var parsed searchResponseBody
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return nil, 0, fmt.Errorf("decode search response: %w", err)
	}

	orders := make([]service_dto.Order, len(parsed.Hits.Hits))
	for i, h := range parsed.Hits.Hits {
		orders[i] = h.Source.ToDomain()
	}
	return orders, parsed.Hits.Total.Value, nil
}
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `cd services/order_service && go test ./repository/es_repo/... -v`
Expected: PASS (all tests in the package).

- [ ] **Step 6: Build and commit**

Run: `cd services/order_service && go build ./... && go vet ./...`
Expected: exits 0.

```bash
git add services/order_service/entity/service/search.go services/order_service/repository/es_repo/search_orders.go services/order_service/repository/es_repo/search_orders_test.go
git commit -m "order_service: add Elasticsearch-backed order search"
```

---

### Task 5: Service layer — `SearchRepository` interface + `SearchOrders` method

**Files:**
- Modify: `services/order_service/service/order_service.go`
- Create: `services/order_service/service/search_orders.go`
- Modify: `services/order_service/service/create_order_test.go` (add `fakeSearchRepository`)
- Create: `services/order_service/service/search_orders_test.go`
- Modify: all other `service/*_test.go` files and `app/run.go` (thread the new constructor parameter through)

**Interfaces:**
- Consumes: `service.OrderSearchFilter` (Task 4), `EsRepository.IndexOrder`/`.SearchOrders` (Tasks 3-4, satisfied structurally, not by name).
- Produces: `service.SearchRepository` interface (`IndexOrder(ctx, order) error`, `SearchOrders(ctx, filter) ([]Order, int64, error)`) — consumed by Task 6 (write points) and Task 7 (`app/run.go` wiring). `NewOrderService(repo, gateway, wallet, driver, search SearchRepository) OrderService` — its signature is now 5 params; every call site must pass a 5th argument. `(OrderService) SearchOrders(ctx, filter) ([]Order, int64, error)` — consumed by Task 8 (handler).

- [ ] **Step 1: Add the `SearchRepository` interface and thread it into `OrderService`**

In `services/order_service/service/order_service.go`, add this interface after `DriverGateway`:

```go
// SearchRepository indexes orders into Elasticsearch and serves search
// queries back for the Analyst-only SearchOrders endpoint. IndexOrder is
// best-effort at every call site (see search_orders.go and the six write
// points in create_order.go/cancel_order.go/start_trip.go/complete_trip.go/
// rate_trip.go) - a search-index write failing must never fail the order
// operation it's mirroring.
type SearchRepository interface {
	IndexOrder(ctx context.Context, order service_dto.Order) error
	SearchOrders(ctx context.Context, filter service_dto.OrderSearchFilter) ([]service_dto.Order, int64, error)
}
```

Change the `OrderService` struct and constructor to:

```go
type OrderService struct {
	repo    OrderRepository
	gateway OrderGateway
	wallet  WalletGateway
	driver  DriverGateway
	search  SearchRepository
}

func NewOrderService(repo OrderRepository, gateway OrderGateway, wallet WalletGateway, driver DriverGateway, search SearchRepository) OrderService {
	return OrderService{repo: repo, gateway: gateway, wallet: wallet, driver: driver, search: search}
}
```

- [ ] **Step 2: Verify the build breaks at every call site**

Run: `cd services/order_service && go build ./... 2>&1 | head -50`
Expected: multiple `not enough arguments in call to NewOrderService` / `not enough arguments in call to service.NewOrderService` errors, in `service/*_test.go` and `app/run.go`.

- [ ] **Step 3: Thread a `nil` 5th argument through every test call site**

These are mechanical fixes - every existing call site passes either `&fakeDriverGateway{})` or `driver)` as its last argument today; both need `, nil)` appended (a `nil` `SearchRepository` is safe here because none of these existing tests exercise search or indexing - Task 6 will replace `nil` with `&fakeSearchRepository{}` wherever a test needs it).

Run:
```bash
cd services/order_service
sed -i -E 's/NewOrderService\((.*), &fakeDriverGateway\{\}\)/NewOrderService(\1, \&fakeDriverGateway{}, nil)/' service/*_test.go
sed -i -E 's/NewOrderService\((.*), driver\)/NewOrderService(\1, driver, nil)/' service/*_test.go
```

- [ ] **Step 4: Fix `app/run.go`'s call site**

In `services/order_service/app/run.go`, change:

```go
	orderService := service.NewOrderService(repo, kafkaGateway, walletGateway, driverGateway)
```

to:

```go
	orderService := service.NewOrderService(repo, kafkaGateway, walletGateway, driverGateway, nil)
```

(Task 7 replaces this `nil` with the real Elasticsearch-backed repository.)

- [ ] **Step 5: Verify everything builds and the existing tests still pass**

Run: `cd services/order_service && go build ./... && go test ./...`
Expected: exits 0, all pre-existing tests still PASS (they don't touch `search`, so `nil` is never dereferenced).

- [ ] **Step 6: Add `fakeSearchRepository` to the shared test fakes**

In `services/order_service/service/create_order_test.go`, add this type after `fakeDriverGateway`'s methods (end of file):

```go
type fakeSearchRepository struct {
	indexErr        error
	indexCalled     bool
	indexCalledWith service_dto.Order

	searchOrders     []service_dto.Order
	searchTotal      int64
	searchErr        error
	searchCalledWith service_dto.OrderSearchFilter
}

func (f *fakeSearchRepository) IndexOrder(_ context.Context, order service_dto.Order) error {
	f.indexCalled = true
	f.indexCalledWith = order
	return f.indexErr
}

func (f *fakeSearchRepository) SearchOrders(_ context.Context, filter service_dto.OrderSearchFilter) ([]service_dto.Order, int64, error) {
	f.searchCalledWith = filter
	if f.searchErr != nil {
		return nil, 0, f.searchErr
	}
	return f.searchOrders, f.searchTotal, nil
}
```

- [ ] **Step 7: Write the failing tests for `SearchOrders`**

Create `services/order_service/service/search_orders_test.go`:

```go
package service

import (
	"context"
	"testing"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
)

func TestSearchOrders_DefaultsLimitWhenUnset(t *testing.T) {
	search := &fakeSearchRepository{}
	svc := NewOrderService(&fakeOrderRepository{}, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{}, search)

	if _, _, err := svc.SearchOrders(context.Background(), service_dto.OrderSearchFilter{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if search.searchCalledWith.Limit != defaultListOrdersLimit {
		t.Fatalf("got limit %d, want default %d", search.searchCalledWith.Limit, defaultListOrdersLimit)
	}
}

func TestSearchOrders_CapsLimitAtMax(t *testing.T) {
	search := &fakeSearchRepository{}
	svc := NewOrderService(&fakeOrderRepository{}, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{}, search)

	if _, _, err := svc.SearchOrders(context.Background(), service_dto.OrderSearchFilter{Limit: 1000}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if search.searchCalledWith.Limit != maxListOrdersLimit {
		t.Fatalf("got limit %d, want cap %d", search.searchCalledWith.Limit, maxListOrdersLimit)
	}
}

func TestSearchOrders_NegativeOffsetClampedToZero(t *testing.T) {
	search := &fakeSearchRepository{}
	svc := NewOrderService(&fakeOrderRepository{}, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{}, search)

	if _, _, err := svc.SearchOrders(context.Background(), service_dto.OrderSearchFilter{Offset: -5}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if search.searchCalledWith.Offset != 0 {
		t.Fatalf("got offset %d, want 0", search.searchCalledWith.Offset)
	}
}

func TestSearchOrders_PassesThroughFilterAndResults(t *testing.T) {
	want := []service_dto.Order{{ID: "order-1"}}
	search := &fakeSearchRepository{searchOrders: want, searchTotal: 1}
	svc := NewOrderService(&fakeOrderRepository{}, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{}, search)

	filter := service_dto.OrderSearchFilter{TaxiType: service_dto.TaxiTypeComfort, Limit: 10}
	orders, total, err := svc.SearchOrders(context.Background(), filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 || len(orders) != 1 || orders[0].ID != "order-1" {
		t.Fatalf("unexpected results: orders=%+v total=%d", orders, total)
	}
	if search.searchCalledWith.TaxiType != service_dto.TaxiTypeComfort {
		t.Fatalf("expected the filter to be passed through, got %+v", search.searchCalledWith)
	}
}
```

- [ ] **Step 8: Run the tests to verify they fail**

Run: `cd services/order_service && go test ./service/... -run TestSearchOrders -v`
Expected: FAIL with `svc.SearchOrders undefined`.

- [ ] **Step 9: Implement `SearchOrders`**

Create `services/order_service/service/search_orders.go`:

```go
package service

import (
	"context"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
)

// SearchOrders is an Analyst-only capability: no ownership scoping and no
// role check here, unlike ListOrders - role gating for this endpoint lives
// entirely in gateway_service (see the design doc's "Data flow: search").
func (s OrderService) SearchOrders(ctx context.Context, filter service_dto.OrderSearchFilter) ([]service_dto.Order, int64, error) {
	switch {
	case filter.Limit <= 0:
		filter.Limit = defaultListOrdersLimit
	case filter.Limit > maxListOrdersLimit:
		filter.Limit = maxListOrdersLimit
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	return s.search.SearchOrders(ctx, filter)
}
```

- [ ] **Step 10: Run the tests to verify they pass**

Run: `cd services/order_service && go test ./... `
Expected: PASS, all packages.

- [ ] **Step 11: Commit**

```bash
git add services/order_service/service/ services/order_service/app/run.go
git commit -m "order_service: add SearchRepository interface and SearchOrders service method"
```

---

### Task 6: Instrument the six write points with best-effort `IndexOrder`

**Files:**
- Modify: `services/order_service/service/create_order.go`
- Modify: `services/order_service/service/cancel_order.go`
- Modify: `services/order_service/service/start_trip.go`
- Modify: `services/order_service/service/complete_trip.go`
- Modify: `services/order_service/service/rate_trip.go`
- Modify: `services/order_service/service/create_order_test.go`, `cancel_order_test.go`, `start_trip_test.go`, `complete_trip_test.go`, `rate_trip_test.go`

**Interfaces:**
- Consumes: `s.search.IndexOrder(ctx, order) error` (Task 5's `SearchRepository` field), `fakeSearchRepository` (Task 5).
- Produces: nothing new consumed by later tasks - this is the last service-layer task.

- [ ] **Step 1: Write the failing tests for `CreateOrder`**

In `services/order_service/service/create_order_test.go`, add after `TestCreateOrder_PublishFailureDoesNotFailCreateOrder` (or any existing test in that file):

```go
func TestCreateOrder_IndexesOrderAfterPersisting(t *testing.T) {
	created := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusCreated}
	repo := &fakeOrderRepository{order: &created}
	search := &fakeSearchRepository{}
	svc := NewOrderService(repo, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{}, search)

	_, err := svc.CreateOrder(context.Background(), validInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !search.indexCalled {
		t.Fatal("expected the order to be indexed after being persisted")
	}
	if search.indexCalledWith.ID != "order-1" {
		t.Fatalf("expected the persisted order to be indexed, got %+v", search.indexCalledWith)
	}
}

func TestCreateOrder_IndexFailureDoesNotFailCreateOrder(t *testing.T) {
	created := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusCreated}
	repo := &fakeOrderRepository{order: &created}
	search := &fakeSearchRepository{indexErr: errors.New("elasticsearch unreachable")}
	svc := NewOrderService(repo, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{}, search)

	_, err := svc.CreateOrder(context.Background(), validInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

- [ ] **Step 2: Run to verify they fail**

Run: `cd services/order_service && go test ./service/... -run TestCreateOrder_Index -v`
Expected: FAIL (`search.indexCalled` is `false` - nothing calls `IndexOrder` yet).

- [ ] **Step 3: Instrument `CreateOrder` and `tryAssignDriver`**

In `services/order_service/service/create_order.go`, change:

```go
	order, err := s.repo.CreateOrder(ctx, orderID.String(), price, input)
	if err != nil {
		// The rider was already charged but the order was never persisted -
		// refund so they aren't left paying for nothing. Best-effort: this is
		// the one step here that can't be made atomic with the write it's
		// compensating for (an HTTP call and a Postgres insert can't share a
		// transaction), same accepted trade-off as the Kafka publish below.
		if refundErr := s.wallet.Refund(ctx, input.UserID, price, orderID.String()); refundErr != nil {
			slog.ErrorContext(ctx, "refund after failed order insert failed", "order_id", orderID.String(), "error", refundErr)
		}
		return service_dto.Order{}, err
	}

	order = s.tryAssignDriver(ctx, order)
```

to:

```go
	order, err := s.repo.CreateOrder(ctx, orderID.String(), price, input)
	if err != nil {
		// The rider was already charged but the order was never persisted -
		// refund so they aren't left paying for nothing. Best-effort: this is
		// the one step here that can't be made atomic with the write it's
		// compensating for (an HTTP call and a Postgres insert can't share a
		// transaction), same accepted trade-off as the Kafka publish below.
		if refundErr := s.wallet.Refund(ctx, input.UserID, price, orderID.String()); refundErr != nil {
			slog.ErrorContext(ctx, "refund after failed order insert failed", "order_id", orderID.String(), "error", refundErr)
		}
		return service_dto.Order{}, err
	}

	if err := s.search.IndexOrder(ctx, order); err != nil {
		slog.ErrorContext(ctx, "index order in search failed", "order_id", order.ID, "error", err)
	}

	order = s.tryAssignDriver(ctx, order)
```

Then, in the same file, change `tryAssignDriver`'s success path from:

```go
	assigned, err := s.repo.AssignDriver(ctx, order.ID, driverID)
	if err != nil {
		slog.ErrorContext(ctx, "assign driver to order failed", "order_id", order.ID, "driver_id", driverID, "error", err)
		if releaseErr := s.driver.ReleaseDriver(ctx, driverID); releaseErr != nil {
			slog.ErrorContext(ctx, "release driver after failed assignment failed", "order_id", order.ID, "driver_id", driverID, "error", releaseErr)
		}
		return order
	}

	return assigned
}
```

to:

```go
	assigned, err := s.repo.AssignDriver(ctx, order.ID, driverID)
	if err != nil {
		slog.ErrorContext(ctx, "assign driver to order failed", "order_id", order.ID, "driver_id", driverID, "error", err)
		if releaseErr := s.driver.ReleaseDriver(ctx, driverID); releaseErr != nil {
			slog.ErrorContext(ctx, "release driver after failed assignment failed", "order_id", order.ID, "driver_id", driverID, "error", releaseErr)
		}
		return order
	}

	if err := s.search.IndexOrder(ctx, assigned); err != nil {
		slog.ErrorContext(ctx, "index order in search failed", "order_id", assigned.ID, "error", err)
	}

	return assigned
}
```

- [ ] **Step 4: Run to verify the `CreateOrder` tests pass**

Run: `cd services/order_service && go test ./service/... -run TestCreateOrder -v`
Expected: PASS, all `TestCreateOrder_*` tests.

- [ ] **Step 5: Write the failing tests for `CancelOrder`**

In `services/order_service/service/cancel_order_test.go`, add:

```go
func TestCancelOrder_IndexesOrderAfterCancelling(t *testing.T) {
	order := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusCreated}
	cancelled := order
	cancelled.Status = service_dto.StatusCancelled
	repo := &fakeOrderRepository{getOrder: &order, updateOrder: &cancelled}
	search := &fakeSearchRepository{}
	svc := NewOrderService(repo, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{}, search)

	_, err := svc.CancelOrder(context.Background(), "order-1", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !search.indexCalled || search.indexCalledWith.Status != service_dto.StatusCancelled {
		t.Fatalf("expected the cancelled order to be indexed, got called=%v order=%+v", search.indexCalled, search.indexCalledWith)
	}
}

func TestCancelOrder_IndexFailureDoesNotFailCancelOrder(t *testing.T) {
	order := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusCreated}
	cancelled := order
	cancelled.Status = service_dto.StatusCancelled
	repo := &fakeOrderRepository{getOrder: &order, updateOrder: &cancelled}
	search := &fakeSearchRepository{indexErr: errors.New("elasticsearch unreachable")}
	svc := NewOrderService(repo, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{}, search)

	_, err := svc.CancelOrder(context.Background(), "order-1", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

Check the top of `cancel_order_test.go` already imports `"errors"` and `service_dto "..."` (it does, for the existing tests) - no new imports needed.

- [ ] **Step 6: Instrument `CancelOrder`**

In `services/order_service/service/cancel_order.go`, change the final line:

```go
	return s.repo.UpdateOrderStatus(ctx, orderID, service_dto.StatusCancelled)
}
```

to:

```go
	updated, err := s.repo.UpdateOrderStatus(ctx, orderID, service_dto.StatusCancelled)
	if err != nil {
		return service_dto.Order{}, err
	}

	if err := s.search.IndexOrder(ctx, updated); err != nil {
		slog.ErrorContext(ctx, "index order in search failed", "order_id", updated.ID, "error", err)
	}

	return updated, nil
}
```

(`log/slog` is already imported in this file for the driver-release logging above.)

- [ ] **Step 7: Run to verify `CancelOrder` tests pass**

Run: `cd services/order_service && go test ./service/... -run TestCancelOrder -v`
Expected: PASS, all `TestCancelOrder_*` tests.

- [ ] **Step 8: Write the failing tests for `StartTrip`**

In `services/order_service/service/start_trip_test.go`, add:

```go
func TestStartTrip_IndexesOrderAfterStarting(t *testing.T) {
	driverID := "driver-1"
	order := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusDriverAssigned, DriverID: &driverID}
	started := order
	started.Status = service_dto.StatusInProgress
	repo := &fakeOrderRepository{getOrder: &order, updateOrder: &started}
	search := &fakeSearchRepository{}
	svc := NewOrderService(repo, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{}, search)

	_, err := svc.StartTrip(context.Background(), "order-1", "driver-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !search.indexCalled || search.indexCalledWith.Status != service_dto.StatusInProgress {
		t.Fatalf("expected the started order to be indexed, got called=%v order=%+v", search.indexCalled, search.indexCalledWith)
	}
}

func TestStartTrip_IndexFailureDoesNotFailStartTrip(t *testing.T) {
	driverID := "driver-1"
	order := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusDriverAssigned, DriverID: &driverID}
	started := order
	started.Status = service_dto.StatusInProgress
	repo := &fakeOrderRepository{getOrder: &order, updateOrder: &started}
	search := &fakeSearchRepository{indexErr: errors.New("elasticsearch unreachable")}
	svc := NewOrderService(repo, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{}, search)

	_, err := svc.StartTrip(context.Background(), "order-1", "driver-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

Check `start_trip_test.go` already imports `"errors"` (it does, for its existing `errors.Is`-based tests) - no new imports needed.

- [ ] **Step 9: Instrument `StartTrip`**

In `services/order_service/service/start_trip.go`, change the final line:

```go
	return s.repo.UpdateOrderStatus(ctx, orderID, service_dto.StatusInProgress)
}
```

to:

```go
	updated, err := s.repo.UpdateOrderStatus(ctx, orderID, service_dto.StatusInProgress)
	if err != nil {
		return service_dto.Order{}, err
	}

	if err := s.search.IndexOrder(ctx, updated); err != nil {
		slog.ErrorContext(ctx, "index order in search failed", "order_id", updated.ID, "error", err)
	}

	return updated, nil
}
```

Add `"log/slog"` to this file's import block (it isn't imported there yet, unlike `cancel_order.go`/`complete_trip.go`).

- [ ] **Step 10: Run to verify `StartTrip` tests pass**

Run: `cd services/order_service && go test ./service/... -run TestStartTrip -v`
Expected: PASS, all `TestStartTrip_*` tests.

- [ ] **Step 11: Write the failing tests for `CompleteTrip`**

In `services/order_service/service/complete_trip_test.go`, add:

```go
func TestCompleteTrip_IndexesOrderAfterCompleting(t *testing.T) {
	driverID := "driver-1"
	order := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusInProgress, DriverID: &driverID}
	completed := order
	completed.Status = service_dto.StatusCompleted
	repo := &fakeOrderRepository{getOrder: &order, updateOrder: &completed}
	search := &fakeSearchRepository{}
	svc := NewOrderService(repo, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{}, search)

	_, err := svc.CompleteTrip(context.Background(), "order-1", "driver-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !search.indexCalled || search.indexCalledWith.Status != service_dto.StatusCompleted {
		t.Fatalf("expected the completed order to be indexed, got called=%v order=%+v", search.indexCalled, search.indexCalledWith)
	}
}

func TestCompleteTrip_IndexFailureDoesNotFailCompleteTrip(t *testing.T) {
	driverID := "driver-1"
	order := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusInProgress, DriverID: &driverID}
	completed := order
	completed.Status = service_dto.StatusCompleted
	repo := &fakeOrderRepository{getOrder: &order, updateOrder: &completed}
	search := &fakeSearchRepository{indexErr: errors.New("elasticsearch unreachable")}
	svc := NewOrderService(repo, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{}, search)

	_, err := svc.CompleteTrip(context.Background(), "order-1", "driver-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

Check `complete_trip_test.go` already imports `"errors"` (it does, for `TestCompleteTrip_NotFound`-style tests using `errors.Is`) - no new imports needed.

- [ ] **Step 12: Instrument `CompleteTrip`**

In `services/order_service/service/complete_trip.go`, change the final line:

```go
	return s.repo.UpdateOrderStatus(ctx, orderID, service_dto.StatusCompleted)
}
```

to:

```go
	updated, err := s.repo.UpdateOrderStatus(ctx, orderID, service_dto.StatusCompleted)
	if err != nil {
		return service_dto.Order{}, err
	}

	if err := s.search.IndexOrder(ctx, updated); err != nil {
		slog.ErrorContext(ctx, "index order in search failed", "order_id", updated.ID, "error", err)
	}

	return updated, nil
}
```

(`log/slog` is already imported in this file for the driver-release logging above.)

- [ ] **Step 13: Run to verify `CompleteTrip` tests pass**

Run: `cd services/order_service && go test ./service/... -run TestCompleteTrip -v`
Expected: PASS, all `TestCompleteTrip_*` tests.

- [ ] **Step 14: Write the failing tests for `RateTrip`**

In `services/order_service/service/rate_trip_test.go`, add:

```go
func TestRateTrip_IndexesOrderAfterRating(t *testing.T) {
	order := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusCompleted}
	rating := int32(5)
	rated := order
	rated.Rating = &rating
	repo := &fakeOrderRepository{getOrder: &order, ratedOrder: &rated}
	search := &fakeSearchRepository{}
	svc := NewOrderService(repo, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{}, search)

	_, err := svc.RateTrip(context.Background(), "order-1", "user-1", 5, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !search.indexCalled || search.indexCalledWith.ID != "order-1" {
		t.Fatalf("expected the rated order to be indexed, got called=%v order=%+v", search.indexCalled, search.indexCalledWith)
	}
}

func TestRateTrip_IndexFailureDoesNotFailRateTrip(t *testing.T) {
	order := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusCompleted}
	rating := int32(5)
	rated := order
	rated.Rating = &rating
	repo := &fakeOrderRepository{getOrder: &order, ratedOrder: &rated}
	search := &fakeSearchRepository{indexErr: errors.New("elasticsearch unreachable")}
	svc := NewOrderService(repo, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{}, search)

	_, err := svc.RateTrip(context.Background(), "order-1", "user-1", 5, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

Check `rate_trip_test.go` already imports `"errors"` (used by its existing `errors.Is` tests) - no new imports needed.

- [ ] **Step 15: Instrument `RateTrip`**

In `services/order_service/service/rate_trip.go`, change:

```go
	rated, err := s.repo.RateOrder(ctx, orderID, rating, comment)
	if err != nil {
		return service_dto.Order{}, err
	}

	// Best-effort and post-commit, same trade-off as order_created's publish.
	if err := s.gateway.PublishOrderRated(ctx, rated); err != nil {
```

to:

```go
	rated, err := s.repo.RateOrder(ctx, orderID, rating, comment)
	if err != nil {
		return service_dto.Order{}, err
	}

	if err := s.search.IndexOrder(ctx, rated); err != nil {
		slog.ErrorContext(ctx, "index order in search failed", "order_id", rated.ID, "error", err)
	}

	// Best-effort and post-commit, same trade-off as order_created's publish.
	if err := s.gateway.PublishOrderRated(ctx, rated); err != nil {
```

- [ ] **Step 16: Run all service tests to verify everything passes**

Run: `cd services/order_service && go test ./... && go vet ./...`
Expected: PASS, all packages, no vet issues.

- [ ] **Step 17: Commit**

```bash
git add services/order_service/service/
git commit -m "order_service: index orders on every status-changing operation"
```

---

### Task 7: Wire the real Elasticsearch repository into `app/run.go`

**Files:**
- Modify: `services/order_service/config/config.go`
- Modify: `services/order_service/app/run.go`

**Interfaces:**
- Consumes: `es_repo.EnsureIndex` (Task 2), `es_repo.NewEsRepo` (Task 2), `service.NewOrderService`'s 5th parameter (Task 5).
- Produces: nothing new consumed by later tasks.

- [ ] **Step 1: Add the index name to config**

In `services/order_service/config/config.go`, add a field to `Config`:

```go
type Config struct {
	DbConn        shared.PostgresConfig
	EsConn        shared.ElasticsearchConfig
	EsIndex       string
	KafkaConf     shared.KafkaConfig
	Grpc          *shared.GrpcServerConfig
	HttpHost      shared.HttpHostConfig
	JWT           *JWTConfig
	WalletService *WalletServiceConfig
	DriverService *DriverServiceConfig
}
```

And in `Load()`, right after the `EsConn: shared.ElasticsearchConfig{...}` block, add:

```go
		EsIndex: shared.GetEnvFallback("ES_ORDER_INDEX", "orders"),
```

- [ ] **Step 2: Wire `EnsureIndex` and `NewEsRepo` into `app/run.go`**

In `services/order_service/app/run.go`, add the import:

```go
	"github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/repository/es_repo"
```

Right after the existing:

```go
	es, err := elastic.NewESClient(ctx, cfg.EsConn)
	if err != nil {
		return err
	}
	defer es.Close(ctx)
```

add:

```go
	if err := es_repo.EnsureIndex(ctx, es, cfg.EsIndex); err != nil {
		return fmt.Errorf("ensure elasticsearch index: %w", err)
	}
	search := es_repo.NewEsRepo(es, cfg.EsIndex)
```

Then change the `NewOrderService` call from:

```go
	orderService := service.NewOrderService(repo, kafkaGateway, walletGateway, driverGateway, nil)
```

to:

```go
	orderService := service.NewOrderService(repo, kafkaGateway, walletGateway, driverGateway, search)
```

- [ ] **Step 3: Build and vet**

Run: `cd services/order_service && go build ./... && go vet ./...`
Expected: exits 0.

- [ ] **Step 4: Run the full test suite**

Run: `cd services/order_service && go test ./...`
Expected: PASS, all packages (this task touches no test files, so this is a regression check).

- [ ] **Step 5: Commit**

```bash
git add services/order_service/config/config.go services/order_service/app/run.go
git commit -m "order_service: wire Elasticsearch index bootstrap and search repository into startup"
```

---

### Task 8: `handler/grpc` — `SearchOrders` method and mapping helpers

**Files:**
- Modify: `services/order_service/handler/grpc/mapping.go`
- Modify: `services/order_service/handler/grpc/server.go`

**Interfaces:**
- Consumes: `order_service.SearchOrdersRequest`/`SearchOrdersResponse` (Task 1), `service.OrderSearchFilter` (Task 4), `OrderServer.svc.SearchOrders` (Task 5).
- Produces: the finished, gateway-reachable `POST`-free `GET /v1/orders/search` HTTP route (via gRPC-Gateway) - consumed by Task 9 (nginx) and Task 10 (verification).

- [ ] **Step 1: Add `statusFromProto` and `searchFilterFromProto` to `mapping.go`**

In `services/order_service/handler/grpc/mapping.go`, add `"fmt"` and `"time"` are already imported (`"time"` is; add `"fmt"`), then append:

```go
func statusFromProto(s order_service.Status) service_dto.Status {
	switch s {
	case order_service.Status_STATUS_CREATED:
		return service_dto.StatusCreated
	case order_service.Status_STATUS_DRIVER_ASSIGNED:
		return service_dto.StatusDriverAssigned
	case order_service.Status_STATUS_IN_PROGRESS:
		return service_dto.StatusInProgress
	case order_service.Status_STATUS_COMPLETED:
		return service_dto.StatusCompleted
	case order_service.Status_STATUS_CANCELLED:
		return service_dto.StatusCancelled
	default:
		return ""
	}
}

// searchFilterFromProto converts the wire request to the domain filter,
// parsing the RFC3339 date-range strings here (not in the service) because
// a malformed string is a parse failure, not a business-rule validation -
// see the design doc's "Data flow: search" section.
func searchFilterFromProto(req *order_service.SearchOrdersRequest) (service_dto.OrderSearchFilter, error) {
	filter := service_dto.OrderSearchFilter{
		TaxiType:           taxiTypeFromProto(req.GetTaxiType()),
		Status:             statusFromProto(req.GetStatus()),
		MinPriceMinorUnits: req.GetMinPriceMinorUnits(),
		MaxPriceMinorUnits: req.GetMaxPriceMinorUnits(),
		Limit:              req.GetLimit(),
		Offset:             req.GetOffset(),
	}

	if s := req.GetCreatedAfter(); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return service_dto.OrderSearchFilter{}, fmt.Errorf("invalid created_after: %w", err)
		}
		filter.CreatedAfter = t
	}
	if s := req.GetCreatedBefore(); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return service_dto.OrderSearchFilter{}, fmt.Errorf("invalid created_before: %w", err)
		}
		filter.CreatedBefore = t
	}

	return filter, nil
}
```

- [ ] **Step 2: Add the `SearchOrders` method to `server.go`**

In `services/order_service/handler/grpc/server.go`, add at the end of the file (after `ListOrders`):

```go
// SearchOrders has no ownership scoping and no role check here, unlike
// every other endpoint in this file - it's an Analyst-only endpoint and
// role gating for it lives entirely in gateway_service (see the design
// doc's "Data flow: search"). It does not call
// interceptor.UserIDFromContext because nothing here needs the caller's
// identity.
func (o OrderServer) SearchOrders(ctx context.Context, req *order_service.SearchOrdersRequest) (*order_service.SearchOrdersResponse, error) {
	filter, err := searchFilterFromProto(req)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	orders, total, err := o.svc.SearchOrders(ctx, filter)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	protoOrders := make([]*order_service.Order, len(orders))
	for i, ord := range orders {
		protoOrders[i] = orderToProto(ord)
	}

	return &order_service.SearchOrdersResponse{
		Orders: protoOrders,
		Total:  total,
	}, nil
}
```

- [ ] **Step 3: Build, vet, and test**

Run: `cd services/order_service && go build ./... && go vet ./... && go test ./...`
Expected: exits 0, all tests PASS (no new tests in this task - `handler/grpc` has no existing test file convention in this service, per its current `[no test files]` state; correctness here is covered by the service-layer tests plus Task 10's manual verification).

- [ ] **Step 4: Commit**

```bash
git add services/order_service/handler/grpc/
git commit -m "order_service: add SearchOrders gRPC handler"
```

---

### Task 9: `gateway_service` — route `/orders/search` to the Analyst role

**Files:**
- Modify: `services/gateway_service/nginx.conf`

**Interfaces:**
- Consumes: the `/v1/orders/search` route (Task 8), the existing `/_gateway/validate/analyst` internal location (already in `nginx.conf`, used by `/analytics/`).
- Produces: the externally-reachable `GET /orders/search` route - consumed by Task 10.

- [ ] **Step 1: Add the new location block**

In `services/gateway_service/nginx.conf`, add this block right before the `# Single-order view (GET /orders/{id})` comment (i.e., before the `location ~ ^/orders/(?<order_id>[^/]+)$` block, so the two `/orders/*` regions read top-to-bottom in the file - exact-match locations win regardless of file order, so this ordering is for readability, not correctness):

```nginx
        # Analyst-only search across all orders - an exact-match location,
        # so nginx selects it over the ^/orders/(?<order_id>[^/]+)$ regex
        # below regardless of file order (exact matches are always checked
        # first). $is_args$args is required for the same reason it was
        # added to /orders - a literal proxy_pass rewrite URI doesn't
        # forward the query string on its own, and every filter here rides
        # in the query string.
        location = /orders/search {
            set $order_service "order_service:8080";
            auth_request /_gateway/validate/analyst;
            auth_request_set $auth_user_id $upstream_http_x_user_id;
            proxy_set_header X-User-Id $auth_user_id;

            proxy_pass http://$order_service/v1/orders/search$is_args$args;
        }

```

- [ ] **Step 2: Validate the nginx config syntax**

Run: `docker run --rm -v /home/user/projects/InnoTaxi/services/gateway_service/nginx.conf:/etc/nginx/conf.d/default.conf:ro nginx:1.27-alpine nginx -t`
Expected: `nginx: configuration file /etc/nginx/nginx.conf test is successful` (this only checks syntax - `auth_request`'s target existing and upstream names resolving are checked in Task 10, against the real compose network).

- [ ] **Step 3: Validate the compose file as a whole**

Run: `cd /home/user/projects/InnoTaxi && docker compose config -q`
Expected: exits 0.

- [ ] **Step 4: Commit**

```bash
git add services/gateway_service/nginx.conf
git commit -m "gateway_service: route GET /orders/search to the Analyst role"
```

---

### Task 10: Final verification

**Files:** none (verification only).

**Interfaces:** none produced - this is the plan's last task.

- [ ] **Step 1: Run every order_service test**

Run: `cd services/order_service && go build ./... && go vet ./... && go test ./...`
Expected: exits 0, all packages PASS.

- [ ] **Step 2: Run the repository-wide Makefile targets**

Run: `cd /home/user/projects/InnoTaxi && make build-order && make test-order`
Expected: both exit 0.

- [ ] **Step 3: Validate compose config once more (belt-and-suspenders after all tasks)**

Run: `cd /home/user/projects/InnoTaxi && docker compose config -q`
Expected: exits 0.

- [ ] **Step 4: (Optional, manual) Live smoke test**

This step needs a running stack and is not part of the automated test suite - run it if you want end-to-end confidence beyond the unit tests above:

```bash
cd /home/user/projects/InnoTaxi
docker compose up -d postgres elasticsearch kafka order_service
# wait for order_service to report healthy / for its logs to show a clean startup
docker compose logs order_service --tail 50
```

Confirm the startup log shows no `ensure elasticsearch index` error, then create an order through the normal flow (`docker-compose`'s existing user/auth/order path) and query Elasticsearch directly to confirm it was indexed:

```bash
curl -s http://localhost:9200/orders/_search?pretty
```

Expected: the created order appears under `hits.hits`.

- [ ] **Step 5: Final commit (if any verification step required a fix)**

If Step 4 surfaced a bug, fix it, re-run the relevant task's tests, and commit the fix with a message describing what was wrong (e.g. `fix: <what broke and why>`). If nothing needed fixing, there is nothing to commit here - Task 9's commit is the plan's last one.
