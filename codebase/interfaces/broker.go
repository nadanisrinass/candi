package interfaces

import (
	"github.com/Shopify/sarama"
	"github.com/golangid/candi/codebase/factory/types"
	"github.com/streadway/amqp"
)

// Broker abstraction
type Broker interface {
	GetKafkaClient() sarama.Client
	GetRabbitMQConn() *amqp.Connection
	Publisher(types.Worker) Publisher
	Health() map[string]error
	Closer
}
