package driver_service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDriverGateway_ClaimAvailableDriver_Found(t *testing.T) {
	var gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		buf, _ := io.ReadAll(r.Body)
		gotBody = string(buf)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"d1","user_id":"driver-1","taxi_type":"economy","status":"on-trip"}`))
	}))
	defer srv.Close()

	gw := NewDriverGateway(srv.URL, nil)
	userID, ok, err := gw.ClaimAvailableDriver(context.Background(), "economy")
	if err != nil {
		t.Fatalf("ClaimAvailableDriver() error = %v, want nil", err)
	}
	if !ok {
		t.Fatal("expected ok=true when a driver is found")
	}
	if userID != "driver-1" {
		t.Errorf("got userID %q, want driver-1", userID)
	}
	if gotPath != "/internal/drivers/claim" {
		t.Errorf("got path %q, want /internal/drivers/claim", gotPath)
	}
	if gotBody != `{"taxi_type":"economy"}` {
		t.Errorf("got body %q", gotBody)
	}
}

func TestDriverGateway_ClaimAvailableDriver_NoneAvailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	gw := NewDriverGateway(srv.URL, nil)
	userID, ok, err := gw.ClaimAvailableDriver(context.Background(), "economy")
	if err != nil {
		t.Fatalf("ClaimAvailableDriver() error = %v, want nil (no driver available isn't an error)", err)
	}
	if ok {
		t.Fatal("expected ok=false when no driver is available")
	}
	if userID != "" {
		t.Errorf("got userID %q, want empty", userID)
	}
}

func TestDriverGateway_ClaimAvailableDriver_UnexpectedStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	gw := NewDriverGateway(srv.URL, nil)
	_, ok, err := gw.ClaimAvailableDriver(context.Background(), "economy")
	if err == nil {
		t.Fatal("expected an error for an unexpected status code")
	}
	if ok {
		t.Fatal("expected ok=false on error")
	}
}

func TestDriverGateway_ReleaseDriver_Succeeds(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	gw := NewDriverGateway(srv.URL, nil)
	if err := gw.ReleaseDriver(context.Background(), "driver-1"); err != nil {
		t.Fatalf("ReleaseDriver() error = %v, want nil", err)
	}
	if gotPath != "/internal/drivers/driver-1/status" {
		t.Errorf("got path %q, want /internal/drivers/driver-1/status", gotPath)
	}
	if gotMethod != http.MethodPatch {
		t.Errorf("got method %q, want PATCH", gotMethod)
	}
}

func TestDriverGateway_ReleaseDriver_Fails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	gw := NewDriverGateway(srv.URL, nil)
	if err := gw.ReleaseDriver(context.Background(), "driver-1"); err == nil {
		t.Fatal("expected an error for a non-200 release response")
	}
}
