package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"
	"ride-sharing/shared/tracing"
	"syscall"

	"github.com/redis/go-redis/v9"
	grpcserver "google.golang.org/grpc"
)

const (
	GrpcAddr = ":9092"
	HTTPAddr = ":8082"
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

	redisAddr := env.GetString("REDIS_URI", "redis:6379")
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer rdb.Close()
	log.Println("Connected to Redis")

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		sig := <-sigCh
		log.Printf("Received signal: %v, shutting down...", sig)
		cancel()
	}()

	lis, err := net.Listen("tcp", GrpcAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	serverErrors := make(chan error, 1)
	svc := NewService(rdb)
	// Starting the gRPC server with otel tracing hooks enabled
	grpcServer := grpcserver.NewServer(tracing.WithTracingInterceptors()...)
	NewGrpcHandler(grpcServer, svc)

	go func() {
		log.Printf("Starting gRPC server Driver service on port %s", lis.Addr().String())
		if err := grpcServer.Serve(lis); err != nil {
			serverErrors <- err
		}
	}()

	// HTTP server for dev tooling (seed endpoint)
	mux := http.NewServeMux()
	mux.HandleFunc("/seed", HandleSeedDrivers(svc))
	go func() {
		log.Printf("Starting HTTP server on %s", HTTPAddr)
		if err := http.ListenAndServe(HTTPAddr, mux); err != nil {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	consumer := NewTripConsumer(rabbitmq, svc)
	go func() {
		if err := consumer.Listen(); err != nil {
			log.Fatalf("Failed to listen to the message: %v", err)
		}
	}()

	locConsumer := NewLocationConsumer(rabbitmq, svc)
	go func() {
		if err := locConsumer.Listen(); err != nil {
			log.Fatalf("Failed to listen to location updates: %v", err)
		}
	}()

	tripAssignedConsumer := NewTripAssignedConsumer(rabbitmq, svc)
	go func() {
		if err := tripAssignedConsumer.Listen(); err != nil {
			log.Fatalf("Failed to listen to trip assignments: %v", err)
		}
	}()

	select {
	case err := <-serverErrors:
		log.Fatalf("server error: %v", err)
		cancel()
	case <-ctx.Done():
		log.Println("Context cancelled, initiating shutdown")
	}

	grpcServer.GracefulStop()

	log.Println("Driver Service stopped gracefully")
}
