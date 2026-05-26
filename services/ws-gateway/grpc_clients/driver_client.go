package grpc_clients

import (
	"log"
	"os"

	pb "ride-sharing/shared/proto/driver"
	"ride-sharing/shared/tracing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type DriverServiceClient struct {
	Client pb.DriverServiceClient
	conn   *grpc.ClientConn
}

func NewDriverServiceClient() (*DriverServiceClient, error) {
	driverServiceURL := os.Getenv("DRIVER_SERVICE_URL")
	if driverServiceURL == "" {
		driverServiceURL = "driver-service:9092"
	}
	dialOptions := append(
		tracing.DialOptionsWithTracing(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	conn, err := grpc.NewClient(driverServiceURL, dialOptions...)
	if err != nil {
		log.Println("failed to connect to driver service:", err)
		return nil, err
	}
	return &DriverServiceClient{
		Client: pb.NewDriverServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *DriverServiceClient) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}
