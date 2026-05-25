package root

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	api "task-service/internal/server"
	"task-service/pkg/http/protocol"
	server "task-service/pkg/http/server"
)

func (r *Root) initHTTPServer() {
	validate := validator.New(validator.WithRequiredStructEnabled())
	responseHandler := protocol.NewResponseHandler(r.logger, protocol.WithValidation(validate))

	apiImplementation := api.NewAPI(responseHandler, r.services.taskService)

	mux := http.NewServeMux()
	mux.Handle("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))
	mux.Handle("/", apiImplementation.InitRoutes(r.config.RouteGroup))

	s := server.NewServer(
		mux,
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
