package linechatbot

import (
	"context"
	"os"

	"agungdwiprasetyo.com/backend-microservices/config"
	"agungdwiprasetyo.com/backend-microservices/config/broker"
	"agungdwiprasetyo.com/backend-microservices/config/database"
	"agungdwiprasetyo.com/backend-microservices/internal/line-chatbot/modules/chatbot"
	"agungdwiprasetyo.com/backend-microservices/internal/line-chatbot/modules/event"
	"agungdwiprasetyo.com/backend-microservices/pkg/codebase/factory"
	"agungdwiprasetyo.com/backend-microservices/pkg/codebase/factory/constant"
	"agungdwiprasetyo.com/backend-microservices/pkg/codebase/factory/dependency"
	"agungdwiprasetyo.com/backend-microservices/pkg/codebase/interfaces"
	"agungdwiprasetyo.com/backend-microservices/pkg/middleware"
	authsdk "agungdwiprasetyo.com/backend-microservices/pkg/sdk/auth-service"
	"agungdwiprasetyo.com/backend-microservices/pkg/validator"
)

// Service model
type Service struct {
	deps    dependency.Dependency
	modules []factory.ModuleFactory
	name    constant.Service
}

// NewService in this service
func NewService(serviceName string, cfg *config.Config) factory.ServiceFactory {
	// See all option in dependency package
	var deps dependency.Dependency

	cfg.LoadFunc(func(ctx context.Context) []interfaces.Closer {
		kafkaDeps := broker.InitKafkaBroker(config.BaseEnv().Kafka.ClientID)
		mongoDeps := database.InitMongoDB(ctx)

		authService := authsdk.NewAuthServiceGRPC(os.Getenv("AUTH_SERVICE_GRPC_HOST"), os.Getenv("AUTH_SERVICE_GRPC_KEY"))

		// inject all service dependencies
		deps = dependency.InitDependency(
			dependency.SetMiddleware(middleware.NewMiddleware(authService)),
			dependency.SetValidator(validator.NewJSONSchemaValidator(serviceName)),
			dependency.SetBroker(kafkaDeps),
			dependency.SetMongoDatabase(mongoDeps),
			// ... add more dependencies
		)
		return []interfaces.Closer{kafkaDeps, mongoDeps} // throw back to config for close connection when application shutdown
	})

	modules := []factory.ModuleFactory{
		chatbot.NewModule(deps),
		event.NewModule(deps),
	}

	return &Service{
		deps:    deps,
		modules: modules,
		name:    constant.Service(serviceName),
	}
}

// GetDependency method
func (s *Service) GetDependency() dependency.Dependency {
	return s.deps
}

// GetModules method
func (s *Service) GetModules() []factory.ModuleFactory {
	return s.modules
}

// Name method
func (s *Service) Name() constant.Service {
	return s.name
}
