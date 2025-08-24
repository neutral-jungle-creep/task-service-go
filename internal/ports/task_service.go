package ports

import "task-service/internal/domain"

type TaskService interface {
	TaskQueries
	TaskCommands
}

type TaskQueries interface {
	List() ([]*domain.Task, error)
	Get(id uint64) (*domain.Task, error)
}

type TaskCommands interface {
	Create(task *domain.Task) (uint64, error)
}
