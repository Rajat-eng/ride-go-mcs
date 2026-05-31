package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"
	pb "ride-sharing/shared/proto/driver"

	"github.com/google/uuid"
)

const wsPongWait = 45 * time.Second

func startWsGateHeartbeat(userID string, rl *RateLimiter) func() {
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				rl.RefreshWsConnectionGate(context.Background(), userID)
			case <-done:
				return
			}
		}
	}()
	return func() { close(done) }
}

func handleRidersWebSocket(
	w http.ResponseWriter,
	r *http.Request,
	rb *messaging.RabbitMQ,
	connManager *messaging.RedisConnectionManager,
	rl *RateLimiter,
) {
	conn, err := connManager.Upgrade(w, r)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	userID, _ := r.Context().Value(ctxKeyUserID).(string)
	socketID := uuid.New().String()

	allowed, release := rl.WsConnectionGate(r.Context(), userID, 3)
	if !allowed {
		conn.Close()
		log.Printf("WS connection rejected for rider %s: too many connections", userID)
		return
	}
	defer release()
	stopHeartbeat := startWsGateHeartbeat(userID, rl)
	defer stopHeartbeat()

	connManager.Add(userID, socketID, conn)
	defer connManager.Remove(socketID)

	_ = conn.SetReadDeadline(time.Now().Add(wsPongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(wsPongWait))
	})

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Rider WS read error: %v", err)
			break
		}

		var msg wsIncomingMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Rider message unmarshal error: %v", err)
			continue
		}

		switch msg.Type {
		case WSChatMessageSend:
			if err := relayTripChatMessage(r.Context(), connManager, rb, socketID, userID, msg.Data); err != nil {
				log.Printf("Rider chat relay error: %v", err)
			}

		case contracts.WSRoomJoin:
			var ctrl contracts.WSRoomControlData
			if err := json.Unmarshal(msg.Data, &ctrl); err != nil {
				log.Printf("Rider room join parse error: %v", err)
				continue
			}
			connManager.JoinRoom(socketID, ctrl.RoomID)

		case contracts.WSRoomLeave:
			var ctrl contracts.WSRoomControlData
			if err := json.Unmarshal(msg.Data, &ctrl); err != nil {
				log.Printf("Rider room leave parse error: %v", err)
				continue
			}
			connManager.LeaveRoom(socketID, ctrl.RoomID)

		// Legacy topic control — map to room model for backward compat.
		case contracts.WSTopicSubscribe:
			var ctrl contracts.WSTopicControlData
			if err := json.Unmarshal(msg.Data, &ctrl); err != nil {
				log.Printf("Rider subscribe parse error: %v", err)
				continue
			}
			connManager.JoinRoom(socketID, ctrl.Topic)

		case contracts.WSTopicUnsubscribe:
			var ctrl contracts.WSTopicControlData
			if err := json.Unmarshal(msg.Data, &ctrl); err != nil {
				log.Printf("Rider unsubscribe parse error: %v", err)
				continue
			}
			connManager.LeaveRoom(socketID, ctrl.Topic)

		case contracts.DriverCmdLocation:
			// Riders cannot send location updates — silently discard.
			continue

		default:
			log.Printf("Unknown rider message type: %s", msg.Type)
		}
	}
}

func handleDriversWebSocket(
	w http.ResponseWriter,
	r *http.Request,
	rb *messaging.RabbitMQ,
	connManager *messaging.RedisConnectionManager,
	rl *RateLimiter,
) {
	conn, err := connManager.Upgrade(w, r)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	userID, _ := r.Context().Value(ctxKeyUserID).(string)
	driverName, _ := r.Context().Value(ctxKeyName).(string)
	packageSlug := r.URL.Query().Get("packageSlug")
	if packageSlug == "" {
		log.Println("No package slug provided — closing driver WS")
		return
	}

	socketID := uuid.New().String()

	allowed, release := rl.WsConnectionGate(r.Context(), userID, 3)
	if !allowed {
		conn.Close()
		log.Printf("WS connection rejected for driver %s: too many connections", userID)
		return
	}
	defer release()
	stopHeartbeat := startWsGateHeartbeat(userID, rl)
	defer stopHeartbeat()

	ctx := r.Context()

	connManager.Add(userID, socketID, conn)

	_ = conn.SetReadDeadline(time.Now().Add(wsPongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(wsPongWait))
	})

	defer func() {
		connManager.Remove(socketID)
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		// on disconnect, attempt to unregister driver to clean up geo state.
		// timeout will trigger cleanup on the driver service side even if this call fails, so we won't leak drivers indefinitely.
		defer cancel()
		if _, err := driverClient.Client.UnregisterDriver(cleanupCtx, &pb.RegisterDriverRequest{
			DriverID:    userID,
			PackageSlug: packageSlug,
		}); err != nil {
			log.Printf("Failed to unregister driver on disconnect: %v", err)
		}
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Driver WS read error: %v", err)
			break
		}

		var msg wsIncomingMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Driver message unmarshal error: %v", err)
			continue
		}

		switch msg.Type {
		case contracts.DriverCmdLocation:
			var locMsg struct {
				Location struct {
					Latitude  float64 `json:"latitude"`
					Longitude float64 `json:"longitude"`
				} `json:"location"`
			}
			if err := json.Unmarshal(msg.Data, &locMsg); err != nil {
				log.Printf("Driver location parse error: %v", err)
				continue
			}
			locPayload, _ := json.Marshal(messaging.DriverLocationUpdateData{
				PackageSlug: packageSlug,
				Latitude:    locMsg.Location.Latitude,
				Longitude:   locMsg.Location.Longitude,
			})
			if err := rb.PublishMessage(ctx, contracts.DriverCmdLocation, contracts.AmqpMessage{
				OwnerID: userID,
				Data:    locPayload,
			}); err != nil {
				log.Printf("Error publishing driver location: %v", err)
			}

		case contracts.DriverCmdTripAccept:
			var frontendData struct {
				TripID  string `json:"tripID"`
				RiderID string `json:"riderID"`
			}
			if err := json.Unmarshal(msg.Data, &frontendData); err != nil {
				log.Printf("Driver trip accept parse error: %v", err)
				continue
			}
			if frontendData.TripID != "" && frontendData.RiderID != "" {
				// Register trip chat pair so chat is authorised from this moment.
				if err := connManager.SetTripChatPair(frontendData.TripID, frontendData.RiderID, userID, 2*time.Hour); err != nil {
					log.Printf("Error setting trip chat pair: %v", err)
				}
				// Driver auto-joins the trip chat room.
				connManager.JoinRoom(socketID, tripChatRoomID(frontendData.TripID))
			}
			enrichedData, _ := json.Marshal(messaging.DriverTripResponseData{
				TripID:      frontendData.TripID,
				RiderID:     frontendData.RiderID,
				DriverID:    userID,
				DriverName:  driverName,
				PackageSlug: packageSlug,
			})
			if err := rb.PublishMessage(ctx, contracts.DriverCmdTripAccept, contracts.AmqpMessage{
				OwnerID: userID,
				Data:    enrichedData,
			}); err != nil {
				log.Printf("Error publishing trip accept: %v", err)
			}

		case contracts.DriverCmdTripDecline:
			var frontendData struct {
				TripID  string `json:"tripID"`
				RiderID string `json:"riderID"`
			}
			if err := json.Unmarshal(msg.Data, &frontendData); err != nil {
				log.Printf("Driver trip decline parse error: %v", err)
				continue
			}
			enrichedData, _ := json.Marshal(messaging.DriverTripResponseData{
				TripID:      frontendData.TripID,
				RiderID:     frontendData.RiderID,
				DriverID:    userID,
				DriverName:  driverName,
				PackageSlug: packageSlug,
			})
			if err := rb.PublishMessage(ctx, contracts.DriverCmdTripDecline, contracts.AmqpMessage{
				OwnerID: userID,
				Data:    enrichedData,
			}); err != nil {
				log.Printf("Error publishing trip decline: %v", err)
			}

		case WSChatMessageSend:
			if err := relayTripChatMessage(ctx, connManager, rb, socketID, userID, msg.Data); err != nil {
				log.Printf("Driver chat relay error: %v", err)
			}

		case contracts.WSRoomJoin:
			var ctrl contracts.WSRoomControlData
			if err := json.Unmarshal(msg.Data, &ctrl); err != nil {
				log.Printf("Driver room join parse error: %v", err)
				continue
			}
			connManager.JoinRoom(socketID, ctrl.RoomID)

		case contracts.WSRoomLeave:
			var ctrl contracts.WSRoomControlData
			if err := json.Unmarshal(msg.Data, &ctrl); err != nil {
				log.Printf("Driver room leave parse error: %v", err)
				continue
			}
			connManager.LeaveRoom(socketID, ctrl.RoomID)

		// Legacy topic control — map to room model for backward compat.
		case contracts.WSTopicSubscribe:
			var ctrl contracts.WSTopicControlData
			if err := json.Unmarshal(msg.Data, &ctrl); err != nil {
				log.Printf("Driver subscribe parse error: %v", err)
				continue
			}
			connManager.JoinRoom(socketID, ctrl.Topic)

		case contracts.WSTopicUnsubscribe:
			var ctrl contracts.WSTopicControlData
			if err := json.Unmarshal(msg.Data, &ctrl); err != nil {
				log.Printf("Driver unsubscribe parse error: %v", err)
				continue
			}
			connManager.LeaveRoom(socketID, ctrl.Topic)

		default:
			log.Printf("Unknown driver message type: %s", msg.Type)
		}
	}
}
