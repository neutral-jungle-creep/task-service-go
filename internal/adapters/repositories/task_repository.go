package repositories

import "task-service/internal/domain"

type TaskRepository struct {
}

func NewTaskRepository() *TaskRepository {
	return &TaskRepository{}
}

func (r *TaskRepository) Store(task *domain.Task) (int64, error) {
	return 0, nil
}

func (r *TaskRepository) List() ([]*domain.Task, error) {
	return []*domain.Task{}, nil

}

func (r *TaskRepository) Get(id int64) (*domain.Task, error) {
	return &domain.Task{}, nil
}
