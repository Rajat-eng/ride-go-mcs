package events

import (
	"context"
	"encoding/json"
	"ride-sharing/services/trip-service/internal/domain"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"
)

type TripEventPublisher struct {
	rabbitMQ *messaging.RabbitMQ
}

func NewTripEventPublisher(rabbitMQ *messaging.RabbitMQ) *TripEventPublisher {
	return &TripEventPublisher{
		rabbitMQ: rabbitMQ,
	}
}

func (p *TripEventPublisher) PublishTripCreated(ctx context.Context, trip *domain.TripModel) error {
	payload := &messaging.TripEventData{
		Trip: trip.ToProto(),
	}
	tripEventJSON, err := json.Marshal(payload) // marshal struct to JSON
	if err != nil {
		return err
	}
	return p.rabbitMQ.PublishMessage(ctx, contracts.TripEventCreated, contracts.AmqpMessage{
		Data:    tripEventJSON,
		OwnerID: trip.UserID,
	})
}
