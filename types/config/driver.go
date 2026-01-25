package config

type StorageDriver int

const (
	Postgres StorageDriver = iota + 1
	//	Redis                  // seconds
)

type MessageQueueDriver int

const (
	RabbitMQ MessageQueueDriver = iota + 1
	//	Kafka
)

func (d MessageQueueDriver) String() string {
	switch d {
	case RabbitMQ:
		return "rabbitmq"
	//case Kafka:
	//	return "kafka"
	default:
		return "unknown"
	}

}

// String converts the StorageDriver enum to a human-readable string.
func (d StorageDriver) String() string {
	switch d {
	//case Redis:
	//	return "redis"
	case Postgres:
		return "postgres"
	}
	return "unknown"
}
