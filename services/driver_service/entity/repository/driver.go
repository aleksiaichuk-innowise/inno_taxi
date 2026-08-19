package repository

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Driver struct {
	ID        bson.ObjectID `bson:"_id,omitempty"`
	UserID    string        `bson:"user_id"`
	TaxiType  string        `bson:"taxi_type"`
	Status    string        `bson:"status"`
	CreatedAt time.Time     `bson:"created_at"`
	UpdatedAt time.Time     `bson:"updated_at"`
}
