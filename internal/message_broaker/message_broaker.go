package message_broaker

type MessageBroker interface {
	Publish(queue string, message []byte) error
	Consume(queue string) (<-chan []byte, error)
	Close() error
}
