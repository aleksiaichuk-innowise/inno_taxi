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
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
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
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
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
