package restserver

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"sort"
	"strings"

	"github.com/labstack/echo"
	echoMidd "github.com/labstack/echo/middleware"
	"github.com/soheilhy/cmux"

	"github.com/golangid/candi/candihelper"
	"github.com/golangid/candi/candishared"
	graphqlserver "github.com/golangid/candi/codebase/app/graphql_server"
	"github.com/golangid/candi/codebase/factory"
	"github.com/golangid/candi/codebase/factory/types"
	"github.com/golangid/candi/config/env"
	"github.com/golangid/candi/logger"
	"github.com/golangid/candi/wrapper"
)

type restServer struct {
	serverEngine *echo.Echo
	service      factory.ServiceFactory
	httpPort     string
	listener     net.Listener
}

// NewServer create new REST server
func NewServer(service factory.ServiceFactory, muxListener cmux.CMux) factory.AppServerFactory {
	server := &restServer{
		serverEngine: echo.New(),
		service:      service,
		httpPort:     fmt.Sprintf(":%d", env.BaseEnv().HTTPPort),
	}

	if muxListener != nil {
		server.listener = muxListener.Match(cmux.HTTP1Fast())
	}

	server.serverEngine.HTTPErrorHandler = wrapper.CustomHTTPErrorHandler
	server.serverEngine.Use(echoMidd.CORS())

	server.serverEngine.GET("/", echo.WrapHandler(http.HandlerFunc(candishared.HTTPRoot(string(service.Name())))))
	server.serverEngine.GET("/memstats",
		echo.WrapHandler(http.HandlerFunc(candishared.HTTPMemstatsHandler)),
		echo.WrapMiddleware(service.GetDependency().GetMiddleware().HTTPBasicAuth))

	restRootPath := server.serverEngine.Group("", echoRestTracerMiddleware)
	if env.BaseEnv().DebugMode {
		restRootPath.Use(echoMidd.Logger())
	}
	for _, m := range service.GetModules() {
		if h := m.RESTHandler(); h != nil {
			h.Mount(restRootPath)
		}
	}

	httpRoutes := server.serverEngine.Routes()
	sort.Slice(httpRoutes, func(i, j int) bool {
		return httpRoutes[i].Path < httpRoutes[j].Path
	})
	for _, route := range httpRoutes {
		if !candihelper.StringInSlice(route.Path, []string{"/", "/memstats"}) && !strings.Contains(route.Name, "(*Group)") {
			logger.LogGreen(fmt.Sprintf("[REST-ROUTE] %-6s %-30s --> %s", route.Method, route.Path, route.Name))
		}
	}

	// inject graphql handler to rest server
	if env.BaseEnv().UseGraphQL {
		graphqlHandler := graphqlserver.NewHandler(service)
		server.serverEngine.Any("/graphql", echo.WrapHandler(graphqlHandler.ServeGraphQL()))
		server.serverEngine.GET("/graphql/playground", echo.WrapHandler(http.HandlerFunc(graphqlHandler.ServePlayground)))
		server.serverEngine.GET("/graphql/voyager", echo.WrapHandler(http.HandlerFunc(graphqlHandler.ServeVoyager)))

		logger.LogYellow("[GraphQL] endpoint : /graphql")
		logger.LogYellow("[GraphQL] playground : /graphql/playground")
		logger.LogYellow("[GraphQL] voyager : /graphql/voyager")
	}

	fmt.Printf("\x1b[34;1m⇨ HTTP server run at port [::]%s\x1b[0m\n\n", server.httpPort)

	return server
}

func (h *restServer) Serve() {

	h.serverEngine.HideBanner = true
	h.serverEngine.HidePort = true

	var err error
	if h.listener != nil {
		h.serverEngine.Listener = h.listener
		err = h.serverEngine.Start("")
	} else {
		err = h.serverEngine.Start(h.httpPort)
	}

	switch e := err.(type) {
	case *net.OpError:
		panic(fmt.Errorf("rest server: %v", e))
	}
}

func (h *restServer) Shutdown(ctx context.Context) {
	log.Println("\x1b[33;1mStopping HTTP server...\x1b[0m")
	defer func() { log.Println("\x1b[33;1mStopping HTTP server:\x1b[0m \x1b[32;1mSUCCESS\x1b[0m") }()

	h.serverEngine.Shutdown(ctx)
	if h.listener != nil {
		h.listener.Close()
	}
}

func (h *restServer) Name() string {
	return string(types.REST)
}
