package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
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

// ValidateEventType validates if event type is supported
func ValidateEventType(eventType string) error {
	validTypes := map[string]bool{
		"payment_failed":   true,
		"payment_success":  true,
		"trip_cancelled":   true,
		"trip_completed":   true,
		"driver_offline":   true,
		"notification_failed": true,
	}

	eventType = strings.ToLower(strings.TrimSpace(eventType))
	if eventType == "" {
		return fmt.Errorf("event type cannot be empty")
	}

	if !validTypes[eventType] {
		return fmt.Errorf("unsupported event type: %s", eventType)
	}

	return nil
}

// ValidateMaxRetries validates retry count is within acceptable range
func ValidateMaxRetries(retries int) error {
	if retries < 0 {
		return fmt.Errorf("retries cannot be negative")
	}
	if retries > 100 {
		return fmt.Errorf("retries exceed maximum allowed (100)")
	}
	return nil
}

// ValidateMessageID validates message ID is not empty
func ValidateMessageID(msgID string) error {
	if msgID == "" {
		return fmt.Errorf("message ID cannot be empty")
	}
	return nil
}

// IsRetryableError determines if an error is retryable
func IsRetryableError(errMsg string) bool {
	retryablePatterns := []string{
		"timeout",
		"unavailable",
		"connection refused",
		"temporarily unavailable",
		"i/o timeout",
	}

	msg := strings.ToLower(errMsg)
	for _, pattern := range retryablePatterns {
		if strings.Contains(msg, pattern) {
			return true
		}
	}

	return false
}
