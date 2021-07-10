package appfactory

import (
	cronworker "github.com/golangid/candi/codebase/app/cron_worker"
	graphqlserver "github.com/golangid/candi/codebase/app/graphql_server"
	grpcserver "github.com/golangid/candi/codebase/app/grpc_server"
	kafkaworker "github.com/golangid/candi/codebase/app/kafka_worker"
	postgresworker "github.com/golangid/candi/codebase/app/postgres_worker"
	rabbitmqworker "github.com/golangid/candi/codebase/app/rabbitmq_worker"
	redisworker "github.com/golangid/candi/codebase/app/redis_worker"
	restserver "github.com/golangid/candi/codebase/app/rest_server"
	taskqueueworker "github.com/golangid/candi/codebase/app/task_queue_worker"
	"github.com/golangid/candi/codebase/factory"
	"github.com/golangid/candi/config/env"
)

/*
NewAppFromEnvironmentConfig constructor

Construct server/worker for running application from environment value

## Server

USE_REST=[bool]

USE_GRPC=[bool]

USE_GRAPHQL=[bool]

## Worker

USE_KAFKA_CONSUMER=[bool] # event driven handler

USE_CRON_SCHEDULER=[bool] # static scheduler

USE_REDIS_SUBSCRIBER=[bool] # dynamic scheduler

USE_TASK_QUEUE_WORKER=[bool]

USE_POSTGRES_LISTENER_WORKER=[bool]

USE_RABBITMQ_CONSUMER=[bool] # event driven handler and dynamic scheduler
*/
func NewAppFromEnvironmentConfig(service factory.ServiceFactory) (apps []factory.AppServerFactory) {

	if env.BaseEnv().UseKafkaConsumer {
		apps = append(apps, kafkaworker.NewWorker(service))
	}
	if env.BaseEnv().UseCronScheduler {
		apps = append(apps, cronworker.NewWorker(service))
	}
	if env.BaseEnv().UseTaskQueueWorker {
		apps = append(apps, taskqueueworker.NewWorker(service))
	}
	if env.BaseEnv().UseRedisSubscriber {
		apps = append(apps, redisworker.NewWorker(service))
	}
	if env.BaseEnv().UsePostgresListenerWorker {
		apps = append(apps, postgresworker.NewWorker(service))
	}
	if env.BaseEnv().UseRabbitMQWorker {
		apps = append(apps, rabbitmqworker.NewWorker(service))
	}

	sharedListener := service.GetConfig().SharedListener
	if env.BaseEnv().UseREST {
		apps = append(apps, restserver.NewServer(service, sharedListener))
	}
	if env.BaseEnv().UseGRPC {
		apps = append(apps, grpcserver.NewServer(service, sharedListener))
	}
	if !env.BaseEnv().UseREST && env.BaseEnv().UseGraphQL {
		apps = append(apps, graphqlserver.NewServer(service, sharedListener))
	}

	return
}
