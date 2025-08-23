package services

import (
	"fmt"

	"task-service/internal/domain"
	"task-service/pkg/logging"
)

type TaskService struct {
	logger *logging.Logger
}

func NewTaskService(logger *logging.Logger) *TaskService {
	return &TaskService{
		logger: logger,
	}
}

func (service *TaskService) Create(task *domain.Task) (int64, error) {
	fmt.Println("ok")
	return 0, nil
}

func (service *TaskService) List() ([]*domain.Task, error) {
	fmt.Println("ok")
	return nil, nil
}

func (service *TaskService) Get(id int64) (*domain.Task, error) {
	fmt.Println("ok")
	return &domain.Task{}, nil
}
