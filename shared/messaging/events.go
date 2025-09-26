package messaging

import (
	pbd "ride-sharing/shared/proto/driver"
	pb "ride-sharing/shared/proto/trip"
)

const (
	FindAvailableDriversQueue        = "find_available_drivers"         // when trip is created then driver service consumes event to find available drivers
	DriverCmdTripRequestQueue        = "driver_cmd_trip_request"        // send notification to driver found and ask to accept/reject--> eevent counsumed by api gateway and send to driver usnd web socket
	DriverTripResponseQueue          = "driver_trip_response"           // on getting response from driver the event is publichsed and send to rider
	NotifyDriverNoDriversFoundQueue  = "notify_driver_no_drivers_found" // on no drivers found event is sent to rider using ws and again event is published for
	NotifyDriverAssignQueue          = "notify_driver_assign"           // yes from driver notifeis rider which is read by ws. on no event is published as trip declined--> read by trip service to assign driver again
	PaymentTripResponseQueue         = "payment_trip_response"
	NotifyPaymentSessionCreatedQueue = "notify_payment_session_created"
)

type TripEventData struct {
	Trip *pb.Trip `json:"trip"` // Protobuf message for Trip bcoz tripModel is part of trip-service and cannot be imported here
}

type DriverTripResponseData struct {
	Driver  *pbd.Driver `json:"driver"`
	TripID  string      `json:"tripID"`
	RiderID string      `json:"riderID"`
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
