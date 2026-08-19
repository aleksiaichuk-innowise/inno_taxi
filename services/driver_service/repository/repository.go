package repository

import (
	"context"
	"time"

	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/app/db/mongo"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/entity/repository"
	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/errorsx"

	"go.mongodb.org/mongo-driver/v2/bson"
	drivermongo "go.mongodb.org/mongo-driver/v2/mongo"
)

const DRIVER_COLLECTION = "drivers"

type DriverRepository struct {
	client mongo.MongoClient
}

func NewDriverRepository(client mongo.MongoClient) *DriverRepository {
	return &DriverRepository{client: client}
}

func (r DriverRepository) CreateDriver(ctx context.Context, dto *service_dto.CreateDriverInput) (service_dto.Driver, error) {
	doc := repository.Driver{
		ID:        bson.NewObjectID(),
		UserID:    dto.UserID,
		TaxiType:  string(dto.TaxiType),
		Status:    string(service_dto.StatusOffline),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err := r.client.Database.Collection(DRIVER_COLLECTION).InsertOne(ctx, doc)
	if err != nil {
		if drivermongo.IsDuplicateKeyError(err) {
			return service_dto.Driver{}, errorsx.ErrDriverAlreadyExists
		}
		return service_dto.Driver{}, err
	}

	return service_dto.Driver{
		ID:        doc.ID.Hex(),
		UserID:    doc.UserID,
		TaxiType:  dto.TaxiType,
		Status:    service_dto.StatusOffline,
		CreatedAt: doc.CreatedAt,
		UpdatedAt: doc.UpdatedAt,
	}, nil

}
