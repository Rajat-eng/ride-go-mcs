package messaging

import (
	pbd "ride-sharing/shared/proto/driver"
	pb "ride-sharing/shared/proto/trip"
)

const (
	FindAvailableDriversQueue        = "find_available_drivers"
	NotifyTripCreatedQueue           = "notify_trip_created"
	DriverCmdTripRequestQueue        = "driver_cmd_trip_request"
	DriverTripResponseQueue          = "driver_trip_response"
	NotifyDriverNoDriversFoundQueue  = "notify_driver_no_drivers_found"
	NotifyDriverAssignQueue          = "notify_driver_assign"
	NotifyTripCompletedQueue         = "notify_trip_completed"
	PaymentTripResponseQueue         = "payment_trip_response"
	NotifyPaymentSessionCreatedQueue = "notify_payment_session_created"
	NotifyPaymentSuccessQueue        = "payment_success"
	DeadLetterQueue                  = "dead_letter_queue"
	DriverLocationUpdateQueue        = "driver_location_update"
	DriverTripAssignedQueue          = "driver_trip_assigned"
	NotifyRiderDriverLocationQueue   = "notify_rider_driver_location"

	// Chat queues — ws-gateway publishes, chat-service consumes (and vice-versa for acks).
	ChatCmdSendQueue        = "chat_cmd_send"
	ChatEventDeliveredQueue = "chat_event_delivered"

	// Cancel queue — trip-service publishes, ws-gateway cancels both rider and driver.
	NotifyTripCancelledQueue = "notify_trip_cancelled"
)

type TripEventData struct {
	Trip      *pb.Trip `json:"trip"`
	PickupLat float64  `json:"pickupLat"`
	PickupLng float64  `json:"pickupLng"`
}

type DriverTripResponseData struct {
	Driver      *pbd.Driver `json:"driver"` // kept for backward compat but may be nil
	TripID      string      `json:"tripID"`
	RiderID     string      `json:"riderID"`
	DriverID    string      `json:"driverID"`
	DriverName  string      `json:"driverName"`
	PackageSlug string      `json:"packageSlug"`
}

type PaymentEventSessionCreatedData struct {
	TripID    string  `json:"tripID"`
	SessionID string  `json:"sessionID"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
}

type PaymentTripResponseData struct {
	TripID   string  `json:"tripID"`
	UserID   string  `json:"userID"`
	DriverID string  `json:"driverID"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type PaymentStatusUpdateData struct {
	TripID   string `json:"tripID"`
	UserID   string `json:"userID"`
	DriverID string `json:"driverID"`
}

type DriverLocationUpdateData struct {
	PackageSlug string  `json:"packageSlug"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}

type DriverLocationEventData struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// TripCancelledData is the payload published on trip.event.cancelled.
// It carries both riderID and driverID so the ws-gateway cancel consumer can
// fan-out to both parties and clean up Redis state in one shot.
type TripCancelledData struct {
	TripID         string `json:"tripID"`
	RiderID        string `json:"riderID"`
	DriverID       string `json:"driverID"`
	DriverAccepted bool   `json:"driverAccepted"`
}

// ChatMessageData is the payload published to ChatCmdSendQueue by ws-gateway
// and consumed by chat-service for persistence and delivery acknowledgement.
type ChatMessageData struct {
	MessageID string `json:"messageID"`
	TripID    string `json:"tripID"`
	SenderID  string `json:"senderID"`
	Text      string `json:"text"`
	SentAt    int64  `json:"sentAt"`
}

// ChatDeliveredData is the payload published by chat-service once a message
// has been persisted, allowing ws-gateway to emit a delivery receipt.
type ChatDeliveredData struct {
	MessageID string `json:"messageID"`
	TripID    string `json:"tripID"`
}
