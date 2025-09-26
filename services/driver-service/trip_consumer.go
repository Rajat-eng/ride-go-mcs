package main

import (
	"context"
	"encoding/json"
	"log"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"

	"github.com/rabbitmq/amqp091-go"
)

type tripConsumer struct {
	rabbitmq *messaging.RabbitMQ
	service  *Service
}

func NewTripConsumer(rabbitmq *messaging.RabbitMQ, service *Service) *tripConsumer {
	return &tripConsumer{
		rabbitmq: rabbitmq,
		service:  service,
	}
}

func (c *tripConsumer) Listen() error {
	return c.rabbitmq.ConsumeMessages(messaging.FindAvailableDriversQueue, func(ctx context.Context, msg amqp091.Delivery) error {
		var tripEvent contracts.AmqpMessage
		if err := json.Unmarshal(msg.Body, &tripEvent); err != nil {
			log.Printf("Error unmarshaling trip event: %v", err)
			return err
		}
		var payload messaging.TripEventData
		if err := json.Unmarshal(tripEvent.Data, &payload); err != nil {
			log.Printf("Error unmarshaling trip event data: %v", err)
			return err
		}
		// only these keys are mapped to FindAvailableDriversQueue
		switch msg.RoutingKey {
		case contracts.TripEventCreated, contracts.TripEventDriverNotInterested:
			return c.handleAndNotifyDrivers(ctx, payload)
		}
		log.Printf("driver received message: %+v", payload)
		return nil
	})
}

func (c *tripConsumer) handleAndNotifyDrivers(ctx context.Context, payload messaging.TripEventData) error {
	suitableIDs := c.service.FindAvailableDrivers(payload.Trip.SelectedFare.PackageSlug) // payload is of type TripEventData which is protobuf message

	if len(suitableIDs) == 0 {
		// 	If no driver → publish TripEventNoDriversFound (not bound to this queue, so goes elsewhere — consumed by user/gateway).
		if err := c.rabbitmq.PublishMessage(ctx, contracts.TripEventNoDriversFound, contracts.AmqpMessage{
			OwnerID: payload.Trip.UserID,
		}); err != nil {
			log.Printf("Failed to publish message to exchange: %v", err)
			return err
		}

		return nil
	}

	suitableDriverID := suitableIDs[0] // For simplicity, pick the first suitable driver
	marshalledEvent, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// If driver found → publish DriverCmdTripRequest (goes to driver-specific consumer/gateway).

	// if rejected by driver then driver service publishes tripNotInterested event to trip exchange after selecting no option
	// this event will be read by trip_consumer and new driver will be assigned
	if err := c.rabbitmq.PublishMessage(ctx, contracts.DriverCmdTripRequest, contracts.AmqpMessage{
		OwnerID: suitableDriverID, // recipient of message
		Data:    marshalledEvent,
	}); err != nil {
		log.Printf("Failed to publish message to exchange: %v", err)
		return err
	}

	return nil

}
