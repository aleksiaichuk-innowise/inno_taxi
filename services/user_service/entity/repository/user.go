package repository

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type User struct {
	ID           bson.ObjectID `bson:"_id,omitempty"`
	Name         string        `bson:"name"`
	Email        string        `bson:"email"`
	Phone        string        `bson:"phone"`
	PasswordHash string        `bson:"password_hash"`
	Roles        []string      `bson:"roles"`
	CreatedAt    time.Time     `bson:"created_at"`
	UpdatedAt    time.Time     `bson:"updated_at"`
	DeletedAt    *time.Time    `bson:"deleted_at,omitempty"`
}
