package grpc_clients

import (
	"log"
	"os"
	pb "ride-sharing/shared/proto/login"
	"ride-sharing/shared/tracing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type LoginServiceClient struct {
	Client pb.LoginServiceClient // The actual gRPC client to call login service methods like Signup, Login, RefreshToken
	conn   *grpc.ClientConn      // Keep the connection to close it gracefully on shutdown
}

func NewLoginServiceClient() (*LoginServiceClient, error) {
	loginServiceURL := os.Getenv("LOGIN_SERVICE_URL")
	if loginServiceURL == "" {
		loginServiceURL = "login-service:9095"
	}

	dialOptions := append(
		tracing.DialOptionsWithTracing(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	conn, err := grpc.NewClient(loginServiceURL, dialOptions...)
	if err != nil {
		log.Println("failed to connect to login service:", err)
		return nil, err
	}

	client := pb.NewLoginServiceClient(conn) // Wrap the client with tracing interceptor
	return &LoginServiceClient{Client: client, conn: conn}, nil
}

func (c *LoginServiceClient) Close() {
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			log.Println("failed to close login service connection:", err)
			return
		}
	}
}
