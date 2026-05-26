package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"

	"github.com/google/uuid"
)

// tripChatRoomID returns the canonical room ID for a trip's chat channel.
func tripChatRoomID(tripID string) string {
	if tripID == "" {
		return ""
	}
	return "trip:" + tripID + ":chat"
}

// relayTripChatMessage validates the sender's membership in the trip chat,
// broadcasts the message to the room (low-latency local + Redis cross-node),
// and then publishes to the chat-service queue for durable persistence.
func relayTripChatMessage(
	ctx context.Context,
	connManager *messaging.RedisConnectionManager,
	rb *messaging.RabbitMQ,
	socketID, senderID string,
	rawData json.RawMessage,
) error {
	var payload tripChatMessageData
	if err := json.Unmarshal(rawData, &payload); err != nil {
		return err
	}
	if payload.TripID == "" || payload.Text == "" {
		return nil
	}

	// Ensure sender is an authorised trip participant before broadcasting.
	if _, err := connManager.ResolveTripChatPeer(payload.TripID, senderID); err != nil {
		return err
	}

	msgID := payload.MessageID
	if msgID == "" {
		msgID = uuid.New().String()
	}
	sentAt := time.Now().Unix()
	roomID := tripChatRoomID(payload.TripID)

	wsMsg := contracts.WSMessage{
		Type:   WSChatMessageReceived,
		RoomID: roomID,
		Data: tripChatMessageReceivedData{
			TripID:    payload.TripID,
			RoomID:    roomID,
			SenderID:  senderID,
			MessageID: msgID,
			Text:      payload.Text,
			SentAt:    sentAt,
		},
	}

	// Broadcast to all sockets in the room (sender + peer, local + cross-node).
	if err := connManager.BroadcastToRoom(roomID, wsMsg); err != nil {
		log.Printf("BroadcastToRoom %s error: %v", roomID, err)
	}

	// Publish to chat-service for durable storage (fire-and-forget from WS perspective).
	chatData, _ := json.Marshal(messaging.ChatMessageData{
		MessageID: msgID,
		TripID:    payload.TripID,
		SenderID:  senderID,
		Text:      payload.Text,
		SentAt:    sentAt,
	})
	if err := rb.PublishMessage(ctx, contracts.ChatCmdSend, contracts.AmqpMessage{
		OwnerID: senderID,
		Data:    chatData,
	}); err != nil {
		log.Printf("Failed to publish chat message to persistence queue: %v", err)
	}

	return nil
}
