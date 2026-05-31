package messaging

import (
	"errors"
	"log"
	"net/http"
	"sync"

	"ride-sharing/shared/contracts"

	"github.com/gorilla/websocket"
)

var (
	ErrConnectionNotFound = errors.New("connection not found")
)

// connRecord holds a WebSocket connection together with its owning userID.
// The per-record mutex ensures only one goroutine writes to the connection at a time.
type connRecord struct {
	conn   *websocket.Conn
	userID string
	mu     sync.Mutex
}

// ConnectionManager tracks connections by socketID (one unique ID per WebSocket
// connection) and maintains a reverse index from userID to all its socketIDs so
// that a single user on multiple devices receives every message.
type ConnectionManager struct {
	bySocket map[string]*connRecord         // socketID → record
	byUser   map[string]map[string]struct{} // userID  → set of socketIDs
	mu       sync.RWMutex
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		bySocket: make(map[string]*connRecord),
		byUser:   make(map[string]map[string]struct{}),
	}
}

func (cm *ConnectionManager) Upgrade(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	return upgrader.Upgrade(w, r, nil)
}

// Add registers a new WebSocket connection under socketID (unique per connection)
// and links it to the owning userID for multi-device fan-out.
func (cm *ConnectionManager) Add(socketID, userID string, conn *websocket.Conn) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.bySocket[socketID] = &connRecord{conn: conn, userID: userID}
	if _, ok := cm.byUser[userID]; !ok {
		cm.byUser[userID] = make(map[string]struct{})
	}
	cm.byUser[userID][socketID] = struct{}{}
	log.Printf("Added socket %s for user %s", socketID, userID)
}

// Remove removes a socket and returns the owning userID (empty string if not found).
func (cm *ConnectionManager) Remove(socketID string) string {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	rec, ok := cm.bySocket[socketID]
	if !ok {
		return ""
	}
	userID := rec.userID
	delete(cm.bySocket, socketID)
	if sockets, ok := cm.byUser[userID]; ok {
		delete(sockets, socketID)
		if len(sockets) == 0 {
			delete(cm.byUser, userID)
		}
	}
	return userID
}

// HasUser returns true when the user has at least one active socket on this node.
func (cm *ConnectionManager) HasUser(userID string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return len(cm.byUser[userID]) > 0
}

// SendToSocket writes a message to a specific socket connection.
func (cm *ConnectionManager) SendToSocket(socketID string, msg contracts.WSMessage) error {
	cm.mu.RLock()
	rec, ok := cm.bySocket[socketID]
	cm.mu.RUnlock()
	if !ok {
		return ErrConnectionNotFound
	}
	rec.mu.Lock()
	err := rec.conn.WriteJSON(msg)
	rec.mu.Unlock()
	if err != nil {
		// Best-effort stale socket cleanup so reconnecting users are not blocked by dead entries.
		_ = rec.conn.Close()
		cm.Remove(socketID)
	}
	return err
}

// SendMessage delivers a message to every socket owned by userID.
// This preserves the original API while supporting multi-device users.
func (cm *ConnectionManager) SendMessage(userID string, msg contracts.WSMessage) error {
	log.Printf("Sending message to user %s: %+v", userID, msg)
	cm.mu.RLock()
	socketIDs := make([]string, 0, len(cm.byUser[userID]))
	for sid := range cm.byUser[userID] {
		socketIDs = append(socketIDs, sid)
	}
	cm.mu.RUnlock()

	if len(socketIDs) == 0 {
		return ErrConnectionNotFound
	}
	var lastErr error
	for _, sid := range socketIDs {
		if err := cm.SendToSocket(sid, msg); err != nil {
			log.Printf("Error sending to socket %s (user %s): %v", sid, userID, err)
			lastErr = err
		}
	}
	return lastErr
}
