package redis_repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const sessionActive = "1"

type SessionRepository struct {
	client *redis.Client
}

func NewSessionRepository(client *redis.Client) *SessionRepository {
	return &SessionRepository{client: client}
}

func accessKey(sid string) string {
	return fmt.Sprintf("auth:session:%s:access", sid)
}

func refreshKey(sid string) string {
	return fmt.Sprintf("auth:session:%s:refresh", sid)
}

func (r *SessionRepository) SaveAccessSession(ctx context.Context, sid string, ttl time.Duration) error {
	return r.client.Set(ctx, accessKey(sid), sessionActive, ttl).Err()
}

func (r *SessionRepository) SaveRefreshSession(ctx context.Context, sid string, ttl time.Duration) error {
	return r.client.Set(ctx, refreshKey(sid), sessionActive, ttl).Err()
}

func (r *SessionRepository) AccessSessionExists(ctx context.Context, sid string) (bool, error) {
	return r.exists(ctx, accessKey(sid))
}

func (r *SessionRepository) RefreshSessionExists(ctx context.Context, sid string) (bool, error) {
	return r.exists(ctx, refreshKey(sid))
}

func (r *SessionRepository) DeleteAccessSession(ctx context.Context, sid string) error {
	return r.delete(ctx, accessKey(sid))
}

func (r *SessionRepository) DeleteRefreshSession(ctx context.Context, sid string) error {
	return r.delete(ctx, refreshKey(sid))
}

func (r *SessionRepository) exists(ctx context.Context, key string) (bool, error) {
	n, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *SessionRepository) delete(ctx context.Context, key string) error {
	err := r.client.Del(ctx, key).Err()
	if err != nil && !errors.Is(err, redis.Nil) {
		return err
	}
	return nil
}
