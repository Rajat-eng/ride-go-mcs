package contracts

import "encoding/json"

// Control message types – legacy topic model (kept for backward compat).
const (
	WSTopicSubscribe   = "ws.topic.subscribe"
	WSTopicUnsubscribe = "ws.topic.unsubscribe"
)

// Room-based control frames (preferred for scalable chat).
const (
	WSRoomJoin  = "ws.room.join"
	WSRoomLeave = "ws.room.leave"
)

// WSMessage is the envelope for every WebSocket message.
// RoomID optionally scopes the message to a chat room (e.g. "trip:{id}:chat").
type WSMessage struct {
	Type   string `json:"type"`
	Topic  string `json:"topic,omitempty"`  // legacy – kept for non-chat system events
	RoomID string `json:"roomID,omitempty"` // room-scoped broadcast
	Data   any    `json:"data"`
}

// WSTopicControlData is the payload for legacy subscribe/unsubscribe frames.
type WSTopicControlData struct {
	Topic string `json:"topic"`
}

// WSRoomControlData is the payload for ws.room.join / ws.room.leave frames.
type WSRoomControlData struct {
	RoomID string `json:"roomID"`
}

type WSDriverMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}
