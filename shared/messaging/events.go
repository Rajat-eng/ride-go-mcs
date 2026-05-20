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
	NotifyPaymentSuccessQueue        = "payment_success"
	DeadLetterQueue                  = "dead_letter_queue"
	DriverLocationUpdateQueue        = "driver_location_update" // driver location updates published by api-gateway, consumed by driver-service
	DriverTripAssignedQueue          = "driver_trip_assigned"   // driver service stores driverID→riderID mapping when trip is accepted
	NotifyRiderDriverLocationQueue   = "notify_rider_driver_location" // driver service publishes real-time location to rider via api-gateway
)

type TripEventData struct {
	Trip      *pb.Trip `json:"trip"`
	PickupLat float64  `json:"pickupLat"`
	PickupLng float64  `json:"pickupLng"`
}

type DriverTripResponseData struct {
	Driver      *pbd.Driver `json:"driver"`  // kept for backward compat but may be nil
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
