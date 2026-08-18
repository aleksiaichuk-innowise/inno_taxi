package mongo

import (
	"context"
	"errors"

	"github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/service"
	"go.mongodb.org/mongo-driver/v2/bson"
	drivermongo "go.mongodb.org/mongo-driver/v2/mongo"

	dbmongo "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/app/db/mongo"
	repoEntity "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/entity/repository"
	serviceEntity "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/entity/service"

	"github.com/aleksiaichuk-innowise/inno_taxi/shared/errorsx"
)

const usersCollection = "users"

var _ service.UserRepository = (*UserRepository)(nil)

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

func (r UserRepository) FindByLogin(ctx context.Context, login string) (*serviceEntity.User, error) {
	filter := bson.M{"$or": []bson.M{{"email": login}, {"phone": login}}}
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

func rolesToStrings(roles []serviceEntity.Role) []string {
	out := make([]string, len(roles))
	for i, r := range roles {
		out[i] = string(r)
	}
	return out
}

func stringsToRoles(roles []string) []serviceEntity.Role {
	out := make([]serviceEntity.Role, len(roles))
	for i, r := range roles {
		out[i] = serviceEntity.Role(r)
	}
	return out
}
