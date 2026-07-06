package broker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gabrielgcosta/ticketblast-core/internal/usecase"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQPublisher struct {
	conn     *amqp.Connection
	ch       *amqp.Channel
	exchange string
}

func NewRabbitMQPublisher(url string, exchange string) (*RabbitMQPublisher, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	// Declare exchange as persistent/durable
	err = ch.ExchangeDeclare(
		exchange, // name
		"direct",  // type
		true,      // durable
		false,     // auto-deleted
		false,     // internal
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Proactively declare and bind the orders queue to ensure it is created
	q, err := ch.QueueDeclare(
		"orders_queue", // name
		true,           // durable
		false,          // delete when unused
		false,          // exclusive
		false,          // no-wait
		nil,            // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	err = ch.QueueBind(
		q.Name,          // queue name
		"order.created", // routing key
		exchange,        // exchange
		false,           // no-wait
		nil,             // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to bind queue to exchange: %w", err)
	}

	return &RabbitMQPublisher{
		conn:     conn,
		ch:       ch,
		exchange: exchange,
	}, nil
}

func (p *RabbitMQPublisher) PublishOrderCreated(ctx context.Context, event *usecase.OrderCreatedEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal order event: %w", err)
	}

	err = p.ch.PublishWithContext(ctx,
		p.exchange,      // exchange
		"order.created", // routing key
		false,           // mandatory
		false,           // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent, // Persistent delivery mode (saves message to disk)
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message to RabbitMQ: %w", err)
	}

	return nil
}

func (p *RabbitMQPublisher) Close() error {
	if p.ch != nil {
		p.ch.Close()
	}
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}
