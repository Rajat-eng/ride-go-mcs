package main

import (
	"context"
	"encoding/json"
	"log"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"

	"github.com/rabbitmq/amqp091-go"
)

// tripAssignedPayload matches the proto Trip JSON produced by trip.ToProto()
type tripAssignedPayload struct {
	UserID string `json:"userID"` // riderID
	Driver struct {
		ID string `json:"id"` // driverID
	} `json:"driver"`
}

type tripAssignedConsumer struct {
	rabbitmq *messaging.RabbitMQ
	service  *Service
}

func NewTripAssignedConsumer(rabbitmq *messaging.RabbitMQ, service *Service) *tripAssignedConsumer {
	return &tripAssignedConsumer{rabbitmq: rabbitmq, service: service}
}

func (c *tripAssignedConsumer) Listen() error {
	return c.rabbitmq.ConsumeMessages(messaging.DriverTripAssignedQueue, func(ctx context.Context, msg amqp091.Delivery) error {
		if msg.RoutingKey != contracts.TripEventDriverAssigned {
			return nil
		}

		var envelope contracts.AmqpMessage
		if err := json.Unmarshal(msg.Body, &envelope); err != nil {
			log.Printf("trip_assigned_consumer: failed to unmarshal envelope: %v", err)
			return err
		}

		var payload tripAssignedPayload
		if err := json.Unmarshal(envelope.Data, &payload); err != nil {
			log.Printf("trip_assigned_consumer: failed to unmarshal payload: %v", err)
			return err
		}

		if payload.Driver.ID == "" || payload.UserID == "" {
			log.Printf("trip_assigned_consumer: missing driverID or riderID, skipping")
			return nil
		}

		if err := c.service.SetActiveRider(payload.Driver.ID, payload.UserID); err != nil {
			log.Printf("trip_assigned_consumer: failed to store active rider for driver %s: %v", payload.Driver.ID, err)
			return err
		}

		log.Printf("trip_assigned_consumer: driver %s → rider %s", payload.Driver.ID, payload.UserID)
		return nil
	})
}
