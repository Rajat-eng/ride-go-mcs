package events

import (
	"context"
	"encoding/json"
	"log"
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
			// OSRM geometry coordinates are [longitude, latitude].
			payload.PickupLng = coords[0][0]
			payload.PickupLat = coords[0][1]
		}
	}

	tripEventJSON, err := json.Marshal(payload) // marshal struct to JSON
	if err != nil {
		return err
	}
	log.Printf("Publishing Trip Created event for tripID: %s, userID: %s", trip.ID.Hex(), trip.UserID)
	return p.rabbitMQ.PublishMessage(ctx, contracts.TripEventCreated, contracts.AmqpMessage{
		// consumed by driver service to find suitable drivers and notify them of the new trip
		Data:    tripEventJSON,
		OwnerID: trip.UserID,
	})
}

// PublishTripCancelled notifies both rider and driver (if assigned) about the cancellation.
// It publishes a single AMQP message whose Data contains the full TripCancelledData so the
// ws-gateway cancel consumer can fan-out to both parties in one shot.
func (p *TripEventPublisher) PublishTripCancelled(ctx context.Context, tripID, riderID, driverID string, driverAccepted bool) error {
	payload := &messaging.TripCancelledData{
		TripID:         tripID,
		RiderID:        riderID,
		DriverID:       driverID,
		DriverAccepted: driverAccepted,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	log.Printf("Publishing Trip Cancelled event for tripID: %s riderID: %s driverID: %s driverAccepted: %t", tripID, riderID, driverID, driverAccepted)
	return p.rabbitMQ.PublishMessage(ctx, contracts.TripEventCancelled, contracts.AmqpMessage{
		OwnerID: riderID, // ownerID is not used by the cancel consumer, but required by the envelope
		Data:    data,
	})
}
