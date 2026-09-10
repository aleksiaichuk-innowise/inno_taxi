package es_repo

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
)

// EnsureIndex creates the given index with this package's mapping if it
// doesn't already exist. Idempotent - safe to call on every startup.
func EnsureIndex(ctx context.Context, client *elasticsearch.Client, index string) error {
	existsRes, err := client.Indices.Exists([]string{index}, client.Indices.Exists.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("check index exists: %w", err)
	}
	defer existsRes.Body.Close()

	if existsRes.StatusCode == 200 {
		return nil
	}
	if existsRes.StatusCode != 404 {
		body, readErr := io.ReadAll(existsRes.Body)
		if readErr != nil {
			return fmt.Errorf("check index exists returned unexpected status %d", existsRes.StatusCode)
		}
		return fmt.Errorf("check index exists returned unexpected status %d: %s", existsRes.StatusCode, body)
	}

	createRes, err := client.Indices.Create(index,
		client.Indices.Create.WithContext(ctx),
		client.Indices.Create.WithBody(strings.NewReader(indexMapping)),
	)
	if err != nil {
		return fmt.Errorf("create index: %w", err)
	}
	defer createRes.Body.Close()

	if createRes.IsError() {
		body, readErr := io.ReadAll(createRes.Body)
		if readErr != nil {
			return fmt.Errorf("create index returned unexpected status %d", createRes.StatusCode)
		}
		return fmt.Errorf("create index returned unexpected status %d: %s", createRes.StatusCode, body)
	}
	return nil
}
