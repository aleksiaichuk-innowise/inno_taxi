package mongo

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	drivermongo "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/errorsx"
)

func (r UserRepository) SetPassword(ctx context.Context, id string, password string) (err error) {
	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	filter := bson.M{"_id": objID, "deleted_at": nil}
	update := bson.M{"$set": bson.M{
		"password_hash": password,
		"updated_at":    time.Now(),
	}}
	opts := options.FindOneAndUpdate()

	res := r.client.Database.Collection(usersCollection).FindOneAndUpdate(ctx, filter, update, opts)
	if err := res.Err(); err != nil {
		if errors.Is(err, drivermongo.ErrNoDocuments) {
			return errorsx.ErrUserNotFound
		}
		return err
	}
	return nil
}
