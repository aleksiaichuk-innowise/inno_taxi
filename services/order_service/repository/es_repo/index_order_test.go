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
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
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
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
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
