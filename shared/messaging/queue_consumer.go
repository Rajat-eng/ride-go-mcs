package messaging

import (
	"encoding/json"
	"log"

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
	msgs, err := qc.rb.Channel.Consume(
		qc.queueName,
		"",
		true,  // auto-ack
		false, // not exclusive
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	go func() {
		for msg := range msgs {
			var amqpMsg contracts.AmqpMessage
			if err := json.Unmarshal(msg.Body, &amqpMsg); err != nil {
				log.Println("❌ Failed to unmarshal AMQP message:", err)
				continue
			}

			userID := amqpMsg.OwnerID
			if userID == "" {
				log.Println("⚠️ Message has no owner ID, skipping")
				continue
			}
			log.Printf("Processing message in consumer for user %s from queue %s", userID, qc.queueName)

			clientMsg := contracts.WSMessage{
				Type:  msg.RoutingKey, // e.g. "trip.request", "trip.cancel", etc.
				Topic: deriveTripTopic(msg.RoutingKey, amqpMsg.Data),
				Data:  amqpMsg.Data,
			}

			// Send via Redis-aware manager
			if err := qc.connMgr.SendMessage(userID, clientMsg); err != nil {
				log.Printf("⚠️ Failed to deliver message for user %s: %v", userID, err)
			} else {
				log.Printf("✅ Delivered message of type '%s' to user %s", clientMsg.Type, userID)
			}
		}
	}()

	return nil
}
