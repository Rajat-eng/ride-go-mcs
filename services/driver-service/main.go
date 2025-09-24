package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"
	"syscall"

	grpcserver "google.golang.org/grpc"
)

const (
	GrpcAddr = ":9092"
)

func main() {
	rabbitMqURI := env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")

	rabbitmq, err := messaging.NewRabbitMQ(rabbitMqURI)
	if err != nil {
		log.Fatal(err)
	}
	defer rabbitmq.Close()

	log.Println("Starting RabbitMQ connection")
	svc := NewService()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		sig := <-sigCh
		log.Printf("Received signal: %v, shutting down...", sig)
		cancel() // call to ctx.done
	}()

	lis, err := net.Listen("tcp", GrpcAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	serverErrors := make(chan error, 1)

	grpcServer := grpcserver.NewServer()
	NewGrpcHandler(grpcServer, svc)

	go func() {
		log.Printf("Starting gRPC server Trip service on port %s", lis.Addr().String())
		if err := grpcServer.Serve(lis); err != nil {
			serverErrors <- err
		}
	}()

	select {
	case err := <-serverErrors:
		log.Fatalf("server error: %v", err)
		cancel() // call to ctx.done--> graceful shutdown
	case <-ctx.Done():
		log.Println("Context cancelled, initiating shutdown")
	}

	// Shutdown gRPC server
	grpcServer.GracefulStop()

	log.Println("Driver Service stopped gracefully")
}
