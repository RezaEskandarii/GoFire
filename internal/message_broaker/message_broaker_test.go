package message_broaker

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

// mockBroker is a minimal implementation of MessageBroker for testing the interface contract.
type mockBroker struct {
	publishErr error
	closeErr   error
}

func (m *mockBroker) Publish(queue string, message []byte) error { return m.publishErr }
func (m *mockBroker) Consume(ctx context.Context, queue string) (<-chan []byte, error) {
	ch := make(chan []byte)
	close(ch)
	return ch, nil
}
func (m *mockBroker) Close() error { return m.closeErr }

func TestMessageBrokerInterface(t *testing.T) {
	var _ MessageBroker = (*mockBroker)(nil)
}

func TestMockBroker_Publish(t *testing.T) {
	broker := &mockBroker{}
	err := broker.Publish("queue", []byte("msg"))
	require.NoError(t, err)
}

func TestMockBroker_Publish_Error(t *testing.T) {
	broker := &mockBroker{publishErr: assert.AnError}
	err := broker.Publish("queue", []byte("msg"))
	assert.Error(t, err)
}

func TestMockBroker_Consume(t *testing.T) {
	broker := &mockBroker{}
	ctx := context.Background()
	ch, err := broker.Consume(ctx, "queue")
	require.NoError(t, err)
	require.NotNil(t, ch)
	_, ok := <-ch
	assert.False(t, ok) // channel is closed
}

func TestMockBroker_Close(t *testing.T) {
	broker := &mockBroker{}
	err := broker.Close()
	require.NoError(t, err)
}
