package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/retry"
	"ride-sharing/shared/tracing"
	"strings"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	TripExchange       = "trip"
	DeadLetterExchange = "dlx"
)

type RabbitMQ struct {
	conn    *amqp.Connection
	Channel *amqp.Channel
	uri     string
	mu      sync.Mutex
}

func NewRabbitMQ(uri string) (*RabbitMQ, error) {
	conn, err := amqp.Dial(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create channel: %v", err)
	}

	rmq := &RabbitMQ{
		conn:    conn,
		Channel: ch,
		uri:     uri,
	}

	if err := rmq.setupExchangesAndQueues(); err != nil {
		// Clean up if setup fails
		rmq.Close()
		return nil, fmt.Errorf("failed to setup exchanges and queues: %v", err)
	}

	return rmq, nil
}

type MessageHandler func(context.Context, amqp.Delivery) error

func (r *RabbitMQ) ConsumeMessages(queueName string, handler MessageHandler) error {
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
		false,     // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		return err
	}

	go func() {
		for msg := range msgs {
			if err := tracing.TracedConsumer(msg, func(ctx context.Context, d amqp.Delivery) error {
				log.Printf("Received a message: %s", msg.Body)

				cfg := retry.DefaultConfig()
				err := retry.WithBackoff(ctx, cfg, handler, d)
				if err != nil {
					log.Printf("Message processing failed after %d retries for message ID: %s, err: %v", cfg.MaxRetries, d.MessageId, err)

					// Add failure context before sending to the DLQ
					headers := amqp.Table{}
					if d.Headers != nil {
						headers = d.Headers
					}

					headers["x-death-reason"] = err.Error()
					headers["x-origin-exchange"] = d.Exchange
					headers["x-original-routing-key"] = d.RoutingKey
					headers["x-retry-count"] = cfg.MaxRetries
					d.Headers = headers

					// Reject without requeue - message will go to the DLQ
					_ = d.Reject(false)
					return err
				}

				// Only Ack if the handler succeeds
				if ackErr := msg.Ack(false); ackErr != nil {
					log.Printf("ERROR: Failed to Ack message: %v. Message body: %s", ackErr, msg.Body)
				}

				return nil
			}); err != nil {
				log.Printf("Error processing message: %v", err)
			}
		}
	}()

	return nil
}

func (r *RabbitMQ) PublishMessage(ctx context.Context, routingKey string, message contracts.AmqpMessage) error {

	jsonMsg, err := json.Marshal(message) // converts go struct into JSON encoded [] byte
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	log.Printf("Publishing message in queue: %v", string(jsonMsg))

	msg := amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		ContentType:  "application/json",
		Body:         jsonMsg,
	}

	return tracing.TracedPublisher(ctx, TripExchange, routingKey, msg, r.publish)
}

func (r *RabbitMQ) reconnect() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// If the channel is still usable, nothing to do.
	if r.Channel != nil && !r.conn.IsClosed() {
		if ch, err := r.conn.Channel(); err == nil {
			r.Channel = ch
			if err := r.setupExchangesAndQueues(); err == nil {
				return nil
			}
		}
	}

	// Full reconnect.
	conn, err := amqp.Dial(r.uri)
	if err != nil {
		return fmt.Errorf("rabbitmq reconnect dial: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("rabbitmq reconnect channel: %w", err)
	}
	if r.conn != nil {
		r.conn.Close()
	}
	r.conn = conn
	r.Channel = ch
	if err := r.setupExchangesAndQueues(); err != nil {
		return fmt.Errorf("rabbitmq reconnect setup: %w", err)
	}
	log.Println("RabbitMQ reconnected successfully")
	return nil
}

func (r *RabbitMQ) publish(ctx context.Context, exchange, routingKey string, msg amqp.Publishing) error {
	err := r.Channel.PublishWithContext(ctx,
		exchange,
		routingKey,
		false,
		false,
		msg,
	)
	if err == nil {
		return nil
	}
	// Channel/connection is dead — reconnect and retry once.
	if strings.Contains(err.Error(), "channel/connection is not open") ||
		strings.Contains(err.Error(), "Exception (504)") {
		log.Printf("RabbitMQ channel lost (%v) — reconnecting", err)
		if reconnErr := r.reconnect(); reconnErr != nil {
			return fmt.Errorf("publish failed and reconnect failed: %w", reconnErr)
		}
		return r.Channel.PublishWithContext(ctx, exchange, routingKey, false, false, msg)
	}
	return err
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

func (r *RabbitMQ) setupExchangesAndQueues() error {
	// First setup the DLQ exchange and queue
	if err := r.setupDeadLetterExchange(); err != nil {
		return err
	}

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

	if err := r.declareAndBindQueue(
		FindAvailableDriversQueue,
		[]string{
			contracts.TripEventCreated, contracts.TripEventDriverNotInterested,
		},
		TripExchange,
	); err != nil {
		return err
	}

	if err := r.declareAndBindQueue(
		NotifyTripCreatedQueue,
		[]string{contracts.TripEventCreated},
		TripExchange,
	); err != nil {
		return err
	}

	if err := r.declareAndBindQueue(
		DriverCmdTripRequestQueue,
		[]string{contracts.DriverCmdTripRequest},
		TripExchange,
	); err != nil {
		return err
	}

	if err := r.declareAndBindQueue(
		DriverTripResponseQueue,
		[]string{contracts.DriverCmdTripAccept, contracts.DriverCmdTripDecline},
		TripExchange,
	); err != nil {
		return err
	}

	if err := r.declareAndBindQueue(
		NotifyDriverNoDriversFoundQueue,
		[]string{contracts.TripEventNoDriversFound},
		TripExchange,
	); err != nil {
		return err
	}

	if err := r.declareAndBindQueue(
		NotifyDriverAssignQueue,
		[]string{contracts.TripEventDriverAssigned},
		TripExchange,
	); err != nil {
		return err
	}

	if err := r.declareAndBindQueue(
		PaymentTripResponseQueue,
		[]string{contracts.PaymentCmdCreateSession},
		TripExchange,
	); err != nil {
		return err
	}

	if err := r.declareAndBindQueue(
		NotifyPaymentSessionCreatedQueue,
		[]string{contracts.PaymentEventSessionCreated},
		TripExchange,
	); err != nil {
		return err
	}

	if err := r.declareAndBindQueue(
		NotifyPaymentSuccessQueue,
		[]string{contracts.PaymentEventSuccess},
		TripExchange,
	); err != nil {
		return err
	}

	if err := r.declareAndBindQueue(
		DriverLocationUpdateQueue,
		[]string{contracts.DriverCmdLocation},
		TripExchange,
	); err != nil {
		return err
	}

	if err := r.declareAndBindQueue(
		DriverTripAssignedQueue,
		[]string{contracts.TripEventDriverAssigned},
		TripExchange,
	); err != nil {
		return err
	}

	if err := r.declareAndBindQueue(
		NotifyRiderDriverLocationQueue,
		[]string{contracts.DriverEventLocation},
		TripExchange,
	); err != nil {
		return err
	}

	if err := r.declareAndBindQueue(
		ChatCmdSendQueue,
		[]string{contracts.ChatCmdSend},
		TripExchange,
	); err != nil {
		return err
	}

	if err := r.declareAndBindQueue(
		ChatEventDeliveredQueue,
		[]string{contracts.ChatEventDelivered},
		TripExchange,
	); err != nil {
		return err
	}

	if err := r.declareAndBindQueue(
		NotifyTripCancelledQueue,
		[]string{contracts.TripEventCancelled},
		TripExchange,
	); err != nil {
		return err
	}

	return nil
}

func (r *RabbitMQ) declareAndBindQueue(queueName string, messageTypes []string, exchange string) error {
	// Add dead letter configuration
	args := amqp.Table{
		"x-dead-letter-exchange": DeadLetterExchange,
	}

	q, err := r.Channel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		args,      // arguments with DLX config
	)
	if err != nil {
		log.Fatal(err)
	}

	for _, msg := range messageTypes {
		if err := r.Channel.QueueBind(
			q.Name,   // queue name
			msg,      // routing key
			exchange, // exchange
			false,
			nil,
		); err != nil {
			return fmt.Errorf("failed to bind queue to %s: %v", queueName, err)
		}
	}

	return nil
}

func (rmq *RabbitMQ) StartDLQConsumer(ctx context.Context) error {
	msgs, err := rmq.Channel.Consume(
		DeadLetterQueue,
		"",
		false, // manual ack
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to start DLQ consumer: %v", err)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		go func() {
			for msg := range msgs {
				if msg.Headers == nil {
					msg.Headers = amqp.Table{}
				}
				headers := msg.Headers

				originExchange, _ := headers["x-origin-exchange"].(string)
				originalRoutingKey, _ := headers["x-original-routing-key"].(string)
				reason, _ := headers["x-death-reason"].(string)

				if originExchange == "" || originalRoutingKey == "" {
					log.Printf("⚠️ Missing origin info, discarding DLQ msg: %s", msg.Body)
					msg.Ack(false)
					continue
				}

				// Parse broker retry count safely
				var brokerRetryCount int32
				if val, ok := headers["broker-retry-count"]; ok {
					switch v := val.(type) {
					case int32:
						brokerRetryCount = v
					case int64:
						brokerRetryCount = int32(v)
					case float64:
						brokerRetryCount = int32(v)
					}
				}
				brokerRetryCount++
				headers["broker-retry-count"] = brokerRetryCount
				msg.Headers = headers

				if brokerRetryCount > 5 {
					log.Printf("⚠️ Dropping DLQ message after %d broker retries. Reason: %s", brokerRetryCount, reason)
					msg.Ack(false)
					continue
				}

				// Small delay before retrying (progressively increases)
				delay := time.Duration(brokerRetryCount*5) * time.Second
				log.Printf("⏳ Retrying DLQ message after %v (exchange=%s, key=%s)", delay, originExchange, originalRoutingKey)
				time.Sleep(delay)

				// Republish to original exchange/routing key
				if err := rmq.retryDLQMessage(originExchange, originalRoutingKey, msg); err != nil {
					log.Printf("⚠️ Failed to republish DLQ msg: %v", err)
					msg.Nack(false, true) // requeue for another try later
					continue
				}

				log.Printf("✅ Republished DLQ msg to %s/%s (retry %d)", originExchange, originalRoutingKey, brokerRetryCount)
				msg.Ack(false)
			}
		}()

	}

	return nil
}

func (r *RabbitMQ) retryDLQMessage(exchange string, routingKey string, msg amqp.Delivery) error {
	pub := amqp.Publishing{
		Headers:      msg.Headers,
		ContentType:  msg.ContentType,
		Body:         msg.Body,
		DeliveryMode: amqp.Persistent,
		MessageId:    msg.MessageId,
		Timestamp:    time.Now(),
	}
	return r.Channel.Publish(exchange, routingKey, false, false, pub)
}

func (r *RabbitMQ) Close() {
	if r.conn != nil {
		r.conn.Close()
	}
	if r.Channel != nil {
		r.Channel.Close()
	}
}
