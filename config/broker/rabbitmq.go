package broker

import (
	"github.com/golangid/candi/codebase/interfaces"
	"github.com/golangid/candi/config/env"
	"github.com/golangid/candi/logger"
	"github.com/golangid/candi/publisher"
	"github.com/streadway/amqp"
)

type rabbitmqBroker struct {
	conn *amqp.Connection
	pub  interfaces.Publisher
}

func initRabbitMQBroker() *rabbitmqBroker {
	deferFunc := logger.LogWithDefer("Load RabbitMQ broker configuration... ")
	defer deferFunc()

	conn, err := amqp.Dial(env.BaseEnv().RabbitMQ.Broker)
	if err != nil {
		panic("RabbitMQ: cannot connect to server broker: " + err.Error())
	}
	return &rabbitmqBroker{
		conn: conn,
		pub:  publisher.NewRabbitMQPublisher(conn),
	}
}
