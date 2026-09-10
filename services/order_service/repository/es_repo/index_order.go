package es_repo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	repo_entity "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/repository"
	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
)

func (r EsRepository) IndexOrder(ctx context.Context, order service_dto.Order) error {
	doc := repo_entity.OrderDocumentFromDomain(order)
	body, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal order document: %w", err)
	}

	res, err := r.client.Index(r.index, bytes.NewReader(body),
		r.client.Index.WithDocumentID(order.ID),
		r.client.Index.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("index order: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("index order returned unexpected status %d", res.StatusCode)
	}
	return nil
}
