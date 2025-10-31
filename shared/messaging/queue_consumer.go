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
				Type: msg.RoutingKey, // e.g. "trip.request", "trip.cancel", etc.
				Data: amqpMsg.Data,
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
