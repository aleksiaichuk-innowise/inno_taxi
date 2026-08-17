package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	drivermongo "go.mongodb.org/mongo-driver/v2/mongo"

	dbmongo "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/app/db/mongo"
	repoEntity "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/entity/repository"
	serviceEntity "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/shared/errorsx"
)

const usersCollection = "users"

type UserRepository struct {
	client *dbmongo.MongoClient
}

func NewUserRepository(client *dbmongo.MongoClient) *UserRepository {
	return &UserRepository{client: client}
}

func (r UserRepository) CreateUser(ctx context.Context, user *serviceEntity.User) (*serviceEntity.User, error) {
	doc := repoEntity.User{
		ID:           bson.NewObjectID(),
		Name:         user.Name,
		Email:        user.Email,
		Phone:        user.Phone,
		PasswordHash: user.PasswordHash,
		Roles:        rolesToStrings(user.Roles),
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}

	if _, err := r.client.Database.Collection(usersCollection).InsertOne(ctx, doc); err != nil {
		if drivermongo.IsDuplicateKeyError(err) {
			return nil, errorsx.ErrUserAlreadyExists
		}
		return nil, err
	}

	created := *user
	created.ID = doc.ID.Hex()
	return &created, nil
}

func rolesToStrings(roles []serviceEntity.Role) []string {
	out := make([]string, len(roles))
	for i, r := range roles {
		out[i] = string(r)
	}
	return out
}
