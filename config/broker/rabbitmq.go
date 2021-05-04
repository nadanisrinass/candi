package broker

import (
	"github.com/golangid/candi/codebase/interfaces"
	"github.com/golangid/candi/config/env"
	"github.com/golangid/candi/logger"
	"github.com/golangid/candi/publisher"
	"github.com/streadway/amqp"
)

// RabbitMQBroker broker
type RabbitMQBroker struct {
	conn *amqp.Connection
	pub  interfaces.Publisher
}

// NewRabbitMQBroker constructor, connection from RABBITMQ_BROKER environment
func NewRabbitMQBroker() *RabbitMQBroker {
	deferFunc := logger.LogWithDefer("Load RabbitMQ broker configuration... ")
	defer deferFunc()

	conn, err := amqp.Dial(env.BaseEnv().RabbitMQ.Broker)
	if err != nil {
		panic("RabbitMQ: cannot connect to server broker: " + err.Error())
	}
	return &RabbitMQBroker{
		conn: conn,
		pub:  publisher.NewRabbitMQPublisher(conn),
	}
}
