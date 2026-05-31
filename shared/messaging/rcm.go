package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"ride-sharing/shared/contracts"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var (
	ErrTripChatPairNotFound = errors.New("trip chat pair not found")
	ErrTripChatUnauthorized = errors.New("sender is not part of trip chat pair")
)

const (
	userEventStreamTTL    = 2 * time.Hour
	userEventStreamMaxLen = int64(500)
)

// RedisConnectionManager manages WebSocket connections across multiple gateway
// nodes using Redis Pub/Sub for cross-node message delivery.
//
// Room model (for chat):
//   - Each WebSocket connection has a unique socketID.
//   - Sockets may join named rooms (e.g. "trip:{id}:chat").
//   - Room messages are fanned out to all sockets in the room across all nodes
//     by publishing to the "room:{roomID}" Redis channel.
//
// User-direct model (for system/trip/payment events):
//   - Each user has a dedicated "user:{userID}:events" Redis channel.
//   - All sockets of the same user on this node receive the message.
type RedisConnectionManager struct {
	localCM     *ConnectionManager
	rdb         *redis.Client
	ctx         context.Context
	userSubs    map[string]*redis.PubSub       // userID  → user-direct channel sub
	roomSubs    map[string]*redis.PubSub       // roomID  → room broadcast sub
	socketRooms map[string]map[string]struct{} // socketID → set of roomIDs
	roomSockets map[string]map[string]struct{} // roomID   → set of local socketIDs
	tracer      trace.Tracer
	persisted   atomic.Uint64
	replayed    atomic.Uint64
	replayFails atomic.Uint64
	mu          sync.Mutex
}

func NewRedisConnectionManager(rdb *redis.Client) *RedisConnectionManager {
	return &RedisConnectionManager{
		localCM:     NewConnectionManager(),
		rdb:         rdb,
		ctx:         context.Background(),
		userSubs:    make(map[string]*redis.PubSub),
		roomSubs:    make(map[string]*redis.PubSub),
		socketRooms: make(map[string]map[string]struct{}),
		roomSockets: make(map[string]map[string]struct{}),
		tracer:      otel.GetTracerProvider().Tracer("redis-streams"),
	}
}

// Add registers a new WebSocket connection. socketID must be unique per
// connection (e.g. a UUID generated at upgrade time). The node subscribes to the
// user's direct Redis channel once — subsequent connections for the same user
// reuse the existing subscription.
func (rcm *RedisConnectionManager) Add(userID, socketID string, conn *websocket.Conn) {
	rcm.localCM.Add(socketID, userID, conn)

	replayPending := false
	rcm.mu.Lock()
	if _, already := rcm.userSubs[userID]; !already {
		pubsub := rcm.rdb.Subscribe(rcm.ctx, "user:"+userID+":events")
		rcm.userSubs[userID] = pubsub
		go rcm.runUserSubscription(userID, pubsub)
		replayPending = true
	}
	rcm.mu.Unlock()

	if replayPending {
		go rcm.replayUserStream(userID)
	}

	log.Printf("Added socket %s for user %s", socketID, userID)
}

func userEventStreamKey(userID string) string {
	return "user:" + userID + ":stream"
}

func (rcm *RedisConnectionManager) persistUserMessage(userID string, msg contracts.WSMessage) error {
	ctx, span := rcm.tracer.Start(rcm.ctx, "rcm.stream.persist",
		trace.WithAttributes(
			attribute.String("messaging.user_id", userID),
			attribute.String("messaging.destination", userEventStreamKey(userID)),
		),
	)
	defer span.End()

	payload, err := json.Marshal(msg)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	key := userEventStreamKey(userID)
	if err := rcm.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: key,
		MaxLen: userEventStreamMaxLen,
		Approx: true,
		Values: map[string]any{
			"payload": string(payload),
		},
	}).Err(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	_ = rcm.rdb.Expire(ctx, key, userEventStreamTTL).Err()
	total := rcm.persisted.Add(1)
	span.SetAttributes(
		attribute.Int("stream.persisted.count", 1),
		attribute.Int64("stream.persisted.total", int64(total)),
	)
	log.Printf("stream persisted count=1 total=%d user=%s stream=%s", total, userID, key)
	return nil
}

func (rcm *RedisConnectionManager) replayUserStream(userID string) {
	ctx, span := rcm.tracer.Start(rcm.ctx, "rcm.stream.replay",
		trace.WithAttributes(
			attribute.String("messaging.user_id", userID),
			attribute.String("messaging.destination", userEventStreamKey(userID)),
		),
	)
	defer span.End()

	key := userEventStreamKey(userID)
	entries, err := rcm.rdb.XRange(ctx, key, "-", "+").Result()
	if err != nil {
		rcm.replayFails.Add(1)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(attribute.Int64("stream.replay.failures.total", int64(rcm.replayFails.Load())))
		log.Printf("Failed to read pending stream for user %s: %v", userID, err)
		return
	}
	span.SetAttributes(attribute.Int("stream.replay.pending", len(entries)))
	if len(entries) == 0 {
		log.Printf("replayed count=0 replay failures=0 user=%s stream=%s", userID, key)
		return
	}

	acked := make([]string, 0, len(entries))
	replayedDelta := 0
	failuresDelta := 0
	for _, entry := range entries {
		raw, ok := entry.Values["payload"]
		if !ok {
			acked = append(acked, entry.ID)
			continue
		}

		payload, ok := raw.(string)
		if !ok {
			acked = append(acked, entry.ID)
			continue
		}

		var wsMsg contracts.WSMessage
		if err := json.Unmarshal([]byte(payload), &wsMsg); err != nil {
			acked = append(acked, entry.ID)
			continue
		}

		if err := rcm.localCM.SendMessage(userID, wsMsg); err != nil {
			failuresDelta++
			rcm.replayFails.Add(1)
			span.RecordError(err)
			log.Printf("Replay paused for user %s due to delivery error: %v", userID, err)
			break
		}

		replayedDelta++
		acked = append(acked, entry.ID)
	}

	if len(acked) > 0 {
		if err := rcm.rdb.XDel(ctx, key, acked...).Err(); err != nil {
			failuresDelta++
			rcm.replayFails.Add(1)
			span.RecordError(err)
			log.Printf("Failed to ack replayed stream events for user %s: %v", userID, err)
		}
	}

	totalReplayed := rcm.replayed.Add(uint64(replayedDelta))
	totalFailures := rcm.replayFails.Load()
	span.SetAttributes(
		attribute.Int("stream.replayed.count", replayedDelta),
		attribute.Int64("stream.replayed.total", int64(totalReplayed)),
		attribute.Int("stream.replay.failures", failuresDelta),
		attribute.Int64("stream.replay.failures.total", int64(totalFailures)),
	)
	if failuresDelta > 0 {
		span.SetStatus(codes.Error, "replay completed with failures")
	}
	log.Printf("replayed count=%d total=%d replay failures=%d total_failures=%d user=%s stream=%s", replayedDelta, totalReplayed, failuresDelta, totalFailures, userID, key)
}

func (rcm *RedisConnectionManager) runUserSubscription(userID string, pubsub *redis.PubSub) {
	for msg := range pubsub.Channel() {
		var wsMsg contracts.WSMessage
		if err := json.Unmarshal([]byte(msg.Payload), &wsMsg); err != nil {
			log.Printf("Invalid Redis message for user %s: %v", userID, err)
			continue
		}
		if err := rcm.localCM.SendMessage(userID, wsMsg); err != nil {
			log.Printf("Error delivering user message to %s: %v", userID, err)
		}
	}
}

// Remove tears down all state for a socket: leaves every room it was in and,
// when the user's last socket on this node disconnects, closes the user's Redis sub.
func (rcm *RedisConnectionManager) Remove(socketID string) {
	userID := rcm.localCM.Remove(socketID)

	rcm.mu.Lock()
	defer rcm.mu.Unlock()

	// Leave all rooms this socket was in.
	for roomID := range rcm.socketRooms[socketID] {
		if sockets, ok := rcm.roomSockets[roomID]; ok {
			delete(sockets, socketID)
			if len(sockets) == 0 {
				delete(rcm.roomSockets, roomID)
				if sub, ok := rcm.roomSubs[roomID]; ok {
					sub.Close()
					delete(rcm.roomSubs, roomID)
				}
			}
		}
	}
	delete(rcm.socketRooms, socketID)

	// Close the user-direct sub when their last socket on this node is gone.
	if userID != "" && !rcm.localCM.HasUser(userID) {
		if sub, ok := rcm.userSubs[userID]; ok {
			sub.Close()
			delete(rcm.userSubs, userID)
		}
	}

	log.Printf("Removed socket %s (user %s)", socketID, userID)
}

// JoinRoom subscribes a socket to a named room. The room's Redis channel is
// subscribed once per node; subsequent sockets in the same room reuse it.
func (rcm *RedisConnectionManager) JoinRoom(socketID, roomID string) {
	rcm.mu.Lock()
	defer rcm.mu.Unlock()

	if _, ok := rcm.socketRooms[socketID]; !ok {
		rcm.socketRooms[socketID] = make(map[string]struct{})
	}
	rcm.socketRooms[socketID][roomID] = struct{}{}

	if _, ok := rcm.roomSockets[roomID]; !ok {
		rcm.roomSockets[roomID] = make(map[string]struct{})
	}
	rcm.roomSockets[roomID][socketID] = struct{}{}

	if _, already := rcm.roomSubs[roomID]; !already {
		pubsub := rcm.rdb.Subscribe(rcm.ctx, "room:"+roomID)
		rcm.roomSubs[roomID] = pubsub
		go rcm.runRoomSubscription(roomID, pubsub)
	}

	log.Printf("Socket %s joined room %s", socketID, roomID)
}

// LeaveRoom removes a socket from a room and cleans up the Redis sub when the
// room has no more local sockets.
func (rcm *RedisConnectionManager) LeaveRoom(socketID, roomID string) {
	rcm.mu.Lock()
	defer rcm.mu.Unlock()

	if rooms, ok := rcm.socketRooms[socketID]; ok {
		delete(rooms, roomID)
	}
	if sockets, ok := rcm.roomSockets[roomID]; ok {
		delete(sockets, socketID)
		if len(sockets) == 0 {
			delete(rcm.roomSockets, roomID)
			if sub, ok := rcm.roomSubs[roomID]; ok {
				sub.Close()
				delete(rcm.roomSubs, roomID)
			}
		}
	}

	log.Printf("Socket %s left room %s", socketID, roomID)
}

func (rcm *RedisConnectionManager) runRoomSubscription(roomID string, pubsub *redis.PubSub) {
	for msg := range pubsub.Channel() {
		var wsMsg contracts.WSMessage
		if err := json.Unmarshal([]byte(msg.Payload), &wsMsg); err != nil {
			log.Printf("Invalid room message for room %s: %v", roomID, err)
			continue
		}
		rcm.deliverToRoomLocally(roomID, wsMsg)
	}
}

func (rcm *RedisConnectionManager) deliverToRoomLocally(roomID string, msg contracts.WSMessage) {
	rcm.mu.Lock()
	sockets := make([]string, 0, len(rcm.roomSockets[roomID]))
	for sid := range rcm.roomSockets[roomID] {
		sockets = append(sockets, sid)
	}
	rcm.mu.Unlock()

	for _, socketID := range sockets {
		if err := rcm.localCM.SendToSocket(socketID, msg); err != nil {
			log.Printf("Failed to deliver room %s message to socket %s: %v", roomID, socketID, err)
		}
	}
}

// BroadcastToRoom delivers a message to every socket in the room across all
// gateway nodes via the "room:{roomID}" Redis channel.
//
// Important: we avoid writing to local sockets directly here because each node
// with local room members is already subscribed to the room channel. Publishing
// once to Redis ensures every node (including the publisher) delivers the
// message exactly once and prevents duplicate delivery on the same node.
func (rcm *RedisConnectionManager) BroadcastToRoom(roomID string, msg contracts.WSMessage) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return rcm.rdb.Publish(rcm.ctx, "room:"+roomID, b).Err()
}

// SendMessage delivers a message to all sock ets of a user on this node.
// If the user has no local connection, it falls back to publishing on their
// Redis channel so another node can deliver it. This is the backward-compatible
// API used by QueueConsumer and other system-notification paths.
func (rcm *RedisConnectionManager) SendMessage(userID string, msg contracts.WSMessage) error {
	log.Printf("Attempting to send message to user %s: %+v", userID, msg)
	if err := rcm.localCM.SendMessage(userID, msg); err == nil {
		return nil
	}

	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	log.Printf("Publishing to Redis channel for user %s", userID)
	receivedBy, err := rcm.rdb.Publish(rcm.ctx, "user:"+userID+":events", b).Result()
	if err != nil {
		return err
	}
	if receivedBy == 0 {
		if err := rcm.persistUserMessage(userID, msg); err != nil {
			return err
		}
		log.Printf("No active subscriber for user %s; persisted message to stream", userID)
	}
	return nil
}

// Upgrade upgrades an HTTP connection to WebSocket.
func (rcm *RedisConnectionManager) Upgrade(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	return rcm.localCM.Upgrade(w, r)
}

// ---- Trip chat pair helpers ----

func tripChatRiderKey(tripID string) string  { return "trip:" + tripID + ":chat:rider" }
func tripChatDriverKey(tripID string) string { return "trip:" + tripID + ":chat:driver" }
func activeRiderKey(driverID string) string  { return "driver:" + driverID + ":active_rider" }
func activeDriverKey(riderID string) string  { return "rider:" + riderID + ":active_driver" }

// SetTripChatPair stores the rider/driver pair for a trip in Redis so that chat
// can be authorised and routed even across gateway restarts.
func (rcm *RedisConnectionManager) SetTripChatPair(tripID, riderID, driverID string, ttl time.Duration) error {
	if tripID == "" || riderID == "" || driverID == "" {
		return fmt.Errorf("tripID, riderID and driverID are required")
	}
	pipe := rcm.rdb.TxPipeline() // pipe is not strictly necessary here but ensures atomicity of the two sets and keeps code cleaner. Not
	pipe.Set(rcm.ctx, tripChatRiderKey(tripID), riderID, ttl)
	pipe.Set(rcm.ctx, tripChatDriverKey(tripID), driverID, ttl)
	_, err := pipe.Exec(rcm.ctx)
	return err
}

// ResolveTripChatPeer returns the peer userID for the sender in a trip chat.
// Returns ErrTripChatPairNotFound when the pair has expired or was never set, and
// ErrTripChatUnauthorized when senderID is not one of the registered participants.
func (rcm *RedisConnectionManager) ResolveTripChatPeer(tripID, senderID string) (string, error) {
	if tripID == "" || senderID == "" {
		return "", fmt.Errorf("tripID and senderID are required")
	}
	values, err := rcm.rdb.MGet(rcm.ctx, tripChatRiderKey(tripID), tripChatDriverKey(tripID)).Result()
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

// ClearTripChatPair removes trip chat pair keys and their derived active rider/driver
// pointers when available. This is used when a trip is cancelled so stale state
// does not keep chat/location paths alive.
func (rcm *RedisConnectionManager) ClearTripChatPair(tripID string) error {
	if tripID == "" {
		return fmt.Errorf("tripID is required")
	}
	values, err := rcm.rdb.MGet(rcm.ctx, tripChatRiderKey(tripID), tripChatDriverKey(tripID)).Result()
	if err != nil {
		return err
	}

	var riderID, driverID string
	if len(values) >= 2 {
		riderID, _ = values[0].(string)
		driverID, _ = values[1].(string)
	}

	pipe := rcm.rdb.TxPipeline()
	pipe.Del(rcm.ctx, tripChatRiderKey(tripID), tripChatDriverKey(tripID))
	if riderID != "" {
		pipe.Del(rcm.ctx, activeDriverKey(riderID))
	}
	if driverID != "" {
		pipe.Del(rcm.ctx, activeRiderKey(driverID))
	}
	_, err = pipe.Exec(rcm.ctx)
	return err
}

// GetActiveDriver returns the current driver mapped to a rider, if present.
// It returns ("", nil) when no mapping exists.
func (rcm *RedisConnectionManager) GetActiveDriver(riderID string) (string, error) {
	if riderID == "" {
		return "", fmt.Errorf("riderID is required")
	}
	driverID, err := rcm.rdb.Get(rcm.ctx, activeDriverKey(riderID)).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return driverID, nil
}

// LeaveUserFromRoom removes all local sockets of userID from roomID.
// This is best-effort cleanup to avoid stale room subscriptions after terminal events.
func (rcm *RedisConnectionManager) LeaveUserFromRoom(userID, roomID string) {
	if userID == "" || roomID == "" {
		return
	}

	rcm.localCM.mu.RLock()
	socketIDs := make([]string, 0, len(rcm.localCM.byUser[userID]))
	for socketID := range rcm.localCM.byUser[userID] {
		socketIDs = append(socketIDs, socketID)
	}
	rcm.localCM.mu.RUnlock()

	for _, socketID := range socketIDs {
		rcm.LeaveRoom(socketID, roomID)
	}
}
