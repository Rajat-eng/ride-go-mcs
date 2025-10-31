package messaging

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"ride-sharing/shared/contracts"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

type RedisConnectionManager struct {
	localCM *ConnectionManager
	rdb     *redis.Client
	ctx     context.Context
	subs    map[string]*redis.PubSub
	mu      sync.Mutex
}

func NewRedisConnectionManager(rdb *redis.Client) *RedisConnectionManager {
	return &RedisConnectionManager{
		localCM: NewConnectionManager(),
		rdb:     rdb,
		ctx:     context.Background(),
		subs:    make(map[string]*redis.PubSub),
	}
}

// Add new WebSocket connection and subscribe to Redis channel
func (rcm *RedisConnectionManager) Add(userID string, conn *websocket.Conn) {
	rcm.localCM.Add(userID, conn)

	channel := "driver:" + userID + ":events"
	pubsub := rcm.rdb.Subscribe(rcm.ctx, channel)

	rcm.mu.Lock()
	rcm.subs[userID] = pubsub // store subscription to close later
	rcm.mu.Unlock()

	go func() {
		for msg := range pubsub.Channel() {
			var wsMsg contracts.WSMessage
			if err := json.Unmarshal([]byte(msg.Payload), &wsMsg); err != nil {
				log.Printf("Invalid Redis message: %v", err)
				continue
			}
			if err := rcm.localCM.SendMessage(userID, wsMsg); err != nil {
				log.Printf("Error sending WS message to %s: %v", userID, err)
			}
		}
	}()

	log.Printf("Subscribed Redis channel for user %s", userID)
}

// Remove WebSocket and unsubscribe
func (rcm *RedisConnectionManager) Remove(userID string) {
	rcm.localCM.Remove(userID)

	rcm.mu.Lock()
	if sub, ok := rcm.subs[userID]; ok {
		sub.Close()
		delete(rcm.subs, userID)
	}
	rcm.mu.Unlock()
	log.Printf("Unsubscribed and removed connection for %s", userID)
}

// SendMessage – if connection local → send directly, else publish to Redis
func (rcm *RedisConnectionManager) SendMessage(userID string, msg contracts.WSMessage) error {
	// try local

	log.Printf("Attempting to send message to user %s locally with message %v", userID, msg)
	if err := rcm.localCM.SendMessage(userID, msg); err == nil {
		return nil
	}

	// fallback to Redis publish
	bytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	log.Printf("Publishing to Redis channel for user %s: %s", userID, string(bytes))

	channel := "driver:" + userID + ":events"
	return rcm.rdb.Publish(rcm.ctx, channel, bytes).Err()
}

// Upgrade WebSocket request
func (rcm *RedisConnectionManager) Upgrade(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	return rcm.localCM.Upgrade(w, r)
}
