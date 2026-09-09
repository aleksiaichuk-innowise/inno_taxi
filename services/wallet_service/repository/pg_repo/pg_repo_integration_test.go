//go:build integration

package pg_repo_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"

	dbpostgres "github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/app/db/postgres"
	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/errorsx"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/repository/pg_repo"
	shared "github.com/aleksiaichuk-innowise/inno_taxi/shared/config"
	"github.com/jmoiron/sqlx"
)

var testRepo *pg_repo.PgRepository

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
		Repository: "postgres",
		Tag:        "18",
		Env: []string{
			"POSTGRES_USER=postgres",
			"POSTGRES_PASSWORD=postgres",
			"POSTGRES_DB=wallet_test",
		},
	}, func(hc *docker.HostConfig) {
		hc.AutoRemove = true
	})
	if err != nil {
		fmt.Println("could not start postgres container:", err)
		return 1
	}
	defer func() { _ = pool.Purge(resource) }()

	cfg := shared.PostgresConfig{
		Host:                  "localhost",
		Port:                  resource.GetPort("5432/tcp"),
		Username:              "postgres",
		Password:              "postgres",
		Database:              "wallet_test",
		MinConnections:        1,
		MaxConnections:        5,
		MaxConnectionLifetime: time.Minute,
		MaxIdleConnections:    time.Minute,
	}

	ctx := context.Background()
	var db *sqlx.DB
	pool.MaxWait = 60 * time.Second
	if err := pool.Retry(func() error {
		var err error
		db, err = dbpostgres.NewDB(ctx, cfg)
		return err
	}); err != nil {
		fmt.Println("could not connect to postgres:", err)
		return 1
	}
	defer db.Close()

	if err := applyMigrations(db); err != nil {
		fmt.Println("could not apply migrations:", err)
		return 1
	}

	testRepo = pg_repo.NewPgRepo(db)

	return m.Run()
}

// applyMigrations runs the "Up" half of every migrations/postgres/*.sql
// file directly (in filename order), the same files goose applies in
// production via `make migrate-wallet-up`. Reusing them as the source of
// truth here means the integration test always exercises the schema
// that's actually shipped, not a hand-copied approximation of it.
func applyMigrations(db *sqlx.DB) error {
	files, err := filepath.Glob("../../migrations/postgres/*.sql")
	if err != nil {
		return err
	}
	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read %s: %w", f, err)
		}
		up, _, _ := strings.Cut(string(content), "-- +goose Down")
		if _, err := db.Exec(up); err != nil {
			return fmt.Errorf("apply %s: %w", f, err)
		}
	}
	return nil
}

var idCounter int64

func uniqueUserID(t *testing.T) string {
	t.Helper()
	n := atomic.AddInt64(&idCounter, 1)
	return fmt.Sprintf("user-%d-%d", time.Now().UnixNano(), n)
}

func TestPgRepository_CreateWallet_DuplicateUserRejected(t *testing.T) {
	ctx := t.Context()
	userID := uniqueUserID(t)

	if _, err := testRepo.CreateWallet(ctx, userID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := testRepo.CreateWallet(ctx, userID); !errors.Is(err, errorsx.ErrWalletAlreadyExists) {
		t.Fatalf("expected ErrWalletAlreadyExists, got %v", err)
	}
}

func TestPgRepository_ChargeIdempotency_ConcurrentSameReference(t *testing.T) {
	ctx := t.Context()
	userID := uniqueUserID(t)

	wallet, err := testRepo.CreateWallet(ctx, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tx, err := testRepo.Start(ctx)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := testRepo.UpdateBalance(ctx, tx, wallet.ID, 10_000); err != nil {
		t.Fatalf("seed balance: %v", err)
	}
	if err := testRepo.Finish(tx); err != nil {
		t.Fatalf("finish: %v", err)
	}

	const referenceID = "order-concurrent-1"
	const workers = 10

	type result struct {
		txID    string
		created bool
	}
	results := make(chan result, workers)
	errs := make(chan error, workers)

	for range workers {
		go func() {
			tx, err := testRepo.Start(ctx)
			if err != nil {
				errs <- err
				return
			}
			defer testRepo.Abort(tx)

			w, err := testRepo.GetWalletForUpdate(ctx, tx, userID)
			if err != nil {
				errs <- err
				return
			}
			if existing, found, err := testRepo.FindTransactionByReference(ctx, tx, w.ID, referenceID, service_dto.TransactionTypeDebit); err != nil {
				errs <- err
				return
			} else if found {
				// Another goroutine already committed; nothing to do.
				if err := testRepo.Finish(tx); err != nil {
					errs <- err
					return
				}
				results <- result{txID: existing.ID, created: false}
				return
			}
			if err := testRepo.UpdateBalance(ctx, tx, w.ID, w.BalanceMinorUnits-1_000); err != nil {
				errs <- err
				return
			}
			record, created, err := testRepo.CreateTransaction(ctx, tx, w.ID, service_dto.TransactionTypeDebit, 1_000, referenceID)
			if err != nil {
				errs <- err
				return
			}
			if err := testRepo.Finish(tx); err != nil {
				errs <- err
				return
			}
			results <- result{txID: record.ID, created: created}
		}()
	}

	var txIDs = map[string]bool{}
	createdCount := 0
	for range workers {
		select {
		case err := <-errs:
			t.Fatalf("worker error: %v", err)
		case r := <-results:
			txIDs[r.txID] = true
			if r.created {
				createdCount++
			}
		}
	}

	if len(txIDs) != 1 {
		t.Fatalf("expected exactly one distinct transaction id across all workers, got %d: %v", len(txIDs), txIDs)
	}
	if createdCount != 1 {
		t.Fatalf("expected exactly one worker to have created the transaction, got %d", createdCount)
	}

	final, err := testRepo.GetWalletByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if final.BalanceMinorUnits != 9_000 {
		t.Fatalf("balance = %d, want 9000 (charged exactly once despite %d concurrent attempts)", final.BalanceMinorUnits, workers)
	}
}
