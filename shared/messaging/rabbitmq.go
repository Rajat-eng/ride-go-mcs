package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/retry"
	"ride-sharing/shared/tracing"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	TripExchange       = "trip"
	DeadLetterExchange = "dlx"
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
	msg := amqp.Publishing{
		ContentType:  "text/plain",
		Body:         jsonMsg,
		DeliveryMode: amqp.Persistent,
	}
	return tracing.TracedPublisher(ctx, TripExchange, routingKey, msg, r.publish)

}

func (r *RabbitMQ) publish(ctx context.Context, exchange, routingKey string, msg amqp.Publishing) error {
	return r.Channel.PublishWithContext(ctx,
		TripExchange, // exchange
		routingKey,   // routing key
		false,        // mandatory
		false,
		msg,
	)
}

type MessageHandler func(context.Context, amqp.Delivery) error

func (r *RabbitMQ) ConsumeMessages(queueName string, handler MessageHandler) error {
	// Fair dispatch: limit to 1 unacknowledged message per consumer
	err := r.Channel.Qos(
		1,     // prefetchCount
		0,     // prefetchSize
		false, // apply per-consumer, not global
	)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %v", err)
	}

	msgs, err := r.Channel.Consume(
		queueName, // queue
		"",        // consumer tag
		false,     // auto-ack (we want manual ack/nack)
		false,     // exclusive consumer
		false,     // no-local (not supported)
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		return err
	}

	go func() {
		for msg := range msgs {
			// Wrap handler with tracing
			err := tracing.TracedConsumer(msg, func(ctx context.Context, d amqp.Delivery) error {
				// Use retry with backoff for the actual handler logic
				cfg := retry.DefaultConfig()
				retryErr := retry.WithBackoff(ctx, cfg, func() error {
					return handler(ctx, d)
				})
				if retryErr != nil {
					// Mark message as permanently failed -> send to DLQ
					log.Printf("Message processing failed after %d retries. ID: %s, err: %v",
						cfg.MaxRetries, d.MessageId, retryErr)

					// Copy or initialize headers
					headers := amqp.Table{}
					if d.Headers != nil {
						headers = d.Headers
					}

					// Attach failure metadata
					headers["x-death-reason"] = retryErr.Error()
					headers["x-origin-exchange"] = d.Exchange
					headers["x-original-routing-key"] = d.RoutingKey
					headers["x-retry-count"] = cfg.MaxRetries
					d.Headers = headers

					// Reject without requeue -> DLQ (if configured)
					if rejErr := d.Reject(false); rejErr != nil {
						log.Printf("Failed to reject message: %v", rejErr)
					}

					return retryErr
				}

				// If retry succeeded, ack the message
				if ackErr := d.Ack(false); ackErr != nil {
					log.Printf("ERROR: Failed to Ack message: %v. Body: %s", ackErr, d.Body)
				}

				return nil
			})

			if err != nil {
				// TracedConsumer failed unexpectedly (should be rare)
				log.Printf("Tracing wrapper failed: %v", err)

				// Nack without requeue to avoid infinite loops
				if nackErr := msg.Nack(false, false); nackErr != nil {
					log.Printf("Failed to nack message: %v", nackErr)
				}
			}
		}
	}()

	return nil
}

func (r *RabbitMQ) setupExchangesAndQueues() error {

	if err := r.setupDeadLetterExchange(); err != nil {
		return err
	}
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

	// driver assigned --> update trip and send event to payment service to create payment session
	if err := r.DeclareAndBindQueue(
		PaymentTripResponseQueue,
		[]string{contracts.PaymentCmdCreateSession},
		TripExchange,
	); err != nil {
		return err
	}

	// create payment session--> pay using stripe on pay button and notify user
	if err := r.DeclareAndBindQueue(
		NotifyPaymentSessionCreatedQueue,
		[]string{contracts.PaymentEventSessionCreated},
		TripExchange,
	); err != nil {
		return err
	}
	if err := r.DeclareAndBindQueue(
		NotifyPaymentSuccessQueue,
		[]string{contracts.PaymentEventSuccess},
		TripExchange,
	); err != nil {
		return err
	}
	return nil
}

func (r *RabbitMQ) setupDeadLetterExchange() error {
	// Declare the dead letter exchange
	err := r.Channel.ExchangeDeclare(
		DeadLetterExchange,
		"topic",
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare dead letter exchange: %v", err)
	}

	// Declare the dead letter queue
	q, err := r.Channel.QueueDeclare(
		DeadLetterQueue,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare dead letter queue: %v", err)
	}

	// Bind the queue to the exchange with a wildcard routing key
	err = r.Channel.QueueBind(
		q.Name,
		"#", // wildcard routing key to catch all messages
		DeadLetterExchange,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind dead letter queue: %v", err)
	}

	return nil
}

func (r *RabbitMQ) DeclareAndBindQueue(queueName string, messageTypes []string, exchange string) error {
	args := amqp.Table{
		"x-dead-letter-exchange": DeadLetterExchange,
	} // all noraml queues are bind to dlq--> when message is rejected send it to dealletterexchange
	queue, err := r.Channel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		args,      // arguments
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
