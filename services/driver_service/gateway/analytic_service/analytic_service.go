package analytic_service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// AnalyticGateway reads aggregated trip-rating stats back from
// analytic_service. It hits the same public route analytic_service exposes
// to gateway_service for the Analyst role - there's no separate /internal
// endpoint for it, since the handler itself performs no role check (role
// gating is gateway_service's job) and this call happens over the docker
// network, not through the gateway.
type AnalyticGateway interface {
	GetDriverRatingStats(ctx context.Context, driverID string) (average float64, count int64, err error)
}

type gateway struct {
	baseURL string
	client  *http.Client
}

func NewAnalyticGateway(baseURL string, client *http.Client) AnalyticGateway {
	if client == nil {
		client = http.DefaultClient
	}
	return &gateway{baseURL: baseURL, client: client}
}

type driverRatingResp struct {
	Average float64 `json:"average"`
	Count   int64   `json:"count"`
}

func (g *gateway) GetDriverRatingStats(ctx context.Context, driverID string) (float64, int64, error) {
	url := fmt.Sprintf("%s/analytics/ratings/drivers/%s", g.baseURL, driverID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("build get driver rating stats request: %w", err)
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("call analytic service get driver rating stats: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("analytic service get driver rating stats returned unexpected status %d", resp.StatusCode)
	}

	var r driverRatingResp
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return 0, 0, fmt.Errorf("decode get driver rating stats response: %w", err)
	}
	return r.Average, r.Count, nil
}
