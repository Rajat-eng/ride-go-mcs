package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/rabbitmq/amqp091-go"

	"ride-sharing/shared/contracts"
	pb "ride-sharing/shared/proto/trip"
)

type QueueConsumer struct {
	rb        *RabbitMQ
	connMgr   *RedisConnectionManager
	queueName string
}

type wsCanonicalizer func(json.RawMessage) (any, error)
type wsTopicResolver func(json.RawMessage) string

var skippedWSRoutingKeys = map[string]struct{}{
	contracts.ChatEventDelivered: {},
}

var wsCanonicalizers = map[string]wsCanonicalizer{
	contracts.TripEventCreated:             canonicalizeTripPayload,
	contracts.DriverCmdTripRequest:         canonicalizeTripPayload,
	contracts.TripEventDriverAssigned:      canonicalizeTripPayload,
	contracts.TripEventDriverNotInterested: canonicalizeTripPayload,
	contracts.PaymentEventSessionCreated:   canonicalizePaymentSessionCreated,
	contracts.DriverEventLocation:          canonicalizeDriverLocation,
}

var wsTopicResolvers = map[string]wsTopicResolver{
	contracts.TripEventDriverNotInterested: resolveTopicFromWrappedTrip,
	contracts.TripEventNoDriversFound:      resolveTopicFromWrappedTrip,
	contracts.TripEventDriverAssigned:      resolveTopicFromFlatTrip,
	contracts.PaymentEventSessionCreated:   resolveTopicFromPayment,
}

func sanitizeTripForWS(trip *pb.Trip) error {
	if trip == nil {
		return fmt.Errorf("trip payload is nil")
	}
	if trip.Id == "" {
		return fmt.Errorf("trip.id is required")
	}
	if trip.UserID == "" {
		return fmt.Errorf("trip.userID is required")
	}
	if trip.Status == "" {
		return fmt.Errorf("trip.status is required")
	}

	// Driver is optional; if present it must be complete for strict frontend schemas.
	if trip.Driver != nil && (trip.Driver.Id == "" || trip.Driver.Name == "") {
		trip.Driver = nil
	}

	return nil
}

func canonicalTripJSON(payload json.RawMessage) (json.RawMessage, error) {
	var wrapped struct {
		Trip *pb.Trip `json:"trip"`
	}
	if err := json.Unmarshal(payload, &wrapped); err == nil && wrapped.Trip != nil {
		if err := sanitizeTripForWS(wrapped.Trip); err != nil {
			return nil, err
		}
		b, err := json.Marshal(wrapped.Trip)
		if err != nil {
			return nil, err
		}
		return b, nil
	}

	var trip pb.Trip
	if err := json.Unmarshal(payload, &trip); err == nil && trip.Id != "" {
		if err := sanitizeTripForWS(&trip); err != nil {
			return nil, err
		}
		b, err := json.Marshal(&trip)
		if err != nil {
			return nil, err
		}
		return b, nil
	}

	return nil, fmt.Errorf("unable to parse trip payload")
}

func canonicalizeTripPayload(payload json.RawMessage) (any, error) {
	tripJSON, err := canonicalTripJSON(payload)
	if err != nil {
		return nil, fmt.Errorf("canonicalize trip payload: %w", err)
	}
	return tripJSON, nil
}

func canonicalizePaymentSessionCreated(payload json.RawMessage) (any, error) {
	var payment PaymentEventSessionCreatedData
	if err := json.Unmarshal(payload, &payment); err != nil {
		return nil, fmt.Errorf("invalid payment session payload: %w", err)
	}
	if payment.TripID == "" || payment.SessionID == "" {
		return nil, fmt.Errorf("invalid payment session payload: tripID and sessionID are required")
	}
	return payment, nil
}

func canonicalizeDriverLocation(payload json.RawMessage) (any, error) {
	var location DriverLocationEventData
	if err := json.Unmarshal(payload, &location); err != nil {
		return nil, fmt.Errorf("invalid driver location payload: %w", err)
	}
	return location, nil
}

func canonicalizeWSData(routingKey string, payload json.RawMessage) (data any, skip bool, err error) {
	if _, shouldSkip := skippedWSRoutingKeys[routingKey]; shouldSkip {
		return nil, true, nil
	}

	canonicalizer, ok := wsCanonicalizers[routingKey]
	if !ok {
		return payload, false, nil
	}

	data, err = canonicalizer(payload)
	if err != nil {
		return nil, false, err
	}

	return data, false, nil
}

func tripTopic(tripID string) string {
	if tripID == "" {
		return ""
	}
	return "trip:" + tripID
}

func resolveTopicFromWrappedTrip(payload json.RawMessage) string {
	var event struct {
		Trip struct {
			ID string `json:"id"`
		} `json:"trip"`
	}
	if err := json.Unmarshal(payload, &event); err == nil {
		return tripTopic(event.Trip.ID)
	}
	return ""
}

func resolveTopicFromFlatTrip(payload json.RawMessage) string {
	var trip struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(payload, &trip); err == nil {
		return tripTopic(trip.ID)
	}
	return ""
}

func resolveTopicFromPayment(payload json.RawMessage) string {
	var payment struct {
		TripID string `json:"tripID"`
	}
	if err := json.Unmarshal(payload, &payment); err == nil {
		return tripTopic(payment.TripID)
	}
	return ""
}

func deriveTripTopic(routingKey string, payload json.RawMessage) string {
	resolver, ok := wsTopicResolvers[routingKey]
	if !ok {
		return ""
	}
	return resolver(payload)
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

		data, skip, err := canonicalizeWSData(msg.RoutingKey, amqpMsg.Data)
		if err != nil {
			log.Printf("Failed to canonicalize message for user %s type %s: %v", userID, msg.RoutingKey, err)
			return err
		}
		if skip {
			log.Printf("Skipping WS fanout for internal event type '%s'", msg.RoutingKey)
			return nil
		}

		clientMsg := contracts.WSMessage{
			Type:  msg.RoutingKey,
			Topic: deriveTripTopic(msg.RoutingKey, amqpMsg.Data),
			Data:  data,
		}

		if err := qc.connMgr.SendMessage(userID, clientMsg); err != nil {
			log.Printf("Failed to deliver message for user %s: %v", userID, err)
			return err
		}

		log.Printf("Processed message of type '%s' for user %s", clientMsg.Type, userID)
		return nil
	})
}
