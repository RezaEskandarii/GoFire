package gofire

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJobHandler_Register(t *testing.T) {
	jh := NewJobHandler()

	err := jh.Register("job1", func(args ...any) error { return nil })
	assert.NoError(t, err)

	err = jh.Register("job1", func(args ...any) error { return nil })
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestJobHandler_Exists(t *testing.T) {
	jh := NewJobHandler()
	assert.False(t, jh.Exists("job1"))

	_ = jh.Register("job1", func(args ...any) error { return nil })
	assert.True(t, jh.Exists("job1"))
}

func TestJobHandler_Execute(t *testing.T) {
	jh := NewJobHandler()

	_ = jh.Register("job1", func(args ...any) error {

		assert.Len(t, args, 2)
		assert.Equal(t, "hello", args[0])
		assert.Equal(t, 123, args[1])
		return nil
	})

	err := jh.Execute("job1", "hello", 123)
	assert.NoError(t, err)

	err = jh.Execute("notfound")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	_ = jh.Register("job2", func(args ...any) error {
		return errors.New("some error")
	})
	err = jh.Execute("job2")
	assert.Error(t, err)
	assert.Equal(t, "some error", err.Error())
}

func TestJobHandler_List(t *testing.T) {
	jh := NewJobHandler()

	_ = jh.Register("job1", func(args ...any) error { return nil })
	_ = jh.Register("job2", func(args ...any) error { return nil })

	list := jh.List()
	assert.Len(t, list, 2)
	assert.Contains(t, list, "job1")
	assert.Contains(t, list, "job2")
}
