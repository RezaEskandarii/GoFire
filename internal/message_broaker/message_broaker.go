package message_broaker

import "context"

type MessageBroker interface {
	Publish(queue string, message []byte) error
	Consume(ctx context.Context, queue string) (<-chan []byte, error)
	Close() error
}
