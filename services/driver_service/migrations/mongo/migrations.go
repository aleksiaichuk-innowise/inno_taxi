package mongo

import (
	"context"

	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/repository"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongoMigration struct {
	Version int
	Name    string
	Up      func(ctx context.Context, db *mongo.Database) error
}

var AllMigrations = []MongoMigration{
	{
		Version: 1,
		Name:    "unique_index_user_id",
		Up: func(ctx context.Context, db *mongo.Database) error {
			_, err := db.Collection(repository.DRIVER_COLLECTION).Indexes().CreateOne(ctx, mongo.IndexModel{
				Keys:    bson.D{{Key: "user_id", Value: 1}},
				Options: options.Index().SetUnique(true).SetName("unique_index_user_id"),
			})
			return err
		},
	},
}
