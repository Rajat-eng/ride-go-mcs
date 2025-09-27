package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"ride-sharing/services/trip-service/internal/infrastructure/events"
	"ride-sharing/services/trip-service/internal/infrastructure/grpc"
	"ride-sharing/services/trip-service/internal/infrastructure/repository"
	"ride-sharing/services/trip-service/internal/service"
	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"
	"syscall"
	"time"

	grpcserver "google.golang.org/grpc"
)

const (
	HttpAddr = ":8083"
	GrpcAddr = ":9093"
)

func main() {
	InMemoryRepository := repository.NewInmemoryRepository()
	TripService := service.NewTripService(InMemoryRepository)

	// rabbit mq connection
	rabbitMqURI := env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")

	rabbitmq, err := messaging.NewRabbitMQ(rabbitMqURI)
	if err != nil {
		log.Fatal(err)
	}
	defer rabbitmq.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		sig := <-sigCh
		log.Printf("Received signal: %v, shutting down...", sig)
		cancel() // call to ctx.done
	}()

	publisher := events.NewTripEventPublisher(rabbitmq)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		// Health check endpoint
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	lis, err := net.Listen("tcp", GrpcAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	serverErrors := make(chan error, 2)

	grpcServer := grpcserver.NewServer()
	grpc.NewGRPCHandler(grpcServer, TripService, publisher) // register grpc handler with grpc server and trip service

	// driver consumer
	driverConsumer := events.NewDriverConsumer(rabbitmq, TripService)
	go func() {
		if err := driverConsumer.Listen(); err != nil {
			log.Fatalf("Failed to listen to the driver consumer message: %v", err)
		}
	}()

	// Start payment consumer
	paymentConsumer := events.NewPaymentConsumer(rabbitmq, TripService)
	go func() {
		if err := paymentConsumer.Listen(); err != nil {
			log.Fatalf("Failed to listen to the payment consumer message: %v", err)
		}
	}()

	httpServer := &http.Server{
		Addr:    HttpAddr,
		Handler: mux,
	}

	go func() {
		log.Printf("Starting gRPC server Trip service on port %s", lis.Addr().String())
		if err := grpcServer.Serve(lis); err != nil {
			serverErrors <- err
		}
	}()

	go func() {
		log.Printf("HTTP server listening on %s", HttpAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server starting error")
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

	// ----- Graceful shutdown -----
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Shutdown HTTP server
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server graceful shutdown error: %v", err)
	}

	// Shutdown gRPC server
	grpcServer.GracefulStop()

	log.Println("Trip Service stopped gracefully")
}
