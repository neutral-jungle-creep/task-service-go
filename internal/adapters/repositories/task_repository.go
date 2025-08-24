package repositories

import (
	"task-service/internal/domain"
	"task-service/internal/ports"
)

type TaskRepository struct {
}

func NewTaskRepository() *TaskRepository {
	return &TaskRepository{}
}

func (r *TaskRepository) Store(task *domain.Task) (uint64, error) {
	return 0, nil
}

func (r *TaskRepository) List(filter *ports.ListTasksFilter) ([]*domain.Task, error) {
	return []*domain.Task{}, nil

}

func (r *TaskRepository) Get(id uint64) (*domain.Task, error) {
	return &domain.Task{}, nil
}
