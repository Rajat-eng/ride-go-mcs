package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"ride-sharing/services/trip-service/internal/domain"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"
	pb "ride-sharing/shared/proto/trip"

	"github.com/rabbitmq/amqp091-go"
)

type driverConsumer struct {
	rabbitmq *messaging.RabbitMQ
	service  domain.TripService
}

func NewDriverConsumer(rabbitmq *messaging.RabbitMQ, service domain.TripService) *driverConsumer {
	return &driverConsumer{
		rabbitmq: rabbitmq,
		service:  service,
	}
}
func (c *driverConsumer) Listen() error {
	return c.rabbitmq.ConsumeMessages(messaging.DriverTripResponseQueue, func(ctx context.Context, msg amqp091.Delivery) error {
		var message contracts.AmqpMessage
		if err := json.Unmarshal(msg.Body, &message); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			return err
		}
		var payload messaging.DriverTripResponseData
		if err := json.Unmarshal(message.Data, &payload); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			return err
		}
		log.Printf("driver response received message: %+v", payload)
		switch msg.RoutingKey {
		case contracts.DriverCmdTripAccept:
			if err := c.handleTripAccepted(ctx, payload); err != nil {
				log.Printf("Failed to handle the trip accept: %v", err)
				return err
			}
		case contracts.DriverCmdTripDecline:
			if err := c.handleTripDeclined(ctx, payload.TripID, payload.RiderID); err != nil {
				log.Printf("Failed to handle the trip decline: %v", err)
				return err
			}
			return nil
		}
		log.Printf("unknown trip event: %+v", payload)
		return nil
	})
}

func (c *driverConsumer) handleTripDeclined(ctx context.Context, tripID, riderID string) error {
	// When a driver declines, we should try to find another driver

	trip, err := c.service.GetTripByID(ctx, tripID)
	if err != nil {
		return err
	}

	newPayload := messaging.TripEventData{
		Trip: trip.ToProto(),
	}

	marshalledPayload, err := json.Marshal(newPayload)
	if err != nil {
		return err
	}

	if err := c.rabbitmq.PublishMessage(ctx, contracts.TripEventDriverNotInterested,
		contracts.AmqpMessage{
			OwnerID: riderID,
			Data:    marshalledPayload,
		},
	); err != nil {
		return err
	}

	return nil
}
func (c *driverConsumer) handleTripAccepted(ctx context.Context, payload messaging.DriverTripResponseData) error {
	// 1. Fetch the trip
	trip, err := c.service.GetTripByID(ctx, payload.TripID)
	if err != nil {
		return err
	}
	if trip == nil {
		return fmt.Errorf("trip was not found %s", payload.TripID)
	}

	// 2. Build TripDriver from the enriched payload (gateway populated this from auth context)
	tripDriver := &pb.TripDriver{
		Id:   payload.DriverID,
		Name: payload.DriverName,
	}

	// 3. Update the trip with status + driver
	if err := c.service.UpdateTrip(ctx, payload.TripID, "accepted", tripDriver); err != nil {
		log.Printf("Failed to update the trip: %v", err)
		return err
	}
	trip, err = c.service.GetTripByID(ctx, payload.TripID)
	if err != nil {
		return err
	}

	// 4. Publish TripEventDriverAssigned with proto JSON so the rider gets lowercase keys
	marshalledTrip, err := json.Marshal(trip.ToProto())
	if err != nil {
		return err
	}
	if err := c.rabbitmq.PublishMessage(ctx, contracts.TripEventDriverAssigned, contracts.AmqpMessage{
		OwnerID: trip.UserID,
		Data:    marshalledTrip,
	}); err != nil {
		return err
	}

	// 5. Notify payment service
	marshalledPayload, err := json.Marshal(messaging.PaymentTripResponseData{
		TripID:   payload.TripID,
		UserID:   trip.UserID,
		DriverID: payload.DriverID,
		Amount:   trip.RideFare.TotalPriceInCents,
		Currency: "USD",
	})
	if err != nil {
		return err
	}
	if err := c.rabbitmq.PublishMessage(ctx, contracts.PaymentCmdCreateSession,
		contracts.AmqpMessage{
			OwnerID: trip.UserID,
			Data:    marshalledPayload,
		},
	); err != nil {
		return err
	}
	return nil
}
