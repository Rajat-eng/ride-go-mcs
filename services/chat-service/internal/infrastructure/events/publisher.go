package events

import (
	"context"
	"encoding/json"
	"log"

	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"
)

// Publisher publishes chat domain events to RabbitMQ.
type Publisher struct {
	rb *messaging.RabbitMQ
}

func NewPublisher(rb *messaging.RabbitMQ) *Publisher {
	return &Publisher{rb: rb}
}

// PublishDelivered notifies ws-gateway (and any other subscriber) that a
// message has been durably stored. ws-gateway will forward the ack to the sender.
func (p *Publisher) PublishDelivered(ctx context.Context, messageID, tripID string) error {
	data, err := json.Marshal(messaging.ChatDeliveredData{
		MessageID: messageID,
		TripID:    tripID,
	})
	if err != nil {
		return err
	}
	log.Printf("chat-service: publishing delivered ack for message %s", messageID)
	return p.rb.PublishMessage(ctx, contracts.ChatEventDelivered, contracts.AmqpMessage{
		OwnerID: tripID, // ownerID = tripID so ws-gateway can route ack to the room
		Data:    data,
	})
}
