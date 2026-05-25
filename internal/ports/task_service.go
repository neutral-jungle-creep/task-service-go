package ports

import (
	"context"

	"task-service/internal/domain"
)

type TaskService interface {
	TaskQueries
	TaskCommands
}

type TaskQueries interface {
	List(ctx context.Context, limit, offset uint64) ([]*domain.Task, uint64, error)
	Get(ctx context.Context, id uint64) (*domain.Task, error)
}

type UpdateTaskParams struct {
	Name   *string
	Body   *string
	Status *string
}

type TaskCommands interface {
	Create(ctx context.Context, task *domain.Task) (uint64, error)
	Update(ctx context.Context, id uint64, params UpdateTaskParams) (*domain.Task, error)
	Delete(ctx context.Context, id uint64) error
}
