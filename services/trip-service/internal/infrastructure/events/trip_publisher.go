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

	// Extract pickup from the first coordinate of the route geometry
	if trip.RideFare != nil && trip.RideFare.Route != nil && len(trip.RideFare.Route.Routes) > 0 {
		coords := trip.RideFare.Route.Routes[0].Geometry.Coordinates
		if len(coords) > 0 {
			payload.PickupLat = coords[0][0] // lat
			payload.PickupLng = coords[0][1] // lng
		}
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
