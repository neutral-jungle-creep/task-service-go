package root

import (
	"task-service/internal/services"
)

func (r *Root) initServices() {
	r.services.taskService = services.NewTaskService(r.logger)
}
