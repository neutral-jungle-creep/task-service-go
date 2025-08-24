package root

import (
	"task-service/internal/services"
)

func (r *Root) initServices() error {
	var err error

	r.services.taskCache, err = services.NewTaskCache(
		r.config.Cache.MemoryCacheLimitMB,
		r.config.Cache.MemoryMonitorCacheInterval,
		r.repositories.taskRepository,
	)
	if err != nil {
		return err
	}

	logChan := make(chan []byte)

	r.services.taskService = services.NewTaskService(
		r.logger,
		r.repositories.taskRepository,
		r.services.taskCache,
	)

	logsProcessor, err := services.NewLogProcessor(
		r.ctx,
		logChan,
		r.config.Logger.FileName,
		r.logger,
	)

	r.RegisterBackgroundJob(func() error {
		return logsProcessor.Process()
	})
	r.RegisterStopHandler(func() {
		logsProcessor.Stop()
	})
	return nil
}
