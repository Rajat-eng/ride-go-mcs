package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"ride-sharing/shared/contracts"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

var (
	ErrTripChatPairNotFound = errors.New("trip chat pair not found")
	ErrTripChatUnauthorized = errors.New("sender is not part of trip chat pair")
)

type RedisConnectionManager struct {
	localCM      *ConnectionManager
	rdb          *redis.Client
	ctx          context.Context
	subs         map[string]*redis.PubSub
	topicsByUser map[string]map[string]struct{} // userID → set of subscribed topics
	mu           sync.Mutex
}

func NewRedisConnectionManager(rdb *redis.Client) *RedisConnectionManager {
	return &RedisConnectionManager{
		localCM:      NewConnectionManager(),
		rdb:          rdb,
		ctx:          context.Background(),
		subs:         make(map[string]*redis.PubSub),
		topicsByUser: make(map[string]map[string]struct{}),
	}
}

// SubscribeTopic registers interest in a topic for a connected user.
// Messages tagged with this topic will be delivered to the user's WS connection.
func (rcm *RedisConnectionManager) SubscribeTopic(userID, topic string) {
	rcm.mu.Lock()
	defer rcm.mu.Unlock()
	if _, ok := rcm.topicsByUser[userID]; !ok {
		rcm.topicsByUser[userID] = make(map[string]struct{})
	}
	rcm.topicsByUser[userID][topic] = struct{}{}
	log.Printf("User %s subscribed to topic %s", userID, topic)
}

// UnsubscribeTopic removes interest in a topic for a user.
func (rcm *RedisConnectionManager) UnsubscribeTopic(userID, topic string) {
	rcm.mu.Lock()
	defer rcm.mu.Unlock()
	if topics, ok := rcm.topicsByUser[userID]; ok {
		delete(topics, topic)
	}
	log.Printf("User %s unsubscribed from topic %s", userID, topic)
}

// isTopicAllowed returns true when:
//   - the message has no topic (global/system messages always delivered), or
//   - the user has explicitly subscribed to the message's topic.
func (rcm *RedisConnectionManager) isTopicAllowed(userID, topic string) bool {
	if topic == "" {
		return true
	}
	rcm.mu.Lock()
	defer rcm.mu.Unlock()
	topics, ok := rcm.topicsByUser[userID]
	if !ok {
		return false
	}
	_, subscribed := topics[topic]
	return subscribed
}

// Add new WebSocket connection and subscribe to Redis channel
func (rcm *RedisConnectionManager) Add(userID string, conn *websocket.Conn) {
	rcm.localCM.Add(userID, conn)

	channel := "user:" + userID + ":events"
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
			if !rcm.isTopicAllowed(userID, wsMsg.Topic) {
				log.Printf("Topic %q not subscribed for user %s — skipping", wsMsg.Topic, userID)
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
	delete(rcm.topicsByUser, userID)
	rcm.mu.Unlock()
	log.Printf("Unsubscribed and removed connection for %s", userID)
}

// SendMessage – if connection local → send directly (topic-filtered), else publish to Redis.
// Messages with a non-empty topic are only delivered when the user has subscribed to that topic.
// Messages with no topic (system/global broadcasts) are always delivered.
func (rcm *RedisConnectionManager) SendMessage(userID string, msg contracts.WSMessage) error {
	if !rcm.isTopicAllowed(userID, msg.Topic) {
		log.Printf("Topic %q not subscribed for user %s — not sending", msg.Topic, userID)
		return nil
	}

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

	channel := "user:" + userID + ":events"
	return rcm.rdb.Publish(rcm.ctx, channel, bytes).Err()
}

// Upgrade WebSocket request
func (rcm *RedisConnectionManager) Upgrade(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	return rcm.localCM.Upgrade(w, r)
}

func tripChatRiderKey(tripID string) string {
	return "trip:" + tripID + ":chat:rider"
}

func tripChatDriverKey(tripID string) string {
	return "trip:" + tripID + ":chat:driver"
}

// SetTripChatPair stores the rider/driver pair for a trip so WS chat can be authorized and routed.
func (rcm *RedisConnectionManager) SetTripChatPair(tripID, riderID, driverID string, ttl time.Duration) error {
	if tripID == "" || riderID == "" || driverID == "" {
		return fmt.Errorf("tripID, riderID and driverID are required")
	}

	pipe := rcm.rdb.TxPipeline()
	pipe.Set(rcm.ctx, tripChatRiderKey(tripID), riderID, ttl)   // registers tripID/riderID in Redis with a TTL so that it expires after some time and doesn't take up space indefinitely
	pipe.Set(rcm.ctx, tripChatDriverKey(tripID), driverID, ttl) // registers tripID/driverID in Redis with a TTL so that it expires after some time and doesn't take up space indefinitely
	_, err := pipe.Exec(rcm.ctx)
	return err
}

// ResolveTripChatPeer returns the peer user ID if senderID is one of the assigned trip participants.
func (rcm *RedisConnectionManager) ResolveTripChatPeer(tripID, senderID string) (string, error) {
	if tripID == "" || senderID == "" {
		return "", fmt.Errorf("tripID and senderID are required")
	}

	values, err := rcm.rdb.MGet(rcm.ctx, tripChatRiderKey(tripID), tripChatDriverKey(tripID)).Result() // mget means multi get, it gets both rider and driver id in one call
	if err != nil {
		return "", err
	}

	riderID, riderOk := values[0].(string)
	driverID, driverOk := values[1].(string)
	if !riderOk || !driverOk || riderID == "" || driverID == "" {
		return "", ErrTripChatPairNotFound
	}

	switch senderID {
	case riderID:
		return driverID, nil
	case driverID:
		return riderID, nil
	default:
		return "", ErrTripChatUnauthorized
	}
}
