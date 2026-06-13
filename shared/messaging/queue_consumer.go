package messaging

import (
	"context"
	"encoding/json"
	"log"

	"github.com/rabbitmq/amqp091-go"

	"ride-sharing/shared/contracts"
)

type QueueConsumer struct {
	rb        *RabbitMQ
	connMgr   *RedisConnectionManager
	queueName string
}

func tripTopic(tripID string) string {
	if tripID == "" {
		return ""
	}
	return "trip:" + tripID
}

func deriveTripTopic(routingKey string, payload json.RawMessage) string {
	switch routingKey {
	// trip event created and driver not interested have no topic because they are only relevant for the initial driver matching phase, which is global and not trip-scoped
	// so we need to unmarshal to extract tripID for topic derivation
	// unmarshalling means converting the JSON payload into a Go struct to access the tripID
	case contracts.TripEventDriverNotInterested, contracts.TripEventNoDriversFound:
		// payload shape is { trip: Trip, pickupLat, pickupLng } — trip ID is nested
		var event struct {
			Trip struct {
				ID string `json:"id"`
			} `json:"trip"`
		}
		if err := json.Unmarshal(payload, &event); err == nil {
			return tripTopic(event.Trip.ID)
		}
	case contracts.TripEventDriverAssigned:
		// payload shape is { id, userID, pickupLat, pickupLng, selectedFare } — trip ID is top-level
		var trip struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(payload, &trip); err == nil {
			return tripTopic(trip.ID)
		}
	case contracts.PaymentEventSessionCreated:
		var payment struct {
			TripID string `json:"tripID"`
		}
		if err := json.Unmarshal(payload, &payment); err == nil {
			return tripTopic(payment.TripID)
		}
	}

	return ""
}

func NewQueueConsumer(rb *RabbitMQ, connMgr *RedisConnectionManager, queueName string) *QueueConsumer {
	return &QueueConsumer{
		rb:        rb,
		connMgr:   connMgr,
		queueName: queueName,
	}
}

func (qc *QueueConsumer) Start() error {
	return qc.rb.ConsumeMessages(qc.queueName, func(ctx context.Context, msg amqp091.Delivery) error {
		var amqpMsg contracts.AmqpMessage
		if err := json.Unmarshal(msg.Body, &amqpMsg); err != nil {
			log.Println("Failed to unmarshal AMQP message:", err)
			return err
		}

		userID := amqpMsg.OwnerID
		if userID == "" {
			log.Println("Message has no owner ID, skipping")
			return nil
		}
		log.Printf("Processing message in consumer for user %s from queue %s", userID, qc.queueName)

		clientMsg := contracts.WSMessage{
			Type:  msg.RoutingKey,
			Topic: deriveTripTopic(msg.RoutingKey, amqpMsg.Data),
			Data:  amqpMsg.Data,
		}

		if err := qc.connMgr.SendMessage(userID, clientMsg); err != nil {
			log.Printf("Failed to deliver message for user %s: %v", userID, err)
			return err
		}

		log.Printf("Processed message of type '%s' for user %s", clientMsg.Type, userID)
		return nil
	})
}
