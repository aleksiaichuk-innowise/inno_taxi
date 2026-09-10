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
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
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
	if !ok || len(clauses) != 4 {
		t.Fatalf("expected 4 filter clauses (taxi_type, status, created_at range, price range), got %+v", boolQuery["filter"])
	}
}

func TestSearchOrders_NoFiltersUsesMatchAll(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
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
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
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
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
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
