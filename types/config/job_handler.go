package config

import (
	"fmt"
	"sync"
)

type JobHandler struct {
	handlers map[string]func(args ...any) error
	mutex    sync.Mutex
}

func NewJobHandler() *JobHandler {
	return &JobHandler{
		handlers: make(map[string]func(args ...any) error),
	}
}

// Register adds a new job handler by name.
func (jh *JobHandler) Register(name string, handler func(args ...any) error) error {
	jh.mutex.Lock()
	defer jh.mutex.Unlock()

	if _, exists := jh.handlers[name]; exists {
		return fmt.Errorf("handler '%s' already registered", name)
	}
	jh.handlers[name] = handler
	return nil
}

func (jh *JobHandler) Exists(name string) bool {
	jh.mutex.Lock()
	defer jh.mutex.Unlock()

	_, exists := jh.handlers[name]
	return exists
}

func (jh *JobHandler) Execute(name string, args ...any) error {
	handler, exists := jh.handlers[name]
	if !exists {
		return fmt.Errorf("handler '%s' not found", name)
	}
	return handler(args...)
}

func (jh *JobHandler) List() []string {
	jh.mutex.Lock()
	defer jh.mutex.Unlock()

	names := make([]string, 0, len(jh.handlers))
	for name := range jh.handlers {
		names = append(names, name)
	}
	return names
}
