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

		driverID := payload.DriverID
		if driverID == "" {
			resolvedDriverID, err := c.connManager.GetActiveDriver(payload.RiderID)
			if err != nil {
				log.Printf("cancelConsumer: failed to resolve active driver for rider %s: %v", payload.RiderID, err)
			} else {
				driverID = resolvedDriverID
			}
		}

		wsMsg := contracts.WSMessage{
			Type:  contracts.TripEventCancelled,
			Topic: "trip:" + payload.TripID,
			Data: map[string]string{
				"tripID": payload.TripID,
			},
		}

		shouldTearDownTripStreams := payload.DriverAccepted || driverID != ""
		if shouldTearDownTripStreams {
			log.Printf("cancelConsumer: closing chat/location streams for trip %s (driverAccepted=%t driverID=%s)", payload.TripID, payload.DriverAccepted, driverID)
			// After acceptance, cancellation must fully terminate trip chat + location relay.
			if err := c.connManager.ClearTripChatPair(payload.TripID); err != nil {
				log.Printf("cancelConsumer: failed to clear trip chat pair for trip %s: %v", payload.TripID, err)
				// non-fatal — continue to notify sockets
			}

			// Force local room cleanup for both participants so stale channels are closed
			// even when clients don't send explicit unsubscribe messages.
			tripRoomID := "trip:" + payload.TripID
			chatRoomID := tripRoomID + ":chat"
			c.connManager.LeaveUserFromRoom(payload.RiderID, tripRoomID)
			c.connManager.LeaveUserFromRoom(payload.RiderID, chatRoomID)
			if driverID != "" {
				c.connManager.LeaveUserFromRoom(driverID, tripRoomID)
				c.connManager.LeaveUserFromRoom(driverID, chatRoomID)
			}
		} else {
			log.Printf("cancelConsumer: trip %s cancelled before driver acceptance; preserving socket room state", payload.TripID)
		}

		// Notify rider
		if err := c.connManager.SendMessage(payload.RiderID, wsMsg); err != nil {
			log.Printf("cancelConsumer: failed to notify rider %s: %v", payload.RiderID, err)
		}

		// Notify driver if one was assigned
		if driverID != "" {
			if err := c.connManager.SendMessage(driverID, wsMsg); err != nil {
				log.Printf("cancelConsumer: failed to notify driver %s: %v", driverID, err)
			}
		}

		log.Printf("cancelConsumer: trip %s cancelled — notified rider %s driver %s", payload.TripID, payload.RiderID, driverID)
		return nil
	})
}
