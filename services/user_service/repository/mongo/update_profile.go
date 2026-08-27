package mongo

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	drivermongo "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	repoEntity "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/entity/repository"
	serviceEntity "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/errorsx"
)

func (r UserRepository) UpdateProfile(ctx context.Context, id, name, email, phone string) (*serviceEntity.User, error) {
	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, errorsx.ErrUserNotFound
	}

	filter := bson.M{"_id": objID, "deleted_at": nil}
	update := bson.M{"$set": bson.M{
		"name":       name,
		"email":      email,
		"phone":      phone,
		"updated_at": time.Now(),
	}}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	res := r.client.Database.Collection(UsersCollection).FindOneAndUpdate(ctx, filter, update, opts)

	var doc repoEntity.User
	if err := res.Decode(&doc); err != nil {
		if errors.Is(err, drivermongo.ErrNoDocuments) {
			return nil, errorsx.ErrUserNotFound
		}
		if drivermongo.IsDuplicateKeyError(err) {
			return nil, errorsx.ErrUserAlreadyExists
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
