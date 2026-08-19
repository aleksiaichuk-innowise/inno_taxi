package mongo

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	drivermongo "go.mongodb.org/mongo-driver/v2/mongo"

	repoEntity "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/entity/repository"
	serviceEntity "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/errorsx"
)

func (r UserRepository) FindByID(ctx context.Context, id string) (*serviceEntity.User, error) {
	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, errorsx.ErrUserNotFound
	}
	filter := bson.M{"_id": objID, "deleted_at": nil}
	res := r.client.Database.Collection(usersCollection).FindOne(ctx, filter)
	var doc repoEntity.User
	if err := res.Decode(&doc); err != nil {
		if errors.Is(err, drivermongo.ErrNoDocuments) {
			return nil, errorsx.ErrUserNotFound
		}
		return nil, err
	}

	return &serviceEntity.User{
		ID:           doc.ID.Hex(),
		Name:         doc.Name,
		Email:        doc.Email,
		Phone:        doc.Phone,
		PasswordHash: doc.PasswordHash,
		Roles:        stringsToRoles(doc.Roles),
		CreatedAt:    doc.CreatedAt,
		UpdatedAt:    doc.UpdatedAt,
		DeletedAt:    doc.DeletedAt,
	}, nil
}
