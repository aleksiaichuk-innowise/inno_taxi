package mongo

import (
	"context"
	"errors"
	"fmt"

	client "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/app/db/mongo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type appliedMigration struct {
	Version int    `bson:"version"`
	Name    string `bson:"name"`
}

func RunMigrations(ctx context.Context, c client.MongoClient) error {

	col := c.Database.Collection("migrations")

	opts := options.FindOne().SetSort(bson.M{"version": -1})

	var last appliedMigration
	err := col.FindOne(ctx, bson.M{"version": 1}, opts).Decode(&last)
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		return fmt.Errorf("find migrations error: %w", err)
	}
	for _, m := range All {
		if last.Version < m.Version {
			if err := m.Up(ctx, c.Database); err != nil {
				return fmt.Errorf("up migration error: %w", err)
			}
		}
	}
	return nil
}
