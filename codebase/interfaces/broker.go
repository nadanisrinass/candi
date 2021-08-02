package interfaces

import (
	"github.com/golangid/candi/codebase/factory/types"
)

// Broker abstraction
type Broker interface {
	GetConfiguration(types.Worker) interface{} // get broker configuration (different type for each broker)
	Publisher(types.Worker) Publisher
	Health() map[string]error
	Closer
}
