package taskqueueworker

import (
	"errors"
	"net/url"
	"reflect"
	"sync"
	"time"

	"github.com/golangid/candi/codebase/factory/types"
	"github.com/golangid/candi/config/env"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type (
	// TaglineResolver resolver
	TaglineResolver struct {
		Banner                    string
		Tagline                   string
		Version                   string
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
		Page                  int
		Limit                 int
		TotalRecords          int
		TotalPages            int
		IsCloseSession        bool
		TotalClientSubscriber int
	}
	// TaskResolver resolver
	TaskResolver struct {
		Name      string
		TotalJobs int
		Detail    struct {
			Failure, Retrying, Success, Queueing, Stopped int
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
			Failure, Retrying, Success, Queueing, Stopped int
		}
	}

	// JobListResolver resolver
	JobListResolver struct {
		Meta MetaJobList
		Data []Job
	}

	// Filter type
	Filter struct {
		Page, Limit  int
		TaskName     string
		TaskNameList []string
		Search       *string
		Status       []string
		ShowAll      bool
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

	queue                                             QueueStorage
	repo                                              *storage
	refreshWorkerNotif, shutdown, closeAllSubscribers chan struct{}
	semaphore                                         []chan struct{}
	mutex                                             sync.Mutex
	tasks                                             []string

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
	defaultOption.DashboardBanner = `
    _________    _   ______  ____
   / ____/   |  / | / / __ \/  _/
  / /   / /| | /  |/ / / / // /  
 / /___/ ___ |/ /|  / /_/ // /   
 \____/_/  |_/_/ |_/_____/___/   `

	for _, opt := range opts {
		opt(&defaultOption)
	}

	refreshWorkerNotif, shutdown, closeAllSubscribers = make(chan struct{}), make(chan struct{}, 1), make(chan struct{})
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

func (f *Filter) toBsonFilter() bson.M {
	pipeQuery := []bson.M{}

	if f.TaskName != "" {
		pipeQuery = append(pipeQuery, bson.M{
			"task_name": f.TaskName,
		})
	} else if len(f.TaskNameList) > 0 {
		pipeQuery = append(pipeQuery, bson.M{
			"task_name": bson.M{
				"$in": f.TaskNameList,
			},
		})
	}

	if f.Search != nil && *f.Search != "" {
		pipeQuery = append(pipeQuery, bson.M{
			"arguments": primitive.Regex{Pattern: *f.Search, Options: "i"},
		})
	}
	if len(f.Status) > 0 {
		pipeQuery = append(pipeQuery, bson.M{
			"status": bson.M{
				"$in": f.Status,
			},
		})
	}

	return bson.M{
		"$and": pipeQuery,
	}
}
