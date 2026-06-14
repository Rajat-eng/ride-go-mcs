package contracts

import "encoding/json"

// AmqpMessage is the message structure for AMQP.
type AmqpMessage struct {
	OwnerID string          `json:"ownerId"`
	Data    json.RawMessage `json:"data"`
}

// Routing keys - using consistent event/command patterns
const (
	// Trip events (trip.event.*)
	TripEventCreated             = "trip.event.created"
	TripEventDriverAssigned      = "trip.event.driver_assigned"
	TripEventCompleted           = "trip.event.completed"
	TripEventNoDriversFound      = "trip.event.no_drivers_found"
	TripEventDriverNotInterested = "trip.event.driver_not_interested"
	TripEventCancelled           = "trip.event.cancelled"

	// Driver commands (driver.cmd.*)
	DriverCmdTripRequest = "driver.cmd.trip_request"
	DriverCmdTripAccept  = "driver.cmd.trip_accept"
	DriverCmdTripDecline = "driver.cmd.trip_decline"
	DriverCmdLocation    = "driver.cmd.location"
	DriverCmdRegister    = "driver.cmd.register"

	// Payment events (payment.event.*)
	PaymentEventSessionCreated = "payment.event.session_created"
	PaymentEventSuccess        = "payment.event.success"
	PaymentEventFailed         = "payment.event.failed"
	PaymentEventCancelled      = "payment.event.cancelled"

	// Payment commands (payment.cmd.*)
	PaymentCmdCreateSession = "payment.cmd.create_session"

	// Driver events (driver.event.*)
	DriverEventLocation = "driver.event.location" // driver service → rider WS: real-time position of assigned driver

	// Chat commands (chat.cmd.*)
	ChatCmdSend = "chat.cmd.send" // ws-gateway → chat-service: persist + ack

	// Chat events (chat.event.*)
	ChatEventDelivered = "chat.event.delivered" // chat-service → ws-gateway: message stored & delivered
)
