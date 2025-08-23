package ports

import "task-service/internal/domain"

type TaskCache interface {
	Store(task *domain.Task)
	List() []*domain.Task
	Get(id int64) (*domain.Task, bool)
}
