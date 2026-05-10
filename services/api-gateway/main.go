package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ride-sharing/services/api-gateway/grpc_clients"
	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"
	"ride-sharing/shared/tracing"

	"strings"

	"github.com/redis/go-redis/v9"
	"github.com/rs/cors"
)

var (
	httpAddr    = env.GetString("HTTP_ADDR", ":8081")
	rabbitMqURI = env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")
	redisUri    = env.GetString("REDIS_URI", "redis:6379")
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
	defer sh(ctx) // Ensure tracer is shutdown when main exits

	// RabbitMQ connection
	rabbitmq, err := messaging.NewRabbitMQ(rabbitMqURI)
	if err != nil {
		log.Fatal(err)
	}
	defer rabbitmq.Close()

	log.Println("Starting RabbitMQ connection")

	// 🔗 Connect to Redis running in Kubernetes
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisUri,
		Password: "",
		DB:       0,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("❌ Failed to connect to Redis: %v", err)
	}
	log.Println("✅ Connected to Redis")

	connManager := messaging.NewRedisConnectionManager(rdb)

	// Initialize shared gRPC clients once at startup.
	tripClient, err = grpc_clients.NewTripServiceClient()
	if err != nil {
		log.Fatalf("failed to create trip service client: %v", err)
	}
	defer tripClient.Close()

	loginClient, err = grpc_clients.NewLoginServiceClient()
	if err != nil {
		log.Fatalf("failed to create login service client: %v", err)
	}
	defer loginClient.Close()

	driverClient, err = grpc_clients.NewDriverServiceClient()
	if err != nil {
		log.Fatalf("failed to create driver service client: %v", err)
	}
	defer driverClient.Close()

	mux := http.NewServeMux() // create a new ServeMux for routing

	// Define a simple health check endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// tracing middleware adds extra headers, automatatic tracing , methods
	// tracer.start becomes child span(cusotm span) for this middlware

	// Route for trip preview
	mux.Handle("POST /trip/preview", wsAuthMiddleware(tracing.WrapHandlerFunc(HandleTripPreview, "/trip/preview")))
	mux.Handle("POST /trip/start", wsAuthMiddleware(tracing.WrapHandlerFunc(HandleStartTrip, "/trip/start")))

	// Auth routes
	mux.Handle("POST /auth/signup", tracing.WrapHandlerFunc(HandleSignup, "/auth/signup"))
	mux.Handle("POST /auth/login", tracing.WrapHandlerFunc(HandleLogin, "/auth/login"))
	mux.Handle("POST /auth/google", tracing.WrapHandlerFunc(HandleGoogleAuth, "/auth/google"))
	mux.Handle("POST /auth/refresh", tracing.WrapHandlerFunc(HandleRefreshToken, "/auth/refresh"))
	mux.Handle("POST /auth/logout", tracing.WrapHandlerFunc(HandleLogout, "/auth/logout"))

	mux.Handle("/ws/drivers", wsAuthMiddleware(tracing.WrapHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleDriversWebSocket(w, r, rabbitmq, connManager)
	}, "/ws/drivers")))
	mux.Handle("/ws/riders", wsAuthMiddleware(tracing.WrapHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleRidersWebSocket(w, r, rabbitmq, connManager)
	}, "/ws/riders")))
	mux.Handle("/webhook/stripe", wsAuthMiddleware(tracing.WrapHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleStripeWebhook(w, r, rabbitmq)
	}, "/webhook/stripe")))

	allowedOrigins := strings.Split(env.GetString("ALLOWED_ORIGINS", "http://localhost:3000"), ",")
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	server := &http.Server{
		Addr:    httpAddr,
		Handler: corsHandler.Handler(mux),
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
