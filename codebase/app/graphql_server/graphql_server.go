package graphqlserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"reflect"
	"runtime"

	"github.com/golangid/candi/candihelper"
	"github.com/golangid/candi/candishared"
	"github.com/golangid/candi/codebase/app/graphql_server/static"
	"github.com/golangid/candi/codebase/app/graphql_server/ws"
	"github.com/golangid/candi/codebase/factory"
	"github.com/golangid/candi/codebase/factory/types"
	"github.com/golangid/candi/config/env"
	"github.com/golangid/candi/logger"
	"github.com/golangid/candi/tracer"

	graphql "github.com/golangid/graphql-go"
	"github.com/soheilhy/cmux"
)

const (
	rootGraphQLPath       = "/graphql"
	rootGraphQLPlayground = "/graphql/playground"
	rootGraphQLVoyager    = "/graphql/voyager"
)

type graphqlServer struct {
	httpEngine *http.Server
	listener   net.Listener
}

// NewServer create new GraphQL server
func NewServer(service factory.ServiceFactory, muxListener cmux.CMux) factory.AppServerFactory {

	httpEngine := new(http.Server)
	httpHandler := NewHandler(service)

	mux := http.NewServeMux()
	mux.HandleFunc("/", candishared.HTTPRoot(string(service.Name())))
	mux.HandleFunc("/memstats", candishared.HTTPMemstatsHandler)
	mux.HandleFunc(rootGraphQLPath, httpHandler.ServeGraphQL())
	mux.HandleFunc(rootGraphQLPlayground, httpHandler.ServePlayground)
	mux.HandleFunc(rootGraphQLVoyager, httpHandler.ServeVoyager)

	httpEngine.Addr = fmt.Sprintf(":%d", env.BaseEnv().HTTPPort)
	httpEngine.Handler = mux

	logger.LogYellow("[GraphQL] endpoint : " + rootGraphQLPath)
	logger.LogYellow("[GraphQL] playground : " + rootGraphQLPlayground)
	logger.LogYellow("[GraphQL] voyager : " + rootGraphQLVoyager)
	fmt.Printf("\x1b[34;1m⇨ GraphQL HTTP server run at port [::]%s\x1b[0m\n\n", httpEngine.Addr)

	server := &graphqlServer{
		httpEngine: httpEngine,
	}

	if muxListener != nil {
		server.listener = muxListener.Match(cmux.HTTP1Fast())
	}

	return server
}

func (s *graphqlServer) Serve() {
	var err error
	if s.listener != nil {
		err = s.httpEngine.Serve(s.listener)
	} else {
		err = s.httpEngine.ListenAndServe()
	}

	switch e := err.(type) {
	case *net.OpError:
		panic(fmt.Errorf("gql http server: %v", e))
	}
}

func (s *graphqlServer) Shutdown(ctx context.Context) {
	log.Println("\x1b[33;1mStopping GraphQL HTTP server...\x1b[0m")
	defer func() { log.Println("\x1b[33;1mStopping GraphQL HTTP server:\x1b[0m \x1b[32;1mSUCCESS\x1b[0m") }()

	s.httpEngine.Shutdown(ctx)
	if s.listener != nil {
		s.listener.Close()
	}
}

func (s *graphqlServer) Name() string {
	return string(types.GraphQL)
}

// Handler interface
type Handler interface {
	ServeGraphQL() http.HandlerFunc
	ServePlayground(resp http.ResponseWriter, req *http.Request)
	ServeVoyager(resp http.ResponseWriter, req *http.Request)
}

// NewHandler for create public graphql handler (maybe inject to rest handler)
func NewHandler(service factory.ServiceFactory) Handler {

	// create dynamic struct
	queryResolverValues := make(map[string]interface{})
	mutationResolverValues := make(map[string]interface{})
	subscriptionResolverValues := make(map[string]interface{})
	middlewareResolvers := make(types.MiddlewareGroup)
	var queryResolverFields, mutationResolverFields, subscriptionResolverFields []reflect.StructField
	for _, m := range service.GetModules() {
		if resolverModule := m.GraphQLHandler(); resolverModule != nil {
			rootName := string(m.Name())
			resolverModule.RegisterMiddleware(&middlewareResolvers)
			query, mutation, subscription := resolverModule.Query(), resolverModule.Mutation(), resolverModule.Subscription()

			appendStructField(rootName, query, &queryResolverFields)
			appendStructField(rootName, mutation, &mutationResolverFields)
			appendStructField(rootName, subscription, &subscriptionResolverFields)

			queryResolverValues[rootName] = query
			mutationResolverValues[rootName] = mutation
			subscriptionResolverValues[rootName] = subscription
		}
	}

	root.rootQuery = constructStruct(queryResolverFields, queryResolverValues)
	root.rootMutation = constructStruct(mutationResolverFields, mutationResolverValues)
	root.rootSubscription = constructStruct(subscriptionResolverFields, subscriptionResolverValues)
	gqlSchema := candihelper.LoadAllFile(os.Getenv(candihelper.WORKDIR)+"api/graphql", ".graphql")

	schemaOpts := []graphql.SchemaOpt{
		graphql.UseStringDescriptions(),
		graphql.UseFieldResolvers(),
		graphql.Tracer(newGraphQLMiddleware(middlewareResolvers)),
		graphql.Logger(&panicLogger{}),
	}
	if env.BaseEnv().IsProduction {
		// handling vulnerabilities exploit schema
		schemaOpts = append(schemaOpts, graphql.DisableIntrospection())
	}
	schema := graphql.MustParseSchema(string(gqlSchema), &root, schemaOpts...)

	return &handlerImpl{
		schema: schema,
	}
}

type handlerImpl struct {
	schema *graphql.Schema
}

func (s *handlerImpl) ServeGraphQL() http.HandlerFunc {

	return ws.NewHandlerFunc(s.schema, http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {

		var params struct {
			Query         string                 `json:"query"`
			OperationName string                 `json:"operationName"`
			Variables     map[string]interface{} `json:"variables"`
		}
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			http.Error(resp, err.Error(), http.StatusBadRequest)
			return
		}
		if err := json.Unmarshal(body, &params); err != nil {
			params.Query = string(body)
		}

		ip := req.Header.Get("X-Forwarded-For")
		if ip == "" {
			ip = req.Header.Get("X-Real-IP")
			if ip == "" {
				ip, _, _ = net.SplitHostPort(req.RemoteAddr)
			}
		}
		req.Header.Set("X-Real-IP", ip)

		ctx := context.WithValue(req.Context(), candishared.ContextKeyHTTPHeader, req.Header)
		response := s.schema.Exec(ctx, params.Query, params.OperationName, params.Variables)
		responseJSON, err := json.Marshal(response)
		if err != nil {
			http.Error(resp, err.Error(), http.StatusInternalServerError)
			return
		}

		resp.Header().Set("Content-Type", "application/json")
		resp.Write(responseJSON)
	}))
}

func (s *handlerImpl) ServePlayground(resp http.ResponseWriter, req *http.Request) {
	if env.BaseEnv().IsProduction {
		http.Error(resp, "Forbidden", http.StatusForbidden)
		return
	}
	resp.Write([]byte(static.PlaygroundAsset))
}

func (s *handlerImpl) ServeVoyager(resp http.ResponseWriter, req *http.Request) {
	if env.BaseEnv().IsProduction {
		http.Error(resp, "Forbidden", http.StatusForbidden)
		return
	}
	resp.Write([]byte(static.VoyagerAsset))
}

// panicLogger is the default logger used to log panics that occur during query execution
type panicLogger struct{}

// LogPanic is used to log recovered panic values that occur during query execution
// https://github.com/graph-gophers/graphql-go/blob/master/log/log.go#L19 + custom add log to trace
func (l *panicLogger) LogPanic(ctx context.Context, value interface{}) {
	const size = 64 << 10
	buf := make([]byte, size)
	buf = buf[:runtime.Stack(buf, false)]

	tracer.Log(ctx, "gql_panic", value)
	tracer.Log(ctx, "gql_panic_trace", buf)
}
