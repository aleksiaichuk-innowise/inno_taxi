package repository

type Driver struct {
	ID        int    `bson:"_id,omitempty"`
	UserID    string `bson:"user_id"`
	TaxiType  string `bson:"taxi_type"`
	Status    string `bson:"status"`
	CreatedAt string `bson:"created_at"`
	UpdatedAt string `bson:"updated_at"`
}
