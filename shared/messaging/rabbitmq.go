package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"ride-sharing/shared/contracts"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	TripExchange = "trip"
)

type RabbitMQ struct {
	conn    *amqp.Connection
	Channel *amqp.Channel
}

func NewRabbitMQ(uri string) (*RabbitMQ, error) {
	conn, err := amqp.Dial(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open a channel: %v", err)
	}

	rmq := &RabbitMQ{
		conn:    conn,
		Channel: ch,
	}

	if err := rmq.setupExchangesAndQueues(); err != nil {
		rmq.Close()
		return nil, fmt.Errorf("failed to setup exchanges and queues: %v", err)
	}

	return rmq, nil
}

func (r *RabbitMQ) PublishMessage(ctx context.Context, routingKey string, message contracts.AmqpMessage) error {
	log.Printf("Publishing message with routing key: %s", routingKey)

	jsonMsg, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return r.Channel.PublishWithContext(ctx,
		TripExchange, // exchange
		routingKey,   // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType:  "text/plain",
			Body:         jsonMsg,
			DeliveryMode: amqp.Persistent,
		})
}

type MessageHandler func(context.Context, amqp.Delivery) error

func (r *RabbitMQ) ConsumeMessages(queueName string, handler MessageHandler) error {
	// Set prefetch count to 1 for fair dispatch
	// This tells RabbitMQ not to give more than one message to a service at a time.
	// The worker will only get the next message after it has acknowledged the previous one.
	err := r.Channel.Qos(
		1,     // prefetchCount: Limit to 1 unacknowledged message per consumer
		0,     // prefetchSize: No specific limit on message size
		false, // global: Apply prefetchCount to each consumer individually
	)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %v", err)
	}
	msgs, err := r.Channel.Consume(
		queueName, // queue
		"",        // consumer
		false,     // auto-ack --> automatically acknowledge message as soon as it's delivered to consumer
		false,     // exclusive --> used by only one consumer at a time
		false,     // no-local --> not supported by RabbitMQ
		false,     // no-wait --> do not wait for server confirmation
		nil,       // args
	)
	if err != nil {
		return err
	}

	ctx := context.Background() // means we are not using any timeout or cancellation for now

	go func() {
		for msg := range msgs {
			if err := handler(ctx, msg); err != nil {
				log.Fatalf("failed to handle the message: %v", err)
				// not requeuing the message to avoid potential infinite loops

				if nackErr := msg.Nack(false, true); nackErr != nil {
					log.Printf("Failed to nack message: %v", nackErr)
				}
				continue
			}

			// Acknowledge the message to remove it from the queue
			if ackErr := msg.Ack(false); ackErr != nil {
				log.Printf("ERROR: Failed to Ack message: %v. Message body: %s", ackErr, msg.Body)
			}
		}
	}()

	return nil

}

func (r *RabbitMQ) setupExchangesAndQueues() error {
	// Declare Trip Exchange
	err := r.Channel.ExchangeDeclare(
		TripExchange, // name
		"topic",      // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %s: %v", TripExchange, err)
	}

	// Declare and bind FindAvailableDriversQueue with relevant routing keys(topics) and TripExchange
	// This queue will listen to TripEventCreated and TripEventDriverNotInterested events--> sent by trip-service when a trip is created
	// and driver-service will consume messages from this queue

	if err := r.DeclareAndBindQueue(
		FindAvailableDriversQueue,
		[]string{
			contracts.TripEventCreated, contracts.TripEventDriverNotInterested,
		},
		TripExchange,
	); err != nil {
		return err
	}

	if err := r.DeclareAndBindQueue(
		DriverCmdTripRequestQueue,
		[]string{contracts.DriverCmdTripRequest},
		TripExchange,
	); err != nil {
		return err
	}
	if err := r.DeclareAndBindQueue(
		DriverTripResponseQueue,
		[]string{contracts.DriverCmdTripAccept, contracts.DriverCmdTripDecline},
		TripExchange,
	); err != nil {
		return err
	}

	if err := r.DeclareAndBindQueue(
		NotifyDriverNoDriversFoundQueue,
		[]string{contracts.TripEventNoDriversFound},
		TripExchange,
	); err != nil {
		return err
	}

	if err := r.DeclareAndBindQueue(
		NotifyDriverAssignQueue,
		[]string{contracts.TripEventDriverAssigned},
		TripExchange,
	); err != nil {
		return err
	}
	return nil
}

func (r *RabbitMQ) DeclareAndBindQueue(queueName string, messageTypes []string, exchange string) error {
	queue, err := r.Channel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %s: %v", queueName, err)
	}

	for _, msg := range messageTypes {
		if err := r.Channel.QueueBind(
			queue.Name, // queue name
			msg,        // routing key
			exchange,   // exchange
			false,
			nil,
		); err != nil {
			return fmt.Errorf("failed to bind queue to %s: %v", queueName, err)
		}
	}

	return nil
}

func (r *RabbitMQ) Close() {
	if r.conn != nil {
		r.conn.Close()
	}
}
