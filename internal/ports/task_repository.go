package ports

import (
	"context"

	"task-service/internal/domain"
)

type TaskRepository interface {
	Store(ctx context.Context, task *domain.Task) (uint64, error)
	List(ctx context.Context, filter *ListTasksFilter) ([]*domain.Task, error)
	Count(ctx context.Context, filter *ListTasksFilter) (uint64, error)
	Get(ctx context.Context, id uint64) (*domain.Task, error)
	Update(ctx context.Context, task *domain.Task) error
	Delete(ctx context.Context, id uint64) error
}

const (
	SortDesc = "desc"
	SortAsc  = "asc"
)

// ListTasksFilter narrows down what List/Count return. Zero Limit means
// "use the repository default".
type ListTasksFilter struct {
	Sort   string
	ToID   uint64
	Limit  uint64
	Offset uint64
}
