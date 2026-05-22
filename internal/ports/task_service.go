package ports

import "task-service/internal/domain"

type TaskService interface {
	TaskQueries
	TaskCommands
}

type TaskQueries interface {
	List(limit, offset uint64) ([]*domain.Task, uint64, error)
	Get(id uint64) (*domain.Task, error)
}

type TaskCommands interface {
	Create(task *domain.Task) (uint64, error)
}
