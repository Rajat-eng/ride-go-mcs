package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"ride-sharing/services/payment-service/internal/events"
	redisstore "ride-sharing/services/payment-service/internal/infrastructure/redis"
	"ride-sharing/services/payment-service/internal/infrastructure/stripe"
	"ride-sharing/services/payment-service/internal/service"
	"ride-sharing/services/payment-service/pkg/types"
	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"
	"ride-sharing/shared/tracing"

	"github.com/redis/go-redis/v9"
)

var GrpcAddr = env.GetString("GRPC_ADDR", ":9004")

func main() {

	ctx, cancel := context.WithCancel(context.Background())

	// Initialize Tracing
	tracerCfg := tracing.Config{
		ServiceName:    "payment-service",
		Environment:    env.GetString("ENVIRONMENT", "development"),
		JaegerEndpoint: env.GetString("JAEGER_ENDPOINT", "http://jaeger:14268/api/traces"),
		OTLPEndpoint:   env.GetString("OTEL_EXPORTER_OTLP_ENDPOINT", "otel-collector:4317"),
	}

	sh, err := tracing.InitTracer(tracerCfg)
	if err != nil {
		log.Fatalf("Failed to initialize the tracer: %v", err)
	}

	msh, err := tracing.InitMeter(tracerCfg)
	if err != nil {
		log.Fatalf("Failed to initialize meter provider: %v", err)
	}
	defer cancel()
	defer sh(ctx)
	defer msh(ctx)
	rabbitMqURI := env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")

	// Setup graceful shutdown

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		cancel()
	}()

	appURL := env.GetString("APP_URL", "http://localhost:3000")

	// Stripe config
	stripeCfg := &types.PaymentConfig{
		StripeSecretKey: env.GetString("STRIPE_SECRET_KEY", ""),
		SuccessURL:      env.GetString("STRIPE_SUCCESS_URL", appURL+"?payment=success"),
		CancelURL:       env.GetString("STRIPE_CANCEL_URL", appURL+"?payment=cancel"),
	}

	paymentProcessor := stripe.NewStripeClient(stripeCfg)

	redisAddr := env.GetString("REDIS_URI", "redis:6379")
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer rdb.Close()
	log.Println("Connected to Redis")

	sessionStore := redisstore.NewSessionStore(rdb)
	svc := service.NewPaymentService(paymentProcessor, sessionStore)

	log.Println(svc)

	if stripeCfg.StripeSecretKey == "" {
		log.Fatalf("STRIPE_SECRET_KEY is not set")
		return
	}

	// RabbitMQ connection
	rabbitmq, err := messaging.NewRabbitMQ(rabbitMqURI)
	if err != nil {
		log.Fatal(err)
	}
	defer rabbitmq.Close()

	tripConsumer := events.NewTripConsumer(rabbitmq, svc)
	go func() {
		if err := tripConsumer.Listen(); err != nil {
			log.Fatalf("Failed to consume message for payment: %v", err)
		}
	}()

	log.Println("Starting RabbitMQ connection")

	// Wait for shutdown signal
	<-ctx.Done()
	log.Println("Shutting down payment service...")
}
