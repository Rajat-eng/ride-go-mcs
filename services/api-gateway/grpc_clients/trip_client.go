package grpc_clients

import (
	"log"
	"os"
	pb "ride-sharing/shared/proto/trip"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type TripServiceClient struct {
	Client pb.TripServiceClient // this is grpc client which creates connection to trip service
	conn   *grpc.ClientConn     // connection object to trip service
}

func NewTripServiceClient() (*TripServiceClient, error) {
	tripServiceURL := os.Getenv("TRIP_SERVICE_URL")
	if tripServiceURL == "" {
		tripServiceURL = "trip-service:9093"
	}
	conn, err := grpc.NewClient(tripServiceURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	// conn is connection object to trip  via grpc
	if err != nil {
		log.Println("failed to connect to trip service:", err)
		return nil, err
	}

	client := pb.NewTripServiceClient(conn) // this gives trip service client object to call trip service methods like CreateTrip, PreviewTrip
	return &TripServiceClient{Client: client, conn: conn}, nil

}

func (c *TripServiceClient) Close() {
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			return
		}
	}
}
