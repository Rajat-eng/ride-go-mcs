package main

import (
	"context"
	"log"
	"os"
	"time"

	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"
)

var (
	rabbitMqURI = env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")
)

func main() {
	rmq, err := messaging.NewRabbitMQ(rabbitMqURI)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rmq.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	log.Println("🚀 Starting DLQ drain job...")

	if err := rmq.StartDLQConsumer(ctx); err != nil {
		log.Printf("⚠️ DLQ drain failed: %v", err)
		os.Exit(1)
	}

	log.Println("✅ DLQ drain completed successfully. Exiting.")
}
