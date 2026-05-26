package events

import (
	"context"
	"encoding/json"
	"log"

	"ride-sharing/services/chat-service/internal/domain"
	"ride-sharing/services/chat-service/internal/service"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"
)

// Consumer reads chat.cmd.send messages from RabbitMQ and delegates to ChatService.
type Consumer struct {
	rb          *messaging.RabbitMQ
	chatService *service.ChatService
}

func NewConsumer(rb *messaging.RabbitMQ, chatService *service.ChatService) *Consumer {
	return &Consumer{rb: rb, chatService: chatService}
}

// Start launches a goroutine that processes incoming chat messages.
func (c *Consumer) Start(ctx context.Context) error {
	msgs, err := c.rb.Channel.Consume(
		messaging.ChatCmdSendQueue,
		"chat-service-consumer",
		false, // manual ack for durability
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgs:
				if !ok {
					return
				}
				if err := c.handleMessage(ctx, msg.Body); err != nil {
					log.Printf("chat-service: failed to handle message: %v", err)
					msg.Nack(false, true) // requeue on error
					continue
				}
				msg.Ack(false)
			}
		}
	}()

	log.Println("chat-service: consumer started on", messaging.ChatCmdSendQueue)
	return nil
}

func (c *Consumer) handleMessage(ctx context.Context, body []byte) error {
	var amqpMsg contracts.AmqpMessage
	if err := json.Unmarshal(body, &amqpMsg); err != nil {
		return err
	}

	var data messaging.ChatMessageData
	if err := json.Unmarshal(amqpMsg.Data, &data); err != nil {
		return err
	}

	return c.chatService.HandleIncoming(ctx, &domain.Message{
		ID:       data.MessageID,
		TripID:   data.TripID,
		SenderID: data.SenderID,
		Text:     data.Text,
		SentAt:   data.SentAt,
	})
}
