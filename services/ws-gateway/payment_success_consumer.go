package main

import (
	"context"
	"encoding/json"
	"log"

	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"

	"github.com/rabbitmq/amqp091-go"
)

type paymentSuccessConsumer struct {
	rabbitmq    *messaging.RabbitMQ
	connManager *messaging.RedisConnectionManager
}

func newPaymentSuccessConsumer(rabbitmq *messaging.RabbitMQ, connManager *messaging.RedisConnectionManager) *paymentSuccessConsumer {
	return &paymentSuccessConsumer{rabbitmq: rabbitmq, connManager: connManager}
}

func (c *paymentSuccessConsumer) Start() error {
	return c.rabbitmq.ConsumeMessages(messaging.NotifyTripCompletedQueue, func(ctx context.Context, msg amqp091.Delivery) error {
		var envelope contracts.AmqpMessage
		if err := json.Unmarshal(msg.Body, &envelope); err != nil {
			log.Printf("paymentSuccessConsumer: failed to unmarshal envelope: %v", err)
			return err
		}

		var payload struct {
			TripID string `json:"tripID"`
		}
		if err := json.Unmarshal(envelope.Data, &payload); err != nil {
			log.Printf("paymentSuccessConsumer: failed to unmarshal payload: %v", err)
			return err
		}

		if payload.TripID == "" || envelope.OwnerID == "" {
			log.Printf("paymentSuccessConsumer: missing tripID or ownerID, skipping")
			return nil
		}

		if err := c.connManager.ClearTripChatPair(payload.TripID); err != nil {
			log.Printf("paymentSuccessConsumer: failed to clear trip chat pair for trip %s: %v", payload.TripID, err)
		}

		tripRoomID := "trip:" + payload.TripID
		chatRoomID := tripRoomID + ":chat"
		c.connManager.LeaveUserFromRoom(envelope.OwnerID, tripRoomID)
		c.connManager.LeaveUserFromRoom(envelope.OwnerID, chatRoomID)

		wsMsg := contracts.WSMessage{
			Type:  contracts.TripEventCompleted,
			Topic: tripRoomID,
			Data: map[string]string{
				"tripID": payload.TripID,
			},
		}

		if err := c.connManager.SendMessage(envelope.OwnerID, wsMsg); err != nil {
			log.Printf("paymentSuccessConsumer: failed to notify user %s: %v", envelope.OwnerID, err)
		}

		log.Printf("paymentSuccessConsumer: trip %s completed — notified user %s", payload.TripID, envelope.OwnerID)
		return nil
	})
}
