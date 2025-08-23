package root

import (
	"task-service/internal/adapters/repositories"
)

func (r *Root) initRepositories() {
	r.repositories.taskRepository = repositories.NewTaskRepository()
}
