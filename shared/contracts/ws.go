package contracts

import "encoding/json"

// Control message types for topic multiplexing over a single WebSocket connection.
const (
	WSTopicSubscribe   = "ws.topic.subscribe"
	WSTopicUnsubscribe = "ws.topic.unsubscribe"
)

// WSMessage is the message structure for the WebSocket.
// Topic optionally scopes the message to a logical channel (e.g. "trip:<id>").
type WSMessage struct {
	Type  string `json:"type"`
	Topic string `json:"topic,omitempty"`
	Data  any    `json:"data"`
}

// WSTopicControlData is the payload for subscribe/unsubscribe control frames.
type WSTopicControlData struct {
	Topic string `json:"topic"`
}

type WSDriverMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}
