package root

import (
	"fmt"

	api "task-service/internal/server"
	server "task-service/pkg/http/server"
)

const (
	defaultRouteGroup = "/api/v1/task-service"
)

func (r *Root) initHttpServer() {
	apiImplementation := api.NewApi(r.services.taskService)

	s := server.NewServer(
		apiImplementation.InitRoutes(defaultRouteGroup),
		server.Port(r.config.HTTPServer.ListenPort),
		server.IdleTimeout(r.config.HTTPServer.KeepAliveTime+r.config.HTTPServer.KeepAliveTimeout),
		server.ReadHeaderTimeout(r.config.HTTPServer.ReadHeaderTimeout),
		server.ReadTimeout(r.config.HTTPServer.ReadTimeout),
	)

	r.RegisterStopHandler(func() { _ = s.Shutdown(r.ctx) })

	r.RegisterBackgroundJob(func() error {
		r.logger.Info(fmt.Sprintf("starting HTTP server on addr %s", r.config.HTTPServer.ListenPort))
		return s.ListenAndServe()
	})
}
