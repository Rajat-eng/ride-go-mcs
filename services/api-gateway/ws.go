package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"
	pb "ride-sharing/shared/proto/driver"
	"time"
)

const (
	WSChatMessageSend     = "chat.message.send"
	WSChatMessageReceived = "chat.message.received"
)

type wsIncomingMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type tripChatMessageData struct {
	TripID    string `json:"tripID"`
	MessageID string `json:"messageID,omitempty"`
	Text      string `json:"text"`
}

type tripChatMessageReceivedData struct {
	TripID    string `json:"tripID"`
	SenderID  string `json:"senderID"`
	MessageID string `json:"messageID,omitempty"`
	Text      string `json:"text"`
	SentAt    int64  `json:"sentAt"`
}

func wsTripTopic(tripID string) string {
	if tripID == "" {
		return ""
	}
	return "trip:" + tripID
}

func relayTripChatMessage(ctx context.Context, connManager *messaging.RedisConnectionManager, senderID string, rawData json.RawMessage) error {
	var payload tripChatMessageData
	if err := json.Unmarshal(rawData, &payload); err != nil {
		return err
	}
	if payload.TripID == "" || payload.Text == "" {
		return nil
	}

	msg := contracts.WSMessage{
		Type:  WSChatMessageReceived,
		Topic: wsTripTopic(payload.TripID),
		Data: tripChatMessageReceivedData{
			TripID:    payload.TripID,
			SenderID:  senderID,
			MessageID: payload.MessageID,
			Text:      payload.Text,
			SentAt:    time.Now().Unix(),
		},
	}

	// Chat is allowed only after the rider/driver pair is registered on accept.
	peerID, err := connManager.ResolveTripChatPeer(payload.TripID, senderID)
	if err != nil {
		return err
	}

	// Echo back to sender so messages reflect immediately in the sender UI.
	if err := connManager.SendMessage(senderID, msg); err != nil {
		log.Printf("Error echoing chat message to sender %s: %v", senderID, err)
	}

	return connManager.SendMessage(peerID, msg)
}

func handleRidersWebSocket(w http.ResponseWriter, r *http.Request, rb *messaging.RabbitMQ, connManager *messaging.RedisConnectionManager, rl *RateLimiter) {
	conn, err := connManager.Upgrade(w, r)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	defer conn.Close()
	userID, _ := r.Context().Value(ctxKeyUserID).(string)

	allowed, release := rl.WsConnectionGate(r.Context(), userID, 3)
	if !allowed {
		conn.Close()
		log.Printf("WS connection rejected for user %s: too many connections", userID)
		return
	}
	defer release()

	// Add connection to manager
	connManager.Add(userID, conn)
	defer connManager.Remove(userID)

	// Initialize queue consumers
	queues := []string{
		messaging.NotifyTripCreatedQueue,
		messaging.NotifyDriverNoDriversFoundQueue,
		messaging.NotifyDriverAssignQueue,
		messaging.NotifyPaymentSessionCreatedQueue,
		messaging.NotifyRiderDriverLocationQueue,
	}

	for _, q := range queues {
		consumer := messaging.NewQueueConsumer(rb, connManager, q)

		if err := consumer.Start(); err != nil {
			log.Printf("Failed to start consumer for queue: %s: err: %v", q, err)
		}
	}
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}

		var riderMsg wsIncomingMessage
		if err := json.Unmarshal(message, &riderMsg); err != nil {
			log.Printf("Error unmarshaling rider message: %v", err)
			continue
		}

		switch riderMsg.Type {
		case WSChatMessageSend:
			if err := relayTripChatMessage(r.Context(), connManager, userID, riderMsg.Data); err != nil {
				log.Printf("Error relaying rider chat message: %v", err)
			}
		case contracts.WSTopicSubscribe:
			var ctrl contracts.WSTopicControlData
			if err := json.Unmarshal(riderMsg.Data, &ctrl); err != nil {
				log.Printf("Error parsing subscribe payload: %v", err)
				continue
			}
			connManager.SubscribeTopic(userID, ctrl.Topic)
		case contracts.WSTopicUnsubscribe:
			var ctrl contracts.WSTopicControlData
			if err := json.Unmarshal(riderMsg.Data, &ctrl); err != nil {
				log.Printf("Error parsing unsubscribe payload: %v", err)
				continue
			}
			connManager.UnsubscribeTopic(userID, ctrl.Topic)
		default:
			log.Printf("Unknown rider message type: %s", riderMsg.Type)
		}

		log.Printf("Received message: %s", message)
	}
}

func handleDriversWebSocket(w http.ResponseWriter, r *http.Request, rb *messaging.RabbitMQ, connManager *messaging.RedisConnectionManager, rl *RateLimiter) {
	conn, err := connManager.Upgrade(w, r)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	defer conn.Close()

	userID, _ := r.Context().Value(ctxKeyUserID).(string)
	driverName, _ := r.Context().Value(ctxKeyName).(string)
	packageSlug := r.URL.Query().Get("packageSlug") // which vehicle package the driver is using
	if packageSlug == "" {
		log.Println("No package slug provided")
		return
	}

	allowed, release := rl.WsConnectionGate(r.Context(), userID, 3)
	if !allowed {
		conn.Close()
		log.Printf("WS connection rejected for user %s: too many connections", userID)
		return
	}
	defer release()

	ctx := r.Context()

	connManager.Add(userID, conn)
	defer func() {
		connManager.Remove(userID)
		// Remove driver from the GEO pool so they stop receiving new trip requests.
		if _, err := driverClient.Client.UnregisterDriver(ctx, &pb.RegisterDriverRequest{DriverID: userID, PackageSlug: packageSlug}); err != nil {
			log.Printf("Failed to unregister driver on disconnect: %v", err)
		}
	}()

	queues := []string{
		messaging.DriverCmdTripRequestQueue, // for receiving new trip requests to drivers
	}

	for _, q := range queues {
		consumer := messaging.NewQueueConsumer(rb, connManager, q)

		if err := consumer.Start(); err != nil {
			log.Printf("Failed to start consumer for queue: %s: err: %v", q, err)
		}
	}

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}

		var driverMsg wsIncomingMessage
		if err := json.Unmarshal(message, &driverMsg); err != nil {
			log.Printf("Error unmarshaling driver message: %v", err)
			continue
		} // converting byte message from ws to go struct driverMessage

		switch driverMsg.Type {
		case contracts.DriverCmdLocation:
			var locMsg struct {
				Location struct {
					Latitude  float64 `json:"latitude"`
					Longitude float64 `json:"longitude"`
				} `json:"location"`
			}
			if err := json.Unmarshal(driverMsg.Data, &locMsg); err != nil {
				log.Printf("Error parsing driver location: %v", err)
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
				log.Printf("Error publishing location update: %v", err)
			}
			continue
		case contracts.DriverCmdTripAccept:
			// when driver clicks accept trip, the driver service sends a message to the trip request service which then notifies the rider of the acceptance outcome (success/failure) through RabbitMQ, which is then forwarded to the rider's frontend through their WebSocket connection.
			var frontendData struct {
				TripID  string `json:"tripID"`
				RiderID string `json:"riderID"`
			}
			if err := json.Unmarshal(driverMsg.Data, &frontendData); err != nil {
				log.Printf("Error parsing trip accept payload: %v", err)
				continue
			}
			// Prime chat pairing immediately so chat works right after accept.
			if frontendData.TripID != "" && frontendData.RiderID != "" {
				if err := connManager.SetTripChatPair(frontendData.TripID, frontendData.RiderID, userID, 2*time.Hour); err != nil {
					log.Printf("Error setting trip chat pair on accept: %v", err)
				}
				// Auto-subscribe driver to the trip topic so they receive scoped events.
				connManager.SubscribeTopic(userID, wsTripTopic(frontendData.TripID))
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
				log.Printf("Error publishing message to RabbitMQ: %v", err)
			}
		case contracts.DriverCmdTripDecline:
			var frontendData struct {
				TripID  string `json:"tripID"`
				RiderID string `json:"riderID"`
			}
			if err := json.Unmarshal(driverMsg.Data, &frontendData); err != nil {
				log.Printf("Error parsing trip decline payload: %v", err)
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
				log.Printf("Error publishing message to RabbitMQ: %v", err)
			}
		case WSChatMessageSend:
			if err := relayTripChatMessage(ctx, connManager, userID, driverMsg.Data); err != nil {
				log.Printf("Error relaying driver chat message: %v", err)
			}
		case contracts.WSTopicSubscribe:
			var ctrl contracts.WSTopicControlData
			if err := json.Unmarshal(driverMsg.Data, &ctrl); err != nil {
				log.Printf("Error parsing subscribe payload: %v", err)
				continue
			}
			connManager.SubscribeTopic(userID, ctrl.Topic)
		case contracts.WSTopicUnsubscribe:
			var ctrl contracts.WSTopicControlData
			if err := json.Unmarshal(driverMsg.Data, &ctrl); err != nil {
				log.Printf("Error parsing unsubscribe payload: %v", err)
				continue
			}
			connManager.UnsubscribeTopic(userID, ctrl.Topic)
		default:
			log.Printf("Unknown message type: %s", driverMsg.Type)
		}
		log.Printf("Received message: %s", message)
	}
}
