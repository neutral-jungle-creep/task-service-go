package root

import (
	"context"
	"fmt"

	"task-service/internal/server"
	"task-service/pkg/http_server"
)

const (
	defaultRouteGroup = "/api/v1/task-service"
)

func (r *Root) initHttpServer() {
	apiImplementation := server.NewApi(r.services.taskService)

	s := http_server.NewServer(
		apiImplementation.InitRoutes(defaultRouteGroup),
		http_server.Port(r.config.HTTPServer.ListenPort),
		http_server.IdleTimeout(r.config.HTTPServer.KeepAliveTime+r.config.HTTPServer.KeepAliveTimeout),
		http_server.ReadHeaderTimeout(r.config.HTTPServer.ReadHeaderTimeout),
		http_server.ReadTimeout(r.config.HTTPServer.ReadTimeout),
	)

	r.RegisterStopHandler(func() { _ = s.Shutdown(r.ctx) })

	r.RegisterBackgroundJob(func() error {
		r.logger.InfoWithCtx(r.ctx, fmt.Sprintf("starting HTTP server on addr %s", r.config.HTTPServer.ListenPort))
		return s.ListenAndServe()
	})
}
