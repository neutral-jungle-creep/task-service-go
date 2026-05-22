package root

import (
	"task-service/internal/services"
)

func (r *Root) initServices() error {
	taskCache, err := services.NewTaskCache(
		r.config.Cache.MemoryCacheLimitMB,
		r.config.Cache.MemoryMonitorCacheInterval,
		r.repositories.taskRepository,
	)
	if err != nil {
		return err
	}
	r.services.taskCache = taskCache

	r.RegisterBackgroundJob(func() error { return taskCache.Run(r.ctx) })

	r.services.taskService = services.NewTaskService(
		r.logger,
		r.repositories.taskRepository,
		r.services.taskCache,
	)
	return nil
}
