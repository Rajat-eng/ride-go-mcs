package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"
	"ride-sharing/shared/tracing"
)

var (
	httpAddr    = env.GetString("HTTP_ADDR", ":8081")
	rabbitMqURI = env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")
)

func main() {
	log.Println("Starting API Gateway")

	// Initialize Tracing
	tracerCfg := tracing.Config{
		ServiceName:    "api-gateway",
		Environment:    env.GetString("ENVIRONMENT", "development"),
		JaegerEndpoint: env.GetString("JAEGER_ENDPOINT", "http://jaeger:14268/api/traces"),
	}

	sh, err := tracing.InitTracer(tracerCfg)
	if err != nil {
		log.Fatalf("Failed to initialize the tracer: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer sh(ctx)

	// RabbitMQ connection
	rabbitmq, err := messaging.NewRabbitMQ(rabbitMqURI)
	if err != nil {
		log.Fatal(err)
	}
	defer rabbitmq.Close()

	log.Println("Starting RabbitMQ connection")

	mux := http.NewServeMux() // create a new ServeMux for routing

	// Define a simple health check endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// tracing middleware adds extra headers, automatatic tracing , methods
	// tracer.start becomes child span(cusotm span) for this middlware

	// Route for trip preview
	mux.Handle("POST /trip/preview", tracing.WrapHandlerFunc(enableCORS(HandleTripPreview), "/trip/preview"))
	// mux.HandleFunc("POST /trip/start", enableCORS(HandleStartTrip))
	mux.Handle("POST /trip/start", tracing.WrapHandlerFunc(enableCORS(HandleStartTrip), "/trip/start"))

	mux.Handle("/ws/drivers", tracing.WrapHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleDriversWebSocket(w, r, rabbitmq)
	}, "/ws/drivers"))
	mux.Handle("/ws/riders", tracing.WrapHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleRidersWebSocket(w, r, rabbitmq)
	}, "/ws/riders"))
	mux.Handle("/webhook/stripe", tracing.WrapHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleStripeWebhook(w, r, rabbitmq)
	}, "/webhook/stripe"))

	server := &http.Server{
		Addr:    httpAddr,
		Handler: mux,
	}

	// graceful shutdown
	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("HTTP server listening on %s", server.Addr)
		serverErrors <- server.ListenAndServe()
	}()
	// goroutine to start the server.This allows the main goroutine to continue executing and listen for shutdown signals

	shutdown := make(chan os.Signal, 1)                    // buffered channel to receive shutdown signals
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM) // relay os.Interrupt,SIGTERM to shutdown channel

	select {
	case err := <-serverErrors:
		log.Fatalf("could not start server: %v", err)
	case sig := <-shutdown:
		log.Printf("starting shutdown due to signal: %v", sig)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		// wait for 10 seconds to finish ongoing requests before shutting down
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("graceful shutdown did not complete in 10s : %v", err)
			server.Close() // force close
		}
	}
}
