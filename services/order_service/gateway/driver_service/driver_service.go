package driver_service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// DriverGateway talks to driver_service's internal endpoints. Unlike
// WalletGateway, "no driver available" isn't an error here - it's a normal
// matching outcome (ok=false, err=nil), same as "no drivers nearby" would
// be in a real app; callers shouldn't have to errors.Is-check for it.
type DriverGateway interface {
	ClaimAvailableDriver(ctx context.Context, taxiType string) (userID string, ok bool, err error)
	ReleaseDriver(ctx context.Context, userID string) error
}

type gateway struct {
	baseURL string
	client  *http.Client
}

func NewDriverGateway(baseURL string, client *http.Client) DriverGateway {
	if client == nil {
		client = http.DefaultClient
	}
	return &gateway{baseURL: baseURL, client: client}
}

type claimReq struct {
	TaxiType string `json:"taxi_type"`
}

type driverResp struct {
	UserID string `json:"user_id"`
}

func (g *gateway) ClaimAvailableDriver(ctx context.Context, taxiType string) (string, bool, error) {
	body, err := json.Marshal(claimReq{TaxiType: taxiType})
	if err != nil {
		return "", false, fmt.Errorf("marshal claim driver request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.baseURL+"/internal/drivers/claim", bytes.NewReader(body))
	if err != nil {
		return "", false, fmt.Errorf("build claim driver request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return "", false, fmt.Errorf("call driver service claim: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var d driverResp
		if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
			return "", false, fmt.Errorf("decode claim driver response: %w", err)
		}
		return d.UserID, true, nil
	case http.StatusNotFound:
		return "", false, nil
	default:
		return "", false, fmt.Errorf("driver service claim returned unexpected status %d", resp.StatusCode)
	}
}

type updateStatusReq struct {
	Status string `json:"status"`
}

func (g *gateway) ReleaseDriver(ctx context.Context, userID string) error {
	body, err := json.Marshal(updateStatusReq{Status: "available"})
	if err != nil {
		return fmt.Errorf("marshal release driver request: %w", err)
	}

	url := fmt.Sprintf("%s/internal/drivers/%s/status", g.baseURL, userID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build release driver request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("call driver service release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("driver service release returned unexpected status %d", resp.StatusCode)
	}
	return nil
}
