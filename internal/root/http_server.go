package root

import (
	"fmt"

	"task-service/internal/server"
	server2 "task-service/pkg/http/server"
)

const (
	defaultRouteGroup = "/api/v1/task-service"
)

func (r *Root) initHttpServer() {
	apiImplementation := server.NewApi(r.services.taskService)

	s := server2.NewServer(
		apiImplementation.InitRoutes(defaultRouteGroup),
		server2.Port(r.config.HTTPServer.ListenPort),
		server2.IdleTimeout(r.config.HTTPServer.KeepAliveTime+r.config.HTTPServer.KeepAliveTimeout),
		server2.ReadHeaderTimeout(r.config.HTTPServer.ReadHeaderTimeout),
		server2.ReadTimeout(r.config.HTTPServer.ReadTimeout),
	)

	r.RegisterStopHandler(func() { _ = s.Shutdown(r.ctx) })

	r.RegisterBackgroundJob(func() error {
		r.logger.Info(fmt.Sprintf("starting HTTP server on addr %s", r.config.HTTPServer.ListenPort))
		return s.ListenAndServe()
	})
}
