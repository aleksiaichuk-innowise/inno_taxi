package redis_repo

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestRepo(t *testing.T) (*SessionRepository, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { client.Close() })
	return NewSessionRepository(client), mr
}

func TestSessionRepository_SaveAndExists(t *testing.T) {
	repo, _ := newTestRepo(t)
	ctx := t.Context()

	if exists, err := repo.AccessSessionExists(ctx, "sid-1"); err != nil || exists {
		t.Fatalf("expected no session yet, got exists=%v err=%v", exists, err)
	}

	if err := repo.SaveAccessSession(ctx, "sid-1", time.Minute); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	exists, err := repo.AccessSessionExists(ctx, "sid-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Fatal("expected session to exist after saving")
	}

	// Access and refresh sessions for the same sid are independent.
	if exists, err := repo.RefreshSessionExists(ctx, "sid-1"); err != nil || exists {
		t.Fatalf("expected refresh session to not exist, got exists=%v err=%v", exists, err)
	}
}

func TestSessionRepository_Delete(t *testing.T) {
	repo, _ := newTestRepo(t)
	ctx := t.Context()

	if err := repo.SaveRefreshSession(ctx, "sid-1", time.Minute); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := repo.DeleteRefreshSession(ctx, "sid-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	exists, err := repo.RefreshSessionExists(ctx, "sid-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Fatal("expected session to be gone after delete")
	}

	// Deleting an already-absent session must not error (logout is idempotent).
	if err := repo.DeleteRefreshSession(ctx, "sid-1"); err != nil {
		t.Fatalf("expected deleting an absent session to be a no-op, got %v", err)
	}
}

func TestSessionRepository_ExpiresAfterTTL(t *testing.T) {
	repo, mr := newTestRepo(t)
	ctx := t.Context()

	if err := repo.SaveAccessSession(ctx, "sid-1", time.Second); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mr.FastForward(2 * time.Second)

	exists, err := repo.AccessSessionExists(ctx, "sid-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Fatal("expected session to have expired")
	}
}
