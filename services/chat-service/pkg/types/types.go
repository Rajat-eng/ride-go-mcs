package types

// ChatMessage is the canonical chat message as stored in MongoDB and surfaced
// via the chat-service API.
type ChatMessage struct {
	ID        string `json:"id" bson:"_id"`
	TripID    string `json:"tripID" bson:"tripID"`
	SenderID  string `json:"senderID" bson:"senderID"`
	Text      string `json:"text" bson:"text"`
	SentAt    int64  `json:"sentAt" bson:"sentAt"`
	Delivered bool   `json:"delivered" bson:"delivered"`
}
