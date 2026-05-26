package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"ride-sharing/services/chat-service/internal/infrastructure/events"
	"ride-sharing/services/chat-service/internal/infrastructure/repository"
	"ride-sharing/services/chat-service/internal/service"
	"ride-sharing/shared/db"
	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"
	"ride-sharing/shared/tracing"
)

func main() {
	log.Println("Starting chat-service")

	tracerCfg := tracing.Config{
		ServiceName:    "chat-service",
		Environment:    env.GetString("ENVIRONMENT", "development"),
		JaegerEndpoint: env.GetString("JAEGER_ENDPOINT", "http://jaeger:14268/api/traces"),
	}
	sh, err := tracing.InitTracer(tracerCfg)
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer sh(ctx)

	// MongoDB
	mongoURI := env.GetString("MONGO_URI", "mongodb://mongodb:27017")
	mongoDB := env.GetString("MONGO_DB", "chat")
	mongoCfg := &db.MongoConfig{URI: mongoURI, Database: mongoDB}
	mongoClient, err := db.NewMongoClient(ctx, mongoCfg)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoClient.Disconnect(ctx)
	log.Println("Connected to MongoDB")

	database := mongoClient.Database(mongoDB)

	// RabbitMQ
	rabbitMqURI := env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")
	rabbitmq, err := messaging.NewRabbitMQ(rabbitMqURI)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rabbitmq.Close()
	log.Println("Connected to RabbitMQ")

	// Wire up layers.
	repo := repository.NewMongoMessageRepository(database)
	publisher := events.NewPublisher(rabbitmq)
	chatSvc := service.New(repo, publisher)
	consumer := events.NewConsumer(rabbitmq, chatSvc)

	if err := consumer.Start(ctx); err != nil {
		log.Fatalf("Failed to start chat consumer: %v", err)
	}

	log.Println("chat-service running")

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	<-shutdown
	log.Println("chat-service shutting down")
}
