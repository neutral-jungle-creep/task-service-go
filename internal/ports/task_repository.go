package ports

import "task-service/internal/domain"

type TaskRepository interface {
	Store(task *domain.Task) (int64, error)
	List() ([]*domain.Task, error)
	Get(id int64) (*domain.Task, error)
}
