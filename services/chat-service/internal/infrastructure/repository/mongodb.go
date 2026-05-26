package repository

import (
	"context"
	"time"

	"ride-sharing/services/chat-service/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const collectionName = "chat_messages"

// MongoMessageRepository implements domain.MessageRepository using MongoDB.
type MongoMessageRepository struct {
	col *mongo.Collection
}

func NewMongoMessageRepository(db *mongo.Database) *MongoMessageRepository {
	col := db.Collection(collectionName)

	// Compound index: tripID + sentAt (descending) for efficient history queries.
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "tripID", Value: 1},
			{Key: "sentAt", Value: -1},
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := col.Indexes().CreateOne(ctx, indexModel); err != nil {
		// Log but do not fatal — index creation may fail on read-only replicas.
		_ = err
	}

	return &MongoMessageRepository{col: col}
}

func (r *MongoMessageRepository) Save(ctx context.Context, msg *domain.Message) error {
	doc := bson.M{
		"_id":       msg.ID,
		"tripID":    msg.TripID,
		"senderID":  msg.SenderID,
		"text":      msg.Text,
		"sentAt":    msg.SentAt,
		"delivered": msg.Delivered,
	}
	_, err := r.col.InsertOne(ctx, doc)
	return err
}

func (r *MongoMessageRepository) GetByTripID(ctx context.Context, tripID string, limit int) ([]*domain.Message, error) {
	if limit <= 0 {
		limit = 50
	}
	opts := options.Find().
		SetSort(bson.D{{Key: "sentAt", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := r.col.Find(ctx, bson.M{"tripID": tripID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var msgs []*domain.Message
	for cursor.Next(ctx) {
		var doc struct {
			ID        string `bson:"_id"`
			TripID    string `bson:"tripID"`
			SenderID  string `bson:"senderID"`
			Text      string `bson:"text"`
			SentAt    int64  `bson:"sentAt"`
			Delivered bool   `bson:"delivered"`
		}
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		msgs = append(msgs, &domain.Message{
			ID:        doc.ID,
			TripID:    doc.TripID,
			SenderID:  doc.SenderID,
			Text:      doc.Text,
			SentAt:    doc.SentAt,
			Delivered: doc.Delivered,
		})
	}
	return msgs, cursor.Err()
}

func (r *MongoMessageRepository) MarkDelivered(ctx context.Context, messageID string) error {
	_, err := r.col.UpdateOne(
		ctx,
		bson.M{"_id": messageID},
		bson.M{"$set": bson.M{"delivered": true}},
	)
	return err
}
