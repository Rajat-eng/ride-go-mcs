package service

import (
	"context"
	"fmt"
	"log"
	"regexp"

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

// ValidateMessageContent validates message content constraints
func ValidateMessageContent(content string, maxLength int) error {
	if content == "" {
		return fmt.Errorf("message content cannot be empty")
	}
	if len(content) > maxLength {
		return fmt.Errorf("message exceeds maximum length of %d characters", maxLength)
	}
	return nil
}

// ValidateTripID validates MongoDB ObjectID format for trip ID
func ValidateTripID(tripID string) error {
	if tripID == "" {
		return fmt.Errorf("trip ID cannot be empty")
	}
	// MongoDB ObjectID is 24 hex characters
	if !regexp.MustCompile(`^[a-f0-9]{24}$`).MatchString(tripID) {
		return fmt.Errorf("invalid trip ID format")
	}
	return nil
}

// ValidateParticipant validates participant ID
func ValidateParticipant(participantID string) error {
	if participantID == "" {
		return fmt.Errorf("participant ID cannot be empty")
	}
	return nil
}
