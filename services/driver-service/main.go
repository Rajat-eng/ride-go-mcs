package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"
	"ride-sharing/shared/tracing"
	"syscall"

	grpcserver "google.golang.org/grpc"
)

const (
	GrpcAddr = ":9092"
)

func main() {

	ctx, cancel := context.WithCancel(context.Background())

	tracerCfg := tracing.Config{
		ServiceName:    "driver-service",
		Environment:    env.GetString("ENVIRONMENT", "development"),
		JaegerEndpoint: env.GetString("JAEGER_ENDPOINT", "http://jaeger:14268/api/traces"),
	}

	sh, err := tracing.InitTracer(tracerCfg)
	if err != nil {
		log.Fatalf("Failed to initialize the tracer: %v", err)
	}

	defer cancel()
	defer sh(ctx)

	rabbitMqURI := env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")
	rabbitmq, err := messaging.NewRabbitMQ(rabbitMqURI)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Starting RabbitMQ connection")
	defer rabbitmq.Close()

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
	svc := NewService()
	// Starting the gRPC server with otel tacing hooks enabled
	grpcServer := grpcserver.NewServer(tracing.WithTracingInterceptors()...)
	NewGrpcHandler(grpcServer, svc)

	go func() {
		log.Printf("Starting gRPC server Trip service on port %s", lis.Addr().String())
		if err := grpcServer.Serve(lis); err != nil {
			serverErrors <- err
		}
	}()

	consumer := NewTripConsumer(rabbitmq, svc)
	go func() {
		if err := consumer.Listen(); err != nil {
			log.Fatalf("Failed to listen to the message: %v", err)
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
