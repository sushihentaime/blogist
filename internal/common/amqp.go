package common

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Exchange string

type Queue string

type BindingKey string

type MessageProducer interface {
	Publish(ctx context.Context, msg []byte, key BindingKey, exchange Exchange) error
}

type MessageConsumer interface {
	Consume(key BindingKey, exchange Exchange, queue Queue) (<-chan amqp.Delivery, error)
}

const (
	UserExchange     Exchange   = "user_exchange"
	UserCreatedQueue Queue      = "user_created_queue"
	UserCreatedKey   BindingKey = "user.created"
)

type MessageBroker struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

func NewMessageBroker(URI string) (*MessageBroker, error) {
	conn, ch, err := connectAMQP(URI)
	if err != nil {
		return nil, err
	}

	return &MessageBroker{
		conn: conn,
		ch:   ch,
	}, nil
}

func connectAMQP(URI string) (*amqp.Connection, *amqp.Channel, error) {
	conn, err := amqp.Dial(URI)
	if err != nil {
		return nil, nil, fmt.Errorf("could not connect to AMQP: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("could not open channel: %w", err)
	}

	return conn, ch, nil
}

// Close closes the connection and channel of the message broker.
func (mb *MessageBroker) Close() error {
	err := mb.ch.Close()
	if err != nil {
		return err
	}

	err = mb.conn.Close()
	if err != nil {
		return err
	}

	return nil
}

func SetupUserExchange(mb *MessageBroker) error {
	err := mb.ch.ExchangeDeclare(string(UserExchange), "direct", true, false, false, false, nil)
	if err != nil {
		return err
	}

	_, err = mb.ch.QueueDeclare(string(UserCreatedQueue), true, false, false, false, nil)
	if err != nil {
		return err
	}

	err = mb.ch.QueueBind(string(UserCreatedQueue), string(UserCreatedKey), string(UserExchange), false, nil)
	if err != nil {
		return err
	}

	return nil
}

func (mb *MessageBroker) Publish(ctx context.Context, msg []byte, key BindingKey, exchange Exchange) error {
	err := mb.ch.PublishWithContext(ctx, string(exchange), string(key), false, false, amqp.Publishing{
		ContentType: "text/plain",
		Body:        msg,
	})
	if err != nil {
		return fmt.Errorf("could not publish message: %w", err)
	}

	return nil
}

func (mb *MessageBroker) Consume(key BindingKey, exchange Exchange, queue Queue) (<-chan amqp.Delivery, error) {
	msgs, err := mb.ch.Consume(string(queue), string(key), false, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("could not consume message: %w", err)
	}

	return msgs, nil
}
