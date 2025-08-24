package ports

import "task-service/internal/domain"

type TaskRepository interface {
	Store(task *domain.Task) (uint64, error)
	List(filter *ListTasksFilter) ([]*domain.Task, error)
	Get(id uint64) (*domain.Task, error)
}

const (
	SortDesc = "desc"
	SortAsc  = "asc"
)

type ListTasksFilter struct {
	Sort string
	ToId uint64
}
