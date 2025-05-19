package models

type StorageDriver int

const (
	Postgres StorageDriver = iota + 1
	Redis
)

type MethodHandler struct {
	MethodName string
	Func       func([]interface{}) error
}

type Config struct {
	Connection           string
	DashboardPort        int
	DashboardAuthEnabled bool
	DashboardUserName    string
	DashboardPassword    string
	Instance             string
	EnableDashboard      bool
	Handlers             []MethodHandler
	Driver               StorageDriver
}

func (c Config) RegisterHandler(handler MethodHandler) Config {
	c.Handlers = append(c.Handlers, handler)
	return c
}
