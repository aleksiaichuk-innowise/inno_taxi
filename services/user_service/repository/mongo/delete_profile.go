package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/aleksiaichuk-innowise/inno_taxi/shared/errorsx"
)

func (r UserRepository) DeleteProfile(ctx context.Context, id string) (err error) {
	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return errorsx.ErrUserNotFound
	}
	filter := bson.M{
		"_id":        objID,
		"deleted_at": nil,
	}
	update := bson.M{
		"$set": bson.M{
			"deleted_at": time.Now(),
			"updated_at": time.Now(),
		},
	}

	res, err := r.client.Database.Collection(usersCollection).UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return errorsx.ErrUserNotFound
	}
	return nil
}
