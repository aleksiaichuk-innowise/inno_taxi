package service

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/errorsx"
)

// captureLogs swaps the global slog default for the duration of the test so
// audit-log assertions can inspect what Charge/Refund actually emitted -
// restored automatically via t.Cleanup.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	original := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(original) })
	return &buf
}

func TestCreateWallet(t *testing.T) {
	repo := newFakeWalletRepository()
	svc := NewWalletService(repo, newFakeTransactionCache())

	w, err := svc.CreateWallet(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.UserID != "user-1" || w.BalanceMinorUnits != 0 {
		t.Fatalf("unexpected wallet: %+v", w)
	}

	if _, err := svc.CreateWallet(context.Background(), "user-1"); !errors.Is(err, errorsx.ErrWalletAlreadyExists) {
		t.Fatalf("expected ErrWalletAlreadyExists, got %v", err)
	}
}

func TestGetWallet_NotFound(t *testing.T) {
	repo := newFakeWalletRepository()
	svc := NewWalletService(repo, newFakeTransactionCache())

	if _, err := svc.GetWallet(context.Background(), "missing"); !errors.Is(err, errorsx.ErrWalletNotFound) {
		t.Fatalf("expected ErrWalletNotFound, got %v", err)
	}
}

func TestCharge_InvalidAmount(t *testing.T) {
	repo := newFakeWalletRepository()
	svc := NewWalletService(repo, newFakeTransactionCache())
	if _, err := svc.CreateWallet(context.Background(), "user-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, _, err := svc.Charge(context.Background(), "user-1", 0, "order-1")
	if !errors.Is(err, errorsx.ErrInvalidAmount) {
		t.Fatalf("expected ErrInvalidAmount for zero, got %v", err)
	}
	_, _, err = svc.Charge(context.Background(), "user-1", -100, "order-1")
	if !errors.Is(err, errorsx.ErrInvalidAmount) {
		t.Fatalf("expected ErrInvalidAmount for negative, got %v", err)
	}
}

func TestCharge_WalletNotFound(t *testing.T) {
	repo := newFakeWalletRepository()
	svc := NewWalletService(repo, newFakeTransactionCache())

	_, _, err := svc.Charge(context.Background(), "missing", 100, "order-1")
	if !errors.Is(err, errorsx.ErrWalletNotFound) {
		t.Fatalf("expected ErrWalletNotFound, got %v", err)
	}
}

func TestCharge_InsufficientFunds(t *testing.T) {
	repo := newFakeWalletRepository()
	cache := newFakeTransactionCache()
	svc := NewWalletService(repo, cache)
	w, _ := svc.CreateWallet(context.Background(), "user-1")
	// Balance starts at 0; any positive charge must fail.
	_, _, err := svc.Charge(context.Background(), w.UserID, 100, "order-1")
	if !errors.Is(err, errorsx.ErrInsufficientFunds) {
		t.Fatalf("expected ErrInsufficientFunds, got %v", err)
	}
}

func TestCharge_SuccessThenIdempotentReplay(t *testing.T) {
	repo := newFakeWalletRepository()
	cache := newFakeTransactionCache()
	svc := NewWalletService(repo, cache)
	svc.CreateWallet(context.Background(), "user-1")
	// Give the wallet funds directly through the repo, bypassing Charge/Refund.
	repo.UpdateBalance(context.Background(), nil, "wallet-user-1", 1000)

	tx1, created1, err := svc.Charge(context.Background(), "user-1", 300, "order-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created1 {
		t.Fatal("expected the first charge to be newly created")
	}

	w, _ := svc.GetWallet(context.Background(), "user-1")
	if w.BalanceMinorUnits != 700 {
		t.Fatalf("balance = %d, want 700", w.BalanceMinorUnits)
	}
	if cache.invalidateCall["wallet-user-1"] != 1 {
		t.Fatalf("expected cache invalidation once, got %d", cache.invalidateCall["wallet-user-1"])
	}

	tx2, created2, err := svc.Charge(context.Background(), "user-1", 300, "order-1")
	if err != nil {
		t.Fatalf("unexpected error on replay: %v", err)
	}
	if created2 {
		t.Fatal("expected the replayed charge to NOT be newly created")
	}
	if tx1.ID != tx2.ID {
		t.Fatalf("expected the replay to return the same transaction, got %q vs %q", tx1.ID, tx2.ID)
	}

	w, _ = svc.GetWallet(context.Background(), "user-1")
	if w.BalanceMinorUnits != 700 {
		t.Fatalf("balance after replay = %d, want unchanged 700", w.BalanceMinorUnits)
	}
}

func TestRefund_SuccessCreditsBalance(t *testing.T) {
	repo := newFakeWalletRepository()
	svc := NewWalletService(repo, newFakeTransactionCache())
	svc.CreateWallet(context.Background(), "user-1")

	tx, created, err := svc.Refund(context.Background(), "user-1", 500, "order-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Fatal("expected the first refund to be newly created")
	}
	if tx.AmountMinorUnits != 500 {
		t.Fatalf("unexpected amount: %d", tx.AmountMinorUnits)
	}

	w, _ := svc.GetWallet(context.Background(), "user-1")
	if w.BalanceMinorUnits != 500 {
		t.Fatalf("balance = %d, want 500", w.BalanceMinorUnits)
	}
}

func TestRefund_InvalidAmount(t *testing.T) {
	repo := newFakeWalletRepository()
	svc := NewWalletService(repo, newFakeTransactionCache())
	svc.CreateWallet(context.Background(), "user-1")

	if _, _, err := svc.Refund(context.Background(), "user-1", 0, "order-1"); !errors.Is(err, errorsx.ErrInvalidAmount) {
		t.Fatalf("expected ErrInvalidAmount, got %v", err)
	}
}

func TestCharge_SuccessIsAuditLogged(t *testing.T) {
	repo := newFakeWalletRepository()
	svc := NewWalletService(repo, newFakeTransactionCache())
	svc.CreateWallet(context.Background(), "user-1")
	repo.UpdateBalance(context.Background(), nil, "wallet-user-1", 1000)

	logs := captureLogs(t)
	if _, _, err := svc.Charge(context.Background(), "user-1", 300, "order-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := logs.String()
	if !strings.Contains(got, "wallet charge recorded") {
		t.Fatalf("expected an audit log line for the successful charge, got: %s", got)
	}
	if !strings.Contains(got, "order-1") || !strings.Contains(got, "wallet-user-1") {
		t.Fatalf("expected the audit log to identify the wallet and reference id, got: %s", got)
	}
}

func TestCharge_RejectionIsAuditLogged(t *testing.T) {
	repo := newFakeWalletRepository()
	svc := NewWalletService(repo, newFakeTransactionCache())
	svc.CreateWallet(context.Background(), "user-1")

	logs := captureLogs(t)
	if _, _, err := svc.Charge(context.Background(), "user-1", 100, "order-1"); !errors.Is(err, errorsx.ErrInsufficientFunds) {
		t.Fatalf("expected ErrInsufficientFunds, got %v", err)
	}

	got := logs.String()
	if !strings.Contains(got, "wallet charge rejected") {
		t.Fatalf("expected an audit log line for the rejected charge, got: %s", got)
	}
	if !strings.Contains(got, "insufficient funds") {
		t.Fatalf("expected the audit log to include the rejection reason, got: %s", got)
	}
}

func TestCharge_InvalidAmountIsAuditLogged(t *testing.T) {
	repo := newFakeWalletRepository()
	svc := NewWalletService(repo, newFakeTransactionCache())
	svc.CreateWallet(context.Background(), "user-1")

	logs := captureLogs(t)
	if _, _, err := svc.Charge(context.Background(), "user-1", 0, "order-1"); !errors.Is(err, errorsx.ErrInvalidAmount) {
		t.Fatalf("expected ErrInvalidAmount, got %v", err)
	}

	got := logs.String()
	if !strings.Contains(got, "wallet charge rejected") {
		t.Fatalf("expected an audit log line for the rejected charge, got: %s", got)
	}
}

func TestRefund_SuccessIsAuditLogged(t *testing.T) {
	repo := newFakeWalletRepository()
	svc := NewWalletService(repo, newFakeTransactionCache())
	svc.CreateWallet(context.Background(), "user-1")

	logs := captureLogs(t)
	if _, _, err := svc.Refund(context.Background(), "user-1", 500, "order-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := logs.String()
	if !strings.Contains(got, "wallet refund recorded") {
		t.Fatalf("expected an audit log line for the successful refund, got: %s", got)
	}
	if !strings.Contains(got, "order-1") || !strings.Contains(got, "wallet-user-1") {
		t.Fatalf("expected the audit log to identify the wallet and reference id, got: %s", got)
	}
}

func TestRefund_RejectionIsAuditLogged(t *testing.T) {
	repo := newFakeWalletRepository()
	svc := NewWalletService(repo, newFakeTransactionCache())
	svc.CreateWallet(context.Background(), "user-1")

	logs := captureLogs(t)
	if _, _, err := svc.Refund(context.Background(), "user-1", 0, "order-1"); !errors.Is(err, errorsx.ErrInvalidAmount) {
		t.Fatalf("expected ErrInvalidAmount, got %v", err)
	}

	got := logs.String()
	if !strings.Contains(got, "wallet refund rejected") {
		t.Fatalf("expected an audit log line for the rejected refund, got: %s", got)
	}
}

func TestCharge_IdempotentReplayIsAuditLoggedAsReplay(t *testing.T) {
	repo := newFakeWalletRepository()
	svc := NewWalletService(repo, newFakeTransactionCache())
	svc.CreateWallet(context.Background(), "user-1")
	repo.UpdateBalance(context.Background(), nil, "wallet-user-1", 1000)

	if _, _, err := svc.Charge(context.Background(), "user-1", 300, "order-1"); err != nil {
		t.Fatalf("unexpected error on first charge: %v", err)
	}

	logs := captureLogs(t)
	if _, created, err := svc.Charge(context.Background(), "user-1", 300, "order-1"); err != nil || created {
		t.Fatalf("expected the replay to succeed with created=false, got created=%v err=%v", created, err)
	}

	got := logs.String()
	if !strings.Contains(got, "wallet charge recorded") {
		t.Fatalf("expected the replay to still be audit logged, got: %s", got)
	}
	if !strings.Contains(got, "replay=true") {
		t.Fatalf("expected the audit log to flag the replay, got: %s", got)
	}
}

func TestListRecentTransactions_CachesOnMiss(t *testing.T) {
	repo := newFakeWalletRepository()
	cache := newFakeTransactionCache()
	svc := NewWalletService(repo, cache)
	svc.CreateWallet(context.Background(), "user-1")
	repo.UpdateBalance(context.Background(), nil, "wallet-user-1", 1000)
	svc.Charge(context.Background(), "user-1", 100, "order-1")

	if _, ok, _ := cache.GetRecent(context.Background(), "wallet-user-1"); ok {
		t.Fatal("expected cache to be empty before the first list call")
	}

	txs, err := svc.ListRecentTransactions(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txs) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(txs))
	}

	cached, ok, _ := cache.GetRecent(context.Background(), "wallet-user-1")
	if !ok || len(cached) != 1 {
		t.Fatalf("expected the list to populate the cache, got ok=%v cached=%v", ok, cached)
	}
}

func TestListRecentTransactions_WalletNotFound(t *testing.T) {
	repo := newFakeWalletRepository()
	svc := NewWalletService(repo, newFakeTransactionCache())

	if _, err := svc.ListRecentTransactions(context.Background(), "missing"); !errors.Is(err, errorsx.ErrWalletNotFound) {
		t.Fatalf("expected ErrWalletNotFound, got %v", err)
	}
}
