package mocks

import (
	"context"
	"errors"
	"sync"
)

type MockMessageBroker struct {
	mu       sync.Mutex
	queues   map[string]chan []byte
	capacity int
	closed   bool
}

func NewMockMessageBroker(capacity int) *MockMessageBroker {
	return &MockMessageBroker{
		queues:   make(map[string]chan []byte),
		capacity: capacity,
	}
}

func (m *MockMessageBroker) Publish(queue string, message []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return errors.New("broker is closed")
	}

	ch, exists := m.queues[queue]
	if !exists {
		ch = make(chan []byte, m.capacity)
		m.queues[queue] = ch
	}

	select {
	case ch <- message:
		return nil
	default:
		return errors.New("queue is full")
	}
}

func (m *MockMessageBroker) Consume(ctx context.Context, queue string) (<-chan []byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil, errors.New("broker is closed")
	}

	ch, exists := m.queues[queue]
	if !exists {
		ch = make(chan []byte, m.capacity)
		m.queues[queue] = ch
	}

	out := make(chan []byte)

	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				out <- msg
			}
		}
	}()

	return out, nil
}

func (m *MockMessageBroker) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return errors.New("already closed")
	}

	for _, ch := range m.queues {
		close(ch)
	}

	m.closed = true
	return nil
}
