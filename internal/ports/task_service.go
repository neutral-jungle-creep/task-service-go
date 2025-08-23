package ports

import "task-service/internal/domain"

type TaskService interface {
	TaskQueries
	TaskCommands
}

type TaskQueries interface {
	List() ([]*domain.Task, error)
}

type TaskCommands interface {
	Create(task *domain.Task) (int64, error)
}
