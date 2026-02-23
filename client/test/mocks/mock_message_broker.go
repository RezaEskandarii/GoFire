package mocks

import "context"

// MockMessageBroker is a mock implementation of message_broaker.MessageBroker for testing.
type MockMessageBroker struct {
	PublishFunc func(queue string, message []byte) error
	ConsumeFunc func(ctx context.Context, queue string) (<-chan []byte, error)
	CloseFunc   func() error
}

func (m *MockMessageBroker) Publish(queue string, message []byte) error {
	if m.PublishFunc != nil {
		return m.PublishFunc(queue, message)
	}
	return nil
}

func (m *MockMessageBroker) Consume(ctx context.Context, queue string) (<-chan []byte, error) {
	if m.ConsumeFunc != nil {
		return m.ConsumeFunc(ctx, queue)
	}
	ch := make(chan []byte)
	close(ch)
	return ch, nil
}

func (m *MockMessageBroker) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}
