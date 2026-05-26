package main

import "ride-sharing/services/api-gateway/grpc_clients"

// Shared, long-lived gRPC clients. gRPC uses HTTP/2 multiplexing so a single
// connection per downstream service handles all concurrent requests safely.
var (
	tripClient  *grpc_clients.TripServiceClient
	loginClient *grpc_clients.LoginServiceClient
)
