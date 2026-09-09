package wallet_service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/errorsx"
)

func TestWalletGateway_Charge_Succeeds(t *testing.T) {
	var gotPath string
	var gotBody transactionReq
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	gw := NewWalletGateway(srv.URL, nil)
	if err := gw.Charge(context.Background(), "user-1", 500, "order-1"); err != nil {
		t.Fatalf("Charge() error = %v, want nil", err)
	}
	if gotPath != "/internal/wallets/user-1/charge" {
		t.Errorf("got path %q, want /internal/wallets/user-1/charge", gotPath)
	}
	if gotBody.AmountMinorUnits != 500 || gotBody.ReferenceID != "order-1" {
		t.Errorf("unexpected request body: %+v", gotBody)
	}
}

func TestWalletGateway_Charge_InsufficientFunds(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
	}))
	defer srv.Close()

	gw := NewWalletGateway(srv.URL, nil)
	err := gw.Charge(context.Background(), "user-1", 500, "order-1")
	if !errors.Is(err, errorsx.ErrInsufficientFunds) {
		t.Fatalf("Charge() error = %v, want ErrInsufficientFunds", err)
	}
}

func TestWalletGateway_Refund_Succeeds(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	gw := NewWalletGateway(srv.URL, nil)
	if err := gw.Refund(context.Background(), "user-1", 500, "order-1"); err != nil {
		t.Fatalf("Refund() error = %v, want nil", err)
	}
	if gotPath != "/internal/wallets/user-1/refund" {
		t.Errorf("got path %q, want /internal/wallets/user-1/refund", gotPath)
	}
}

func TestWalletGateway_UnexpectedStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	gw := NewWalletGateway(srv.URL, nil)
	err := gw.Charge(context.Background(), "user-1", 500, "order-1")
	if err == nil {
		t.Fatal("expected an error for an unexpected status code")
	}
	if errors.Is(err, errorsx.ErrInsufficientFunds) {
		t.Fatal("a 500 must not be mistaken for ErrInsufficientFunds")
	}
}
