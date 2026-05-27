package root

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	api "task-service/internal/server"
	"task-service/pkg/http/middleware"
	"task-service/pkg/http/protocol"
	"task-service/pkg/http/server"
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

	// Order matters: rate limit is the outermost gate (cheap reject before any
	// further work); body limit runs after, before handlers touch r.Body.
	handler := middleware.IPRateLimit(middleware.IPRateLimitConfig{
		Rate:  r.config.HTTPServer.IPRateLimit,
		Burst: r.config.HTTPServer.IPRateBurst,
		TTL:   r.config.HTTPServer.IPRateLimiterTTL,
	})(mux)
	handler = middleware.MaxBodyBytes(r.config.HTTPServer.MaxRequestBodyBytes)(handler)

	s := server.NewServer(
		handler,
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
