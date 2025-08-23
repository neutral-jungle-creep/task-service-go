package root

import (
	"context"
	"sync"

	"task-service/internal/config"
	"task-service/internal/ports"
	"task-service/pkg/logging"
)

type Root struct {
	ctx      context.Context
	config   *config.Config
	logger   *logging.Logger
	services struct {
		taskService ports.TaskService
		taskCache   ports.TaskCache
	}
	repositories struct {
		taskRepository ports.TaskRepository
	}

	startupTasks   []func() error
	backgroundJobs []func() error
	stopHandlers   []func()
}

func New(ctx context.Context, config *config.Config, logger *logging.Logger) (*Root, error) {
	root := Root{
		ctx:    ctx,
		config: config,
		logger: logger,
	}

	root.initRepositories()
	root.initServices()
	root.initHttpServer()

	return &root, nil
}

func (r *Root) Run() error {
	defer r.stop()

	errors := r.startBackgroundJobs()

	select {
	case <-r.ctx.Done():
		r.logger.Warn("stopping application, context was cancelled")
		return nil
	case err := <-errors:
		return err
	}
}

func (r *Root) RegisterBackgroundJob(backgroundJob func() error) {
	r.backgroundJobs = append(r.backgroundJobs, backgroundJob)
}

func (r *Root) RegisterStopHandler(stopHandler func()) {
	r.stopHandlers = append(r.stopHandlers, stopHandler)
}

func (r *Root) startBackgroundJobs() chan error {
	errors := make(chan error)

	for _, job := range r.backgroundJobs {
		go func() {
			errors <- job()
		}()
	}

	return errors
}

func (r *Root) stop() {
	var wg sync.WaitGroup
	wg.Add(len(r.stopHandlers))
	for _, handler := range r.stopHandlers {
		go func() {
			defer wg.Done()
			handler()
		}()
	}
	wg.Wait()
}
