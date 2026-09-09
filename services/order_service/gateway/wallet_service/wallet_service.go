package wallet_service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/errorsx"
)

type WalletGateway interface {
	Charge(ctx context.Context, userID string, amountMinorUnits int64, referenceID string) error
	Refund(ctx context.Context, userID string, amountMinorUnits int64, referenceID string) error
}

type gateway struct {
	baseURL string
	client  *http.Client
}

func NewWalletGateway(baseURL string, client *http.Client) WalletGateway {
	if client == nil {
		client = http.DefaultClient
	}
	return &gateway{baseURL: baseURL, client: client}
}

type transactionReq struct {
	AmountMinorUnits int64  `json:"amount_minor_units"`
	ReferenceID      string `json:"reference_id"`
}

func (g *gateway) Charge(ctx context.Context, userID string, amountMinorUnits int64, referenceID string) error {
	return g.doTransaction(ctx, "charge", userID, amountMinorUnits, referenceID)
}

func (g *gateway) Refund(ctx context.Context, userID string, amountMinorUnits int64, referenceID string) error {
	return g.doTransaction(ctx, "refund", userID, amountMinorUnits, referenceID)
}

func (g *gateway) doTransaction(ctx context.Context, kind, userID string, amountMinorUnits int64, referenceID string) error {
	body, err := json.Marshal(transactionReq{AmountMinorUnits: amountMinorUnits, ReferenceID: referenceID})
	if err != nil {
		return fmt.Errorf("marshal %s request: %w", kind, err)
	}

	url := fmt.Sprintf("%s/internal/wallets/%s/%s", g.baseURL, userID, kind)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build %s request: %w", kind, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("call wallet service %s: %w", kind, err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated:
		return nil
	case resp.StatusCode == http.StatusPaymentRequired:
		return errorsx.ErrInsufficientFunds
	default:
		return fmt.Errorf("wallet service %s returned unexpected status %d", kind, resp.StatusCode)
	}
}
