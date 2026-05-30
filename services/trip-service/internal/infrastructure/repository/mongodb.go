package repository

import (
	"context"
	"fmt"

	"ride-sharing/services/trip-service/internal/domain"
	"ride-sharing/shared/db"
	pbd "ride-sharing/shared/proto/trip"
	"ride-sharing/shared/tracing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type mongoRepository struct {
	db *mongo.Database
}

func NewMongoRepository(db *mongo.Database) *mongoRepository {
	return &mongoRepository{db: db}
}

func (r *mongoRepository) CreateTrip(ctx context.Context, trip *domain.TripModel) (*domain.TripModel, error) {
	var created *domain.TripModel
	err := tracing.RunInSpan(ctx, "db", "mongodb.trips.insert", tracing.DBSpanAttrs("mongodb",
		attribute.String("db.collection", db.TripsCollection),
		attribute.String("trip.user_id", trip.UserID),
	), func(ctx context.Context, _ trace.Span) error {
		result, err := r.db.Collection(db.TripsCollection).InsertOne(ctx, trip)
		if err != nil {
			return err
		}

		trip.ID = result.InsertedID.(primitive.ObjectID)
		created = trip
		return nil
	})
	return created, err
}

func (r *mongoRepository) GetTripByID(ctx context.Context, id string) (*domain.TripModel, error) {
	var trip *domain.TripModel
	err := tracing.RunInSpan(ctx, "db", "mongodb.trips.find_one", tracing.DBSpanAttrs("mongodb",
		attribute.String("db.collection", db.TripsCollection),
		attribute.String("trip.id", id),
	), func(ctx context.Context, _ trace.Span) error {
		_id, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return err
		}

		result := r.db.Collection(db.TripsCollection).FindOne(ctx, bson.M{"_id": _id})
		if result.Err() != nil {
			return result.Err()
		}

		var decoded domain.TripModel
		if err := result.Decode(&decoded); err != nil {
			return err
		}
		trip = &decoded
		return nil
	})
	return trip, err
}

func (r *mongoRepository) UpdateTrip(ctx context.Context, tripID string, status string, driver *pbd.TripDriver) error {
	return tracing.RunInSpan(ctx, "db", "mongodb.trips.update_one", tracing.DBSpanAttrs("mongodb",
		attribute.String("db.collection", db.TripsCollection),
		attribute.String("trip.id", tripID),
		attribute.String("trip.status", status),
	), func(ctx context.Context, _ trace.Span) error {
		_id, err := primitive.ObjectIDFromHex(tripID)
		if err != nil {
			return err
		}

		setFields := bson.M{"status": status}
		if driver != nil {
			setFields["driver"] = driver
		}
		update := bson.M{"$set": setFields}

		result, err := r.db.Collection(db.TripsCollection).UpdateOne(ctx, bson.M{"_id": _id}, update)
		if err != nil {
			return err
		}

		if result.ModifiedCount == 0 {
			return fmt.Errorf("trip not found: %s", tripID)
		}

		return nil
	})
}
