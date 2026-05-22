package ports

import "task-service/internal/domain"

type TaskRepository interface {
	Store(task *domain.Task) (uint64, error)
	List(filter *ListTasksFilter) ([]*domain.Task, error)
	Count(filter *ListTasksFilter) (uint64, error)
	Get(id uint64) (*domain.Task, error)
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
