package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Migration struct {
	Version int
	Name    string
	Up      func(c context.Context, db *mongo.Database) error
}

var All = []Migration{
	{
		Version: 1,
		Name:    "unique_index_email",
		Up: func(ctx context.Context, db *mongo.Database) error {
			_, err := db.Collection("users").Indexes().CreateOne(ctx, mongo.IndexModel{
				Keys:    bson.D{{Key: "email", Value: 1}},
				Options: options.Index().SetUnique(true).SetName("uniq_email"),
			})
			return err
		},
	},
	{
		Version: 2,
		Name:    "unique_index_phone",
		Up: func(ctx context.Context, db *mongo.Database) error {
			_, err := db.Collection("users").Indexes().CreateOne(ctx, mongo.IndexModel{
				Keys:    bson.D{{Key: "phone", Value: 1}},
				Options: options.Index().SetUnique(true).SetName("uniq_phone"),
			})
			return err
		},
	},
}
