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

type UpdateTaskParams struct {
	Name   *string
	Body   *string
	Status *string
}

type TaskCommands interface {
	Create(task *domain.Task) (uint64, error)
	Update(id uint64, params UpdateTaskParams) (*domain.Task, error)
	Delete(id uint64) error
}
