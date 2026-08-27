package repository

import (
	"context"
	"errors"
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

func (r DriverRepository) FindByUserID(ctx context.Context, id string) (service_dto.Driver, error) {
	filter := bson.M{"user_id": id}
	res := r.client.Database.Collection(DRIVER_COLLECTION).FindOne(ctx, filter)

	var d service_dto.Driver
	if err := res.Decode(&d); err != nil {
		if errors.Is(err, drivermongo.ErrNoDocuments) {
			return service_dto.Driver{}, errorsx.ErrDriverNotFound
		}
		return service_dto.Driver{}, err
	}
	return d, nil
}
func (r DriverRepository) UpdateStatusByUserID(ctx context.Context, userID, status string) error {
	filter := bson.M{"user_id": userID}
	update := bson.M{"$set": bson.M{"status": status, "updated_at": time.Now()}}

	res, err := r.client.Database.Collection(DRIVER_COLLECTION).UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return errorsx.ErrDriverNotFound
	}
	return nil
}

func (r DriverRepository) UpdateTaxiTypeByUserID(ctx context.Context, userID, status string) error {
	filter := bson.M{"user_id": userID}
	update := bson.M{"$set": bson.M{"taxi_type": status, "updated_at": time.Now()}}

	res, err := r.client.Database.Collection(DRIVER_COLLECTION).UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return errorsx.ErrDriverNotFound
	}
	return nil
}

func (r *DriverRepository) FindByStatus(ctx context.Context, status service_dto.Status) ([]service_dto.Driver, error) {
	filter := bson.M{}
	if status != "" {
		filter["status"] = string(status)
	}

	cursor, err := r.client.Database.Collection(DRIVER_COLLECTION).Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []repository.Driver
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	drivers := make([]service_dto.Driver, 0, len(docs))
	for _, doc := range docs {
		drivers = append(drivers, service_dto.Driver{
			ID:        doc.ID.Hex(),
			UserID:    doc.UserID,
			TaxiType:  service_dto.TaxiType(doc.TaxiType),
			Status:    service_dto.Status(doc.Status),
			CreatedAt: doc.CreatedAt,
			UpdatedAt: doc.UpdatedAt,
		})
	}

	return drivers, nil
}
