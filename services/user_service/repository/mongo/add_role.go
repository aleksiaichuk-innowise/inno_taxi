package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/errorsx"
)

func (r UserRepository) AddRole(ctx context.Context, id, role string) error {
	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	filter := bson.M{"_id": objID, "deleted_at": nil}
	update := bson.M{
		"$addToSet": bson.M{
			"roles": role,
		},
		"$set": bson.M{
			"updated_at": time.Now(),
		},
	}
	res, err := r.client.Database.Collection(UsersCollection).UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if res.MatchedCount == 0 {
		return errorsx.ErrUserNotFound
	}

	return nil
}
