package main

import "encoding/json"

// Incoming WebSocket message types from clients.
const (
	WSChatMessageSend     = "chat.message.send"
	WSChatMessageReceived = "chat.message.received"
	WSChatMessageAck      = "chat.message.ack" // delivery receipt sent to sender
)

// wsIncomingMessage is the top-level envelope for every client → server frame.
type wsIncomingMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// tripChatMessageData is the payload the client sends with chat.message.send.
type tripChatMessageData struct {
	TripID    string `json:"tripID"`
	MessageID string `json:"messageID,omitempty"` // client-generated idempotency key
	Text      string `json:"text"`
}

// tripChatMessageReceivedData is the payload broadcast to room members.
type tripChatMessageReceivedData struct {
	TripID    string `json:"tripID"`
	RoomID    string `json:"roomID"`
	SenderID  string `json:"senderID"`
	MessageID string `json:"messageID,omitempty"`
	Text      string `json:"text"`
	SentAt    int64  `json:"sentAt"`
}
