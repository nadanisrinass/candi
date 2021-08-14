package broker

import (
	"context"

	"github.com/golangid/candi/codebase/factory/types"
	"github.com/golangid/candi/codebase/interfaces"
	"github.com/golangid/candi/config/env"
	"github.com/golangid/candi/logger"
	"github.com/golangid/candi/publisher"
	"github.com/streadway/amqp"
)

// RabbitMQOptionFunc func type
type RabbitMQOptionFunc func(*RabbitMQBroker)

// RabbitMQSetChannel set custom channel configuration
func RabbitMQSetChannel(ch *amqp.Channel) RabbitMQOptionFunc {
	return func(bk *RabbitMQBroker) {
		bk.ch = ch
	}
}

// RabbitMQSetPublisher set custom publisher
func RabbitMQSetPublisher(pub interfaces.Publisher) RabbitMQOptionFunc {
	return func(bk *RabbitMQBroker) {
		bk.pub = pub
	}
}

// RabbitMQBroker broker
type RabbitMQBroker struct {
	conn *amqp.Connection
	ch   *amqp.Channel
	pub  interfaces.Publisher
}

// NewRabbitMQBroker constructor, connection from RABBITMQ_BROKER environment
func NewRabbitMQBroker(opts ...RabbitMQOptionFunc) *RabbitMQBroker {
	deferFunc := logger.LogWithDefer("Load RabbitMQ broker configuration... ")
	defer deferFunc()
	var err error

	rabbitmq := new(RabbitMQBroker)
	for _, opt := range opts {
		opt(rabbitmq)
	}

	rabbitmq.conn, err = amqp.Dial(env.BaseEnv().RabbitMQ.Broker)
	if err != nil {
		panic("RabbitMQ: cannot connect to server broker: " + err.Error())
	}

	if rabbitmq.ch == nil {
		// set default configuration
		rabbitmq.ch, err = rabbitmq.conn.Channel()
		if err != nil {
			panic("RabbitMQ channel: " + err.Error())
		}
		if err := rabbitmq.ch.ExchangeDeclare("amq.direct", "direct", true, false, false, false, nil); err != nil {
			panic("RabbitMQ exchange declare direct: " + err.Error())
		}
		if err := rabbitmq.ch.ExchangeDeclare(
			env.BaseEnv().RabbitMQ.ExchangeName, // name
			"x-delayed-message",                 // type
			true,                                // durable
			false,                               // auto-deleted
			false,                               // internal
			false,                               // no-wait
			amqp.Table{
				"x-delayed-type": "direct",
			},
		); err != nil {
			panic("RabbitMQ exchange declare delayed: " + err.Error())
		}
		if err := rabbitmq.ch.Qos(2, 0, false); err != nil {
			panic("RabbitMQ Qos: " + err.Error())
		}
	}

	if rabbitmq.pub == nil {
		rabbitmq.pub = publisher.NewRabbitMQPublisher(rabbitmq.conn)
	}

	return rabbitmq
}

// GetConfiguration method
func (r *RabbitMQBroker) GetConfiguration() interface{} {
	return r.ch
}

// GetPublisher method
func (r *RabbitMQBroker) GetPublisher() interfaces.Publisher {
	return r.pub
}

// Health method
func (r *RabbitMQBroker) Health() map[string]error {
	return map[string]error{string(types.RabbitMQ): nil}
}

// Disconnect method
func (r *RabbitMQBroker) Disconnect(ctx context.Context) error {
	deferFunc := logger.LogWithDefer("rabbitmq: disconnect...")
	defer deferFunc()

	return r.conn.Close()
}
