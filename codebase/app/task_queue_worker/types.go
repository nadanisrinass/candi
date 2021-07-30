package taskqueueworker

import (
	"errors"
	"net/url"
	"reflect"
	"sync"
	"time"

	"github.com/golangid/candi/codebase/factory/types"
	"github.com/golangid/candi/config/env"
	"go.mongodb.org/mongo-driver/mongo"
)

type (
	// TaglineResolver resolver
	TaglineResolver struct {
		Tagline                   string
		TaskListClientSubscribers []string
		JobListClientSubscribers  []string
		MemoryStatistics          MemstatsResolver
	}
	// MemstatsResolver resolver
	MemstatsResolver struct {
		Alloc         string
		TotalAlloc    string
		NumGC         int
		NumGoroutines int
	}
	// MetaTaskResolver meta resolver
	MetaTaskResolver struct {
		Page           int
		Limit          int
		TotalRecords   int
		TotalPages     int
		IsCloseSession bool
	}
	// TaskResolver resolver
	TaskResolver struct {
		Name      string
		TotalJobs int
		Detail    struct {
			GiveUp, Retrying, Success, Queueing, Stopped int
		}
	}
	// TaskListResolver resolver
	TaskListResolver struct {
		Meta MetaTaskResolver
		Data []TaskResolver
	}

	// MetaJobList resolver
	MetaJobList struct {
		Page           int
		Limit          int
		TotalRecords   int
		TotalPages     int
		IsCloseSession bool
		Detail         struct {
			GiveUp, Retrying, Success, Queueing, Stopped int
		}
	}

	// JobListResolver resolver
	JobListResolver struct {
		Meta MetaJobList
		Data []Job
	}

	// Filter type
	Filter struct {
		Page, Limit int
		TaskName    string
		Search      *string
		Status      []string
	}

	clientJobTaskSubscriber struct {
		c      chan JobListResolver
		filter Filter
	}

	jobStatusEnum string
)

const (
	statusRetrying jobStatusEnum = "RETRYING"
	statusFailure  jobStatusEnum = "FAILURE"
	statusSuccess  jobStatusEnum = "SUCCESS"
	statusQueueing jobStatusEnum = "QUEUEING"
	statusStopped  jobStatusEnum = "STOPPED"
)

var (
	registeredTask map[string]struct {
		handler     types.WorkerHandler
		workerIndex int
	}

	workers         []reflect.SelectCase
	workerIndexTask map[int]*struct {
		taskName       string
		activeInterval *time.Ticker
	}

	queue                        QueueStorage
	repo                         *storage
	refreshWorkerNotif, shutdown chan struct{}
	semaphore                    []chan struct{}
	mutex                        sync.Mutex
	tasks                        []string

	clientTaskSubscribers    map[string]chan TaskListResolver
	clientJobTaskSubscribers map[string]clientJobTaskSubscriber

	errClientLimitExceeded = errors.New("client limit exceeded, please try again later")

	defaultOption option
)

func makeAllGlobalVars(q QueueStorage, db *mongo.Database, opts ...OptionFunc) {
	createMongoIndex(db)

	queue = q
	repo = &storage{db: db}

	if env.BaseEnv().JaegerTracingDashboard != "" {
		defaultOption.JaegerTracingDashboard = env.BaseEnv().JaegerTracingDashboard
	} else if urlTracerAgent, _ := url.Parse("//" + env.BaseEnv().JaegerTracingHost); urlTracerAgent != nil {
		defaultOption.JaegerTracingDashboard = urlTracerAgent.Hostname()
	}
	defaultOption.MaxClientSubscriber = env.BaseEnv().TaskQueueDashboardMaxClientSubscribers
	defaultOption.AutoRemoveClientInterval = 30 * time.Minute

	for _, opt := range opts {
		opt(&defaultOption)
	}

	refreshWorkerNotif, shutdown = make(chan struct{}), make(chan struct{}, 1)
	clientTaskSubscribers = make(map[string]chan TaskListResolver, defaultOption.MaxClientSubscriber)
	clientJobTaskSubscribers = make(map[string]clientJobTaskSubscriber, defaultOption.MaxClientSubscriber)

	registeredTask = make(map[string]struct {
		handler     types.WorkerHandler
		workerIndex int
	})
	workerIndexTask = make(map[int]*struct {
		taskName       string
		activeInterval *time.Ticker
	})

	// add refresh worker channel to first index
	workers = append(workers, reflect.SelectCase{
		Dir: reflect.SelectRecv, Chan: reflect.ValueOf(refreshWorkerNotif),
	})
}
