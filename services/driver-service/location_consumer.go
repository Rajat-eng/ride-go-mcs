package main

import (
	"context"
	"encoding/json"
	"log"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"

	"github.com/rabbitmq/amqp091-go"
)

type locationConsumer struct {
	rabbitmq *messaging.RabbitMQ
	service  *Service
}

func NewLocationConsumer(rabbitmq *messaging.RabbitMQ, service *Service) *locationConsumer {
	return &locationConsumer{rabbitmq: rabbitmq, service: service}
}

func (c *locationConsumer) Listen() error {
	return c.rabbitmq.ConsumeMessages(messaging.DriverLocationUpdateQueue, func(ctx context.Context, msg amqp091.Delivery) error {
		var message contracts.AmqpMessage
		if err := json.Unmarshal(msg.Body, &message); err != nil {
			log.Printf("location_consumer: failed to unmarshal envelope: %v", err)
			return err
		}

		var payload messaging.DriverLocationUpdateData
		if err := json.Unmarshal(message.Data, &payload); err != nil {
			log.Printf("location_consumer: failed to unmarshal payload: %v", err)
			return err
		}

		if err := c.service.UpdateDriverLocation(message.OwnerID, payload.PackageSlug, payload.Latitude, payload.Longitude); err != nil {
			log.Printf("location_consumer: failed to update location for driver %s: %v", message.OwnerID, err)
			return err
		}

		log.Printf("location_consumer: updated driver %s → %.5f, %.5f", message.OwnerID, payload.Latitude, payload.Longitude)

		// Publish rider-facing location event if this driver has an active rider
		riderID, err := c.service.GetActiveRider(message.OwnerID)
		if err != nil {
			// No active rider or error — just skip rider notification
			log.Printf("location_consumer: no active rider for driver %s (or error: %v), skipping rider update", message.OwnerID, err)
			return nil
		}

		locationEvent := messaging.DriverLocationEventData{
			Latitude:  payload.Latitude,
			Longitude: payload.Longitude,
		}
		eventPayload, _ := json.Marshal(locationEvent)
		if err := c.rabbitmq.PublishMessage(ctx, contracts.DriverEventLocation, contracts.AmqpMessage{
			OwnerID: riderID,
			Data:    eventPayload,
		}); err != nil {
			log.Printf("location_consumer: failed to publish rider location update: %v", err)
			return err
		}

		log.Printf("location_consumer: published location to rider %s", riderID)
		return nil
	})
}
