package root

import (
	"context"

	"task-service/internal/config"
	"task-service/internal/ports"
	"task-service/pkg/background"
	"task-service/pkg/logging"
)

type Root struct {
	ctx      context.Context
	config   *config.Config
	logger   *logging.AsyncLogger
	services struct {
		taskService ports.TaskService
		taskCache   ports.TaskCache
	}
	repositories struct {
		taskRepository ports.TaskRepository
	}

	background *background.Registrar
}

func New(ctx context.Context, config *config.Config, logger *logging.Logger) (*Root, error) {
	root := Root{
		ctx:        ctx,
		config:     config,
		background: background.New(),
	}

	root.initObservability(logger)

	if err := root.initRepositories(); err != nil {
		return nil, err
	}

	if err := root.initServices(); err != nil {
		return nil, err
	}

	root.initHTTPServer()

	return &root, nil
}

func (r *Root) Run() error {
	defer r.background.Stop()

	err := r.background.Run(r.ctx)
	if err == nil {
		r.logger.Warn("stopping application, context was cancelled")
	}
	return err
}

// RegisterBackgroundJob and RegisterStopHandler stay on *Root so that the
// existing init* helpers (observability, repository, services, http_server)
// can keep calling r.Register*. Internally they delegate to background.Registrar.

func (r *Root) RegisterBackgroundJob(job func() error) {
	r.background.RegisterJob(job)
}

func (r *Root) RegisterStopHandler(h func()) {
	r.background.RegisterStopHandler(h)
}
