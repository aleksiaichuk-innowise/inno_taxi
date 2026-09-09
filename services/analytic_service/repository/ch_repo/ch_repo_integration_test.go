//go:build integration

package ch_repo_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"

	dbclickhouse "github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/app/db/clickhouse"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/config"
	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/repository/ch_repo"
)

var testRepo *ch_repo.ClickHouseRepository

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	pool, err := dockertest.NewPool("")
	if err != nil {
		fmt.Println("could not connect to docker:", err)
		return 1
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "clickhouse/clickhouse-server",
		Tag:        "24.8-alpine",
		Env: []string{
			"CLICKHOUSE_DB=analytics_test",
			"CLICKHOUSE_USER=default",
			"CLICKHOUSE_PASSWORD=clickhouse",
		},
	}, func(hc *docker.HostConfig) {
		hc.AutoRemove = true
	})
	if err != nil {
		fmt.Println("could not start clickhouse container:", err)
		return 1
	}
	defer func() { _ = pool.Purge(resource) }()

	cfg := config.ClickHouseConfig{
		Addr:     fmt.Sprintf("localhost:%s", resource.GetPort("9000/tcp")),
		Database: "analytics_test",
		Username: "default",
		Password: "clickhouse",
	}

	ctx := context.Background()
	var conn interface {
		Close() error
	}
	var chRepo *ch_repo.ClickHouseRepository
	pool.MaxWait = 60 * time.Second
	if err := pool.Retry(func() error {
		c, err := dbclickhouse.New(ctx, cfg)
		if err != nil {
			return err
		}
		conn = c
		chRepo = ch_repo.NewClickHouseRepository(c)
		return nil
	}); err != nil {
		fmt.Println("could not connect to clickhouse:", err)
		return 1
	}
	defer conn.Close()

	testRepo = chRepo

	return m.Run()
}

// Tests in this file share one ClickHouse instance and table for the
// whole run (a fresh container per `go test` invocation, not per test) -
// they rely on Go's default declaration-order execution and deliberately
// use non-overlapping status values (`completed`/`cancelled` here,
// `created` in the dedup test below) so neither test's aggregate query
// counts the other's rows.
func TestClickHouseRepository_InsertAndStats(t *testing.T) {
	ctx := t.Context()
	now := time.Now().UTC().Truncate(time.Millisecond)

	events := []service_dto.OrderEvent{
		{OrderID: "order-1", UserID: "user-1", TaxiType: "economy", Status: "completed", CreatedAt: now},
		{OrderID: "order-2", UserID: "user-1", TaxiType: "comfort", Status: "completed", CreatedAt: now},
		{OrderID: "order-3", UserID: "user-2", TaxiType: "economy", Status: "cancelled", CreatedAt: now},
	}
	for _, e := range events {
		if err := testRepo.InsertOrderEvent(ctx, e); err != nil {
			t.Fatalf("insert order event: %v", err)
		}
	}

	from := now.Add(-time.Hour)
	to := now.Add(time.Hour)

	stats, err := testRepo.GetOrderStats(ctx, from, to)
	if err != nil {
		t.Fatalf("get order stats: %v", err)
	}
	if stats.TotalOrders != 3 {
		t.Fatalf("total orders = %d, want 3", stats.TotalOrders)
	}
	if stats.CountsByStatus["completed"] != 2 || stats.CountsByStatus["cancelled"] != 1 {
		t.Fatalf("unexpected counts by status: %+v", stats.CountsByStatus)
	}
	if stats.CountsByTaxiType["economy"] != 2 || stats.CountsByTaxiType["comfort"] != 1 {
		t.Fatalf("unexpected counts by taxi type: %+v", stats.CountsByTaxiType)
	}
}

func TestClickHouseRepository_ReplacingMergeTree_DedupesRedeliveredEvent(t *testing.T) {
	ctx := t.Context()
	now := time.Now().UTC().Truncate(time.Millisecond)

	// Simulate a Kafka redelivery: the exact same order_id inserted twice.
	event := service_dto.OrderEvent{OrderID: "order-redelivered", UserID: "user-1", TaxiType: "economy", Status: "created", CreatedAt: now}
	if err := testRepo.InsertOrderEvent(ctx, event); err != nil {
		t.Fatalf("first insert: %v", err)
	}
	if err := testRepo.InsertOrderEvent(ctx, event); err != nil {
		t.Fatalf("second (redelivered) insert: %v", err)
	}

	from := now.Add(-time.Hour)
	to := now.Add(time.Hour)

	stats, err := testRepo.GetOrderStats(ctx, from, to)
	if err != nil {
		t.Fatalf("get order stats: %v", err)
	}
	// FINAL must collapse the duplicate row down to one, even though
	// ReplacingMergeTree hasn't run a background merge yet.
	if stats.CountsByStatus["created"] != 1 {
		t.Fatalf("expected the redelivered event to be deduped to 1, got %d", stats.CountsByStatus["created"])
	}
}
