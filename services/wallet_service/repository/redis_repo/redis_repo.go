package redis_repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/entity/service"
	"github.com/redis/go-redis/v9"
)

const recentTransactionsTTL = 60 * time.Second

type TransactionCache struct {
	client *redis.Client
}

func NewTransactionCache(client *redis.Client) *TransactionCache {
	return &TransactionCache{client: client}
}

func recentKey(walletID string) string {
	return fmt.Sprintf("wallet:%s:recent_transactions", walletID)
}

func (c *TransactionCache) GetRecent(ctx context.Context, walletID string) ([]service_dto.Transaction, bool, error) {
	raw, err := c.client.Get(ctx, recentKey(walletID)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("get cached recent transactions: %w", err)
	}

	var txs []service_dto.Transaction
	if err := json.Unmarshal(raw, &txs); err != nil {
		return nil, false, fmt.Errorf("unmarshal cached recent transactions: %w", err)
	}
	return txs, true, nil
}

func (c *TransactionCache) SetRecent(ctx context.Context, walletID string, txs []service_dto.Transaction) error {
	raw, err := json.Marshal(txs)
	if err != nil {
		return fmt.Errorf("marshal recent transactions: %w", err)
	}
	if err := c.client.Set(ctx, recentKey(walletID), raw, recentTransactionsTTL).Err(); err != nil {
		return fmt.Errorf("set cached recent transactions: %w", err)
	}
	return nil
}

func (c *TransactionCache) Invalidate(ctx context.Context, walletID string) error {
	if err := c.client.Del(ctx, recentKey(walletID)).Err(); err != nil {
		return fmt.Errorf("invalidate cached recent transactions: %w", err)
	}
	return nil
}
