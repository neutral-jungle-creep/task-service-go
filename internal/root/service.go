package root

import (
	"task-service/internal/services"
)

func (r *Root) initServices() {
	r.services.taskCache = services.NewTaskCache(
		r.config.MemoryCacheLimitMB,
		r.config.MemoryMonitorCacheInterval,
	)

	r.services.taskService = services.NewTaskService(
		r.logger,
		r.repositories.taskRepository,
		r.services.taskCache,
	)
}
