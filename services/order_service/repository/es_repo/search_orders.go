package es_repo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	repo_entity "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/repository"
	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
)

// buildSearchQuery builds a filter-only bool query (no scoring needed) from
// the given filter, falling back to match_all when nothing is set - see the
// design doc's "Data flow: search" section.
func buildSearchQuery(filter service_dto.OrderSearchFilter) map[string]any {
	var clauses []map[string]any

	if filter.TaxiType != "" {
		clauses = append(clauses, map[string]any{"term": map[string]any{"taxi_type": string(filter.TaxiType)}})
	}
	if filter.Status != "" {
		clauses = append(clauses, map[string]any{"term": map[string]any{"status": string(filter.Status)}})
	}
	if !filter.CreatedAfter.IsZero() || !filter.CreatedBefore.IsZero() {
		r := map[string]any{}
		if !filter.CreatedAfter.IsZero() {
			r["gte"] = filter.CreatedAfter.Format(time.RFC3339)
		}
		if !filter.CreatedBefore.IsZero() {
			r["lte"] = filter.CreatedBefore.Format(time.RFC3339)
		}
		clauses = append(clauses, map[string]any{"range": map[string]any{"created_at": r}})
	}
	if filter.MinPriceMinorUnits != 0 || filter.MaxPriceMinorUnits != 0 {
		r := map[string]any{}
		if filter.MinPriceMinorUnits != 0 {
			r["gte"] = filter.MinPriceMinorUnits
		}
		if filter.MaxPriceMinorUnits != 0 {
			r["lte"] = filter.MaxPriceMinorUnits
		}
		clauses = append(clauses, map[string]any{"range": map[string]any{"price_minor_units": r}})
	}

	query := map[string]any{"match_all": map[string]any{}}
	if len(clauses) > 0 {
		query = map[string]any{"bool": map[string]any{"filter": clauses}}
	}

	return map[string]any{
		"query": query,
		"sort":  []map[string]any{{"created_at": map[string]any{"order": "desc"}}},
	}
}

type searchResponseBody struct {
	Hits struct {
		Total struct {
			Value int64 `json:"value"`
		} `json:"total"`
		Hits []struct {
			Source repo_entity.OrderDocument `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

func (r EsRepository) SearchOrders(ctx context.Context, filter service_dto.OrderSearchFilter) ([]service_dto.Order, int64, error) {
	body, err := json.Marshal(buildSearchQuery(filter))
	if err != nil {
		return nil, 0, fmt.Errorf("marshal search query: %w", err)
	}

	res, err := r.client.Search(
		r.client.Search.WithContext(ctx),
		r.client.Search.WithIndex(r.index),
		r.client.Search.WithBody(bytes.NewReader(body)),
		r.client.Search.WithFrom(int(filter.Offset)),
		r.client.Search.WithSize(int(filter.Limit)),
		r.client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("search orders: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, 0, fmt.Errorf("search orders returned unexpected status %d", res.StatusCode)
	}

	var parsed searchResponseBody
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return nil, 0, fmt.Errorf("decode search response: %w", err)
	}

	orders := make([]service_dto.Order, len(parsed.Hits.Hits))
	for i, h := range parsed.Hits.Hits {
		orders[i] = h.Source.ToDomain()
	}
	return orders, parsed.Hits.Total.Value, nil
}
