package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ride-sharing/services/ws-gateway/grpc_clients"
	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"
	"ride-sharing/shared/tracing"

	"strings"

	"github.com/redis/go-redis/v9"
	"github.com/rs/cors"
)

var (
	httpAddr    = env.GetString("HTTP_ADDR", ":8082")
	rabbitMqURI = env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")
	redisURI    = env.GetString("REDIS_URI", "redis:6379")
)

func main() {
	log.Println("Starting WS Gateway")

	tracerCfg := tracing.Config{
		ServiceName:    "ws-gateway",
		Environment:    env.GetString("ENVIRONMENT", "development"),
		JaegerEndpoint: env.GetString("JAEGER_ENDPOINT", "http://jaeger:14268/api/traces"),
		OTLPEndpoint:   env.GetString("OTEL_EXPORTER_OTLP_ENDPOINT", "otel-collector:4317"),
	}
	sh, err := tracing.InitTracer(tracerCfg)
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}
	msh, err := tracing.InitMeter(tracerCfg)
	if err != nil {
		log.Fatalf("Failed to initialize meter provider: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer sh(ctx)
	defer msh(ctx)

	rabbitmq, err := messaging.NewRabbitMQ(rabbitMqURI)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rabbitmq.Close()
	log.Println("Connected to RabbitMQ")

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisURI,
		Password: "",
		DB:       0,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("Connected to Redis")

	connManager := messaging.NewRedisConnectionManager(rdb)
	rateLimiter := NewRateLimiter(rdb)

	// Start one global consumer per queue. Each consumer reads from RabbitMQ and
	// routes by ownerID via connManager.SendMessage — no per-connection consumers needed.
	globalQueues := []string{
		messaging.NotifyTripCreatedQueue,
		messaging.NotifyDriverNoDriversFoundQueue,
		messaging.NotifyDriverAssignQueue,
		messaging.NotifyPaymentSessionCreatedQueue,
		messaging.NotifyRiderDriverLocationQueue,
		messaging.ChatEventDeliveredQueue,
		messaging.DriverCmdTripRequestQueue,
	}
	for _, q := range globalQueues {
		c := messaging.NewQueueConsumer(rabbitmq, connManager, q)
		if err := c.Start(); err != nil {
			log.Fatalf("Failed to start global consumer for queue %s: %v", q, err)
		}
		log.Printf("Global consumer started for queue: %s", q)
	}

	var clientErr error
	driverClient, clientErr = grpc_clients.NewDriverServiceClient()
	if clientErr != nil {
		log.Fatalf("Failed to create driver service client: %v", clientErr)
	}
	defer driverClient.Close()

	// Start the dedicated cancel consumer — handles Redis cleanup + bi-directional WS notification.
	cc := newCancelConsumer(rabbitmq, connManager)
	if err := cc.Start(); err != nil {
		log.Fatalf("Failed to start cancel consumer: %v", err)
	}
	log.Println("Cancel consumer started")

	psc := newPaymentSuccessConsumer(rabbitmq, connManager)
	if err := psc.Start(); err != nil {
		log.Fatalf("Failed to start payment success consumer: %v", err)
	}
	log.Println("Payment success consumer started")

	mux := http.NewServeMux()

	mux.Handle("/", tracing.WrapHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ws-gateway OK"))
	}, "/"))

	mux.Handle("/ws/riders", tracing.WrapHandler(
		wsAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handleRidersWebSocket(w, r, rabbitmq, connManager, rateLimiter)
		})),
		"/ws/riders",
	))

	mux.Handle("/ws/drivers", tracing.WrapHandler(
		wsAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handleDriversWebSocket(w, r, rabbitmq, connManager, rateLimiter)
		})),
		"/ws/drivers",
	))

	allowedOrigins := strings.Split(env.GetString("ALLOWED_ORIGINS", "http://localhost:3000"), ",")
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	server := &http.Server{
		Addr:    httpAddr,
		Handler: corsHandler.Handler(mux),
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("WS Gateway listening on %s", server.Addr)
		serverErrors <- server.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Fatalf("WS Gateway failed to start: %v", err)
	case sig := <-shutdown:
		log.Printf("Shutting down WS Gateway due to signal: %v", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Graceful shutdown incomplete: %v", err)
			server.Close()
		}
	}
}
