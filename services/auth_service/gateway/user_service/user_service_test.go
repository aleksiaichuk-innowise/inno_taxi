package user_service

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aleksiaichuk-innowise/inno_taxi/services/auth_service/errorsx"
)

func TestVerifyCredentials_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/verify-credentials" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var req verifyCredentialsReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Login != "user@example.com" || req.Password != "password123" {
			t.Fatalf("unexpected request body: %+v", req)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(userResp{ID: "user-1", Roles: []string{"user", "driver"}})
	}))
	defer srv.Close()

	gw := NewUserServiceGateway(srv.URL, nil)
	info, err := gw.VerifyCredentials(t.Context(), "user@example.com", "password123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.ID != "user-1" || len(info.Roles) != 2 {
		t.Fatalf("unexpected user info: %+v", info)
	}
}

func TestVerifyCredentials_InvalidCredentials(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	gw := NewUserServiceGateway(srv.URL, nil)
	_, err := gw.VerifyCredentials(t.Context(), "user@example.com", "wrong")
	if !errors.Is(err, errorsx.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestVerifyCredentials_UnexpectedStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	gw := NewUserServiceGateway(srv.URL, nil)
	_, err := gw.VerifyCredentials(t.Context(), "user@example.com", "password123")
	if err == nil {
		t.Fatal("expected an error for an unexpected status code")
	}
}
