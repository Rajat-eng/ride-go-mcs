package redis

import (
	"context"
	"time"

	"ride-sharing/services/payment-service/internal/domain"

	goredis "github.com/redis/go-redis/v9"
)

const sessionTTL = 24 * time.Hour

type sessionStore struct {
	rdb *goredis.Client
}

func NewSessionStore(rdb *goredis.Client) domain.SessionStore {
	return &sessionStore{rdb: rdb}
}

func (s *sessionStore) Get(ctx context.Context, tripID string) (string, error) {
	val, err := s.rdb.Get(ctx, key(tripID)).Result()
	if err == goredis.Nil {
		return "", nil
	}
	return val, err
}

func (s *sessionStore) Set(ctx context.Context, tripID string, sessionID string) error {
	return s.rdb.Set(ctx, key(tripID), sessionID, sessionTTL).Err()
}

func key(tripID string) string {
	return "payment:session:" + tripID
}
