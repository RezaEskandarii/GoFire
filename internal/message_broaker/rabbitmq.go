package message_broaker

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	conn       *amqp.Connection
	channel    *amqp.Channel
	queueName  string
	exchange   string
	routingKey string
}

// NewRabbitMQ creates a new instance of RabbitMQ message broker.
func NewRabbitMQ(url, exchange, queue, routingKey string) (*RabbitMQ, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	if err := ch.ExchangeDeclare(
		exchange,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}

	if _, err := ch.QueueDeclare(
		queue,
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}

	if err := ch.QueueBind(
		queue,
		routingKey,
		exchange,
		false,
		nil,
	); err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}

	return &RabbitMQ{
		conn:       conn,
		channel:    ch,
		queueName:  queue,
		exchange:   exchange,
		routingKey: routingKey,
	}, nil
}

func (r *RabbitMQ) Publish(queue string, message []byte) error {
	return r.channel.Publish(
		r.exchange,
		r.routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        message,
		},
	)
}

func (r *RabbitMQ) Consume(queue string) (<-chan []byte, error) {
	msgs, err := r.channel.Consume(
		queue,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	out := make(chan []byte)

	go func() {
		for msg := range msgs {
			out <- msg.Body
		}
		close(out)
	}()

	return out, nil
}

func (r *RabbitMQ) Close() error {
	if err := r.channel.Close(); err != nil {
		_ = r.conn.Close()
		return err
	}
	return r.conn.Close()
}
