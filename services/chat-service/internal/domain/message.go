package domain

import "context"

// Message represents a single chat message in the domain layer.
type Message struct {
	ID        string
	TripID    string
	SenderID  string
	Text      string
	SentAt    int64
	Delivered bool
}

// MessageRepository defines the persistence contract for chat messages.
type MessageRepository interface {
	Save(ctx context.Context, msg *Message) error
	GetByTripID(ctx context.Context, tripID string, limit int) ([]*Message, error)
	MarkDelivered(ctx context.Context, messageID string) error
}

// MessagePublisher notifies downstream consumers that a message was persisted.
// Defined here in the domain so the service layer can depend on the interface
// without creating an import cycle with the infrastructure/events package.
type MessagePublisher interface {
	PublishDelivered(ctx context.Context, messageID, tripID string) error
}
