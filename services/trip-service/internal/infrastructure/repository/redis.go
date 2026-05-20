package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ride-sharing/services/trip-service/internal/domain"

	"github.com/redis/go-redis/v9"
)

const (
	rideFareTTL       = 20 * time.Minute
	farePreviewPrefix = "fare_preview:"
)

type redisRideFareRepository struct {
	client *redis.Client
}

func NewRedisRideFareRepository(client *redis.Client) *redisRideFareRepository {
	return &redisRideFareRepository{client: client}
}

// SaveFarePreview stores all package options + route as a single Redis key per user.
// A new preview overwrites the previous one, handling cancel + retry automatically.
func (r *redisRideFareRepository) SaveFarePreview(ctx context.Context, preview *domain.FarePreview) error {
	data, err := json.Marshal(preview)
	if err != nil {
		return fmt.Errorf("failed to marshal fare preview: %w", err)
	}

	key := farePreviewPrefix + preview.UserID
	return r.client.Set(ctx, key, data, rideFareTTL).Err()
}

// GetRideFareByID fetches the user's fare preview and finds the specific fare by ID.
// It reconstructs a full RideFareModel by attaching the shared route from the preview.
func (r *redisRideFareRepository) GetRideFareByID(ctx context.Context, fareID, userID string) (*domain.RideFareModel, error) {
	key := farePreviewPrefix + userID
	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, fmt.Errorf("no fare preview found for user: %s", userID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get fare preview from redis: %w", err)
	}

	var preview domain.FarePreview
	if err := json.Unmarshal(data, &preview); err != nil {
		return nil, fmt.Errorf("failed to unmarshal fare preview: %w", err)
	}

	for _, opt := range preview.Fares {
		if opt.ID.Hex() == fareID {
			// reconstruct full RideFareModel — userID and route come from the preview
			return &domain.RideFareModel{
				ID:                opt.ID,
				UserID:            preview.UserID,
				PackageSlug:       opt.PackageSlug,
				TotalPriceInCents: opt.TotalPriceInCents,
				Route:             preview.Route,
			}, nil
		}
	}

	return nil, fmt.Errorf("fare not found with ID: %s", fareID)
}
