package main

import (
	"context"
	"encoding/json"
	"log"

	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"

	"github.com/rabbitmq/amqp091-go"
)

type cancelConsumer struct {
	rabbitmq    *messaging.RabbitMQ
	connManager *messaging.RedisConnectionManager
}

func newCancelConsumer(rabbitmq *messaging.RabbitMQ, connManager *messaging.RedisConnectionManager) *cancelConsumer {
	return &cancelConsumer{rabbitmq: rabbitmq, connManager: connManager}
}

func (c *cancelConsumer) Start() error {
	return c.rabbitmq.ConsumeMessages(messaging.NotifyTripCancelledQueue, func(ctx context.Context, msg amqp091.Delivery) error {
		var envelope contracts.AmqpMessage
		if err := json.Unmarshal(msg.Body, &envelope); err != nil {
			log.Printf("cancelConsumer: failed to unmarshal envelope: %v", err)
			return err
		}

		var payload messaging.TripCancelledData
		if err := json.Unmarshal(envelope.Data, &payload); err != nil {
			log.Printf("cancelConsumer: failed to unmarshal payload: %v", err)
			return err
		}

		if payload.TripID == "" || payload.RiderID == "" {
			log.Printf("cancelConsumer: missing tripID or riderID, skipping")
			return nil
		}

		wsMsg := contracts.WSMessage{
			Type:  contracts.TripEventCancelled,
			Topic: "trip:" + payload.TripID,
			Data: map[string]string{
				"tripID": payload.TripID,
			},
		}

		// Clear Redis chat pair — this also removes active_rider and active_driver keys
		// so the driver becomes available for new trips immediately.
		if err := c.connManager.ClearTripChatPair(payload.TripID); err != nil {
			log.Printf("cancelConsumer: failed to clear trip chat pair for trip %s: %v", payload.TripID, err)
			// non-fatal — continue to notify sockets
		}

		// Notify rider
		if err := c.connManager.SendMessage(payload.RiderID, wsMsg); err != nil {
			log.Printf("cancelConsumer: failed to notify rider %s: %v", payload.RiderID, err)
		}

		// Notify driver if one was assigned
		if payload.DriverID != "" {
			if err := c.connManager.SendMessage(payload.DriverID, wsMsg); err != nil {
				log.Printf("cancelConsumer: failed to notify driver %s: %v", payload.DriverID, err)
			}
		}

		log.Printf("cancelConsumer: trip %s cancelled — notified rider %s driver %s", payload.TripID, payload.RiderID, payload.DriverID)
		return nil
	})
}
