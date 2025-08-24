package ports

import "task-service/internal/domain"

type TaskCache interface {
	Store(task *domain.Task)
	List() ([]*domain.Task, uint64)
	Get(id uint64) (*domain.Task, bool)
}
