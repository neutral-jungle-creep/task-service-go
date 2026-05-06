package root

import (
	api "task-service/internal/server"
	server "task-service/pkg/http/server"
)

func (r *Root) initHTTPServer() {
	apiImplementation := api.NewAPI(r.services.taskService)

	s := server.NewServer(
		apiImplementation.InitRoutes(r.config.RouteGroup),
		server.Port(r.config.HTTPServer.ListenPort),
		server.IdleTimeout(r.config.HTTPServer.KeepAliveTime+r.config.HTTPServer.KeepAliveTimeout),
		server.ReadHeaderTimeout(r.config.HTTPServer.ReadHeaderTimeout),
		server.ReadTimeout(r.config.HTTPServer.ReadTimeout),
	)

	r.RegisterStopHandler(func() { _ = s.Shutdown(r.ctx) })

	r.RegisterBackgroundJob(func() error {
		r.logger.Info("starting HTTP server on addr " + r.config.HTTPServer.ListenPort)
		return s.ListenAndServe()
	})
}
