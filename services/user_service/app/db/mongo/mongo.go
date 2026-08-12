package mongo

import (
	"context"
	"fmt"
	"time"

	"github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/config"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoClient struct {
	Client   *mongo.Client
	Database *mongo.Database
}

func New(ctx context.Context, cfg *config.MongoConfig) (*MongoClient, error) {
	uri := fmt.Sprintf(
		"mongodb://%s:%s@%s:%s",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port,
	)

	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(connectCtx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("mongo: connect: %w", err)
	}

	pingCtx, cancelPing := context.WithTimeout(ctx, 5*time.Second)
	defer cancelPing()

	if err := client.Ping(pingCtx, nil); err != nil {
		return nil, fmt.Errorf("mongo: ping: %w", err)
	}

	return &MongoClient{
		Client:   client,
		Database: client.Database(cfg.Database),
	}, nil
}

func (m *MongoClient) Close(ctx context.Context) error {
	return m.Client.Disconnect(ctx)
}
