package service

import (
	"context"
	"log"

	"ride-sharing/services/chat-service/internal/domain"
)

// ChatService orchestrates message persistence and delivery acknowledgement.
type ChatService struct {
	repo      domain.MessageRepository
	publisher domain.MessagePublisher
}

func New(repo domain.MessageRepository, publisher domain.MessagePublisher) *ChatService {
	return &ChatService{repo: repo, publisher: publisher}
}

// HandleIncoming persists a new message and publishes a delivery receipt.
func (s *ChatService) HandleIncoming(ctx context.Context, msg *domain.Message) error {
	if err := s.repo.Save(ctx, msg); err != nil {
		log.Printf("chat-service: failed to persist message %s: %v", msg.ID, err)
		return err
	}

	if err := s.publisher.PublishDelivered(ctx, msg.ID, msg.TripID); err != nil {
		// Non-fatal: message is persisted; the receipt is best-effort.
		log.Printf("chat-service: failed to publish delivery receipt for %s: %v", msg.ID, err)
	}

	return nil
}

// GetHistory returns the last n messages for a trip, newest first.
func (s *ChatService) GetHistory(ctx context.Context, tripID string, limit int) ([]*domain.Message, error) {
	return s.repo.GetByTripID(ctx, tripID, limit)
}
