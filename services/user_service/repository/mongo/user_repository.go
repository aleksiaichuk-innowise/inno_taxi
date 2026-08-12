package mongo

import "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/app/db/mongo"

type UserRepository struct {
	client *mongo.MongoClient
}

func NewUserRepository(client *mongo.MongoClient) *UserRepository {
	return &UserRepository{client: client}
}
