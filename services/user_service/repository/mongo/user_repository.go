package mongo

import (
	dbmongo "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/app/db/mongo"
	serviceEntity "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/service"
)

const usersCollection = "users"

var _ service.UserRepository = (*UserRepository)(nil)

type UserRepository struct {
	client *dbmongo.MongoClient
}

func NewUserRepository(client *dbmongo.MongoClient) *UserRepository {
	return &UserRepository{client: client}
}

func stringsToRoles(roles []string) []serviceEntity.Role {
	out := make([]serviceEntity.Role, len(roles))
	for i, r := range roles {
		out[i] = serviceEntity.Role(r)
	}
	return out
}
