package main

import (
	"encoding/json"
	"log"
	"net/http"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"
	pb "ride-sharing/shared/proto/driver"
)

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
		messaging.DriverCmdTripRequestQueue,
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

		type driverMessage struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data"`
		}

		var driverMsg driverMessage
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
			// Extract only what the frontend reliably provides (tripID, riderID).
			// Driver identity is taken from the authenticated WS context, not the client payload.
			var frontendData struct {
				TripID  string `json:"tripID"`
				RiderID string `json:"riderID"`
			}
			if err := json.Unmarshal(driverMsg.Data, &frontendData); err != nil {
				log.Printf("Error parsing trip accept payload: %v", err)
				continue
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
		default:
			log.Printf("Unknown message type: %s", driverMsg.Type)
		}
		log.Printf("Received message: %s", message)
	}
}
