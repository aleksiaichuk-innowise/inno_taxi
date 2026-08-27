package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	drivermongo "go.mongodb.org/mongo-driver/v2/mongo"

	repoEntity "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/entity/repository"
	serviceEntity "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/errorsx"
)

func (r UserRepository) CreateUser(ctx context.Context, user *serviceEntity.User) (*serviceEntity.User, error) {
	doc := repoEntity.User{
		ID:           bson.NewObjectID(),
		Name:         user.Name,
		Email:        user.Email,
		Phone:        user.Phone,
		PasswordHash: user.PasswordHash,
		Roles:        serviceEntity.RolesToStrings(user.Roles),
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}

	if _, err := r.client.Database.Collection(UsersCollection).InsertOne(ctx, doc); err != nil {
		if drivermongo.IsDuplicateKeyError(err) {
			return nil, errorsx.ErrUserAlreadyExists
		}
		return nil, err
	}

	created := *user
	created.ID = doc.ID.Hex()
	return &created, nil
}
