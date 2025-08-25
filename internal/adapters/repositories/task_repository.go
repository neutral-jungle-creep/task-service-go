package repositories

import (
	"sync/atomic"

	"task-service/internal/domain"
	"task-service/internal/ports"
)

type TaskRepository struct {
	taskIdSequence atomic.Uint64
}

func NewTaskRepository() *TaskRepository {
	return &TaskRepository{}
}

func (r *TaskRepository) Store(task *domain.Task) (uint64, error) {
	return r.autoIncrementID(), nil
}

func (r *TaskRepository) List(filter *ports.ListTasksFilter) ([]*domain.Task, error) {
	return []*domain.Task{}, nil
}

func (r *TaskRepository) Get(id uint64) (*domain.Task, error) {
	return &domain.Task{}, nil
}

// это заглушка вместо автоинкремента базы данных чтобы при создании тасок генерировать новые id
func (r *TaskRepository) autoIncrementID() uint64 {
	return r.taskIdSequence.Add(1)
}
