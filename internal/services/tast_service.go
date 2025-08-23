package services

import (
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

}

func (service *TaskService) List() ([]*domain.Task, error) {

}

func (service *TaskService) Get(id int64) (*domain.Task, error) {

}
