package services

import "task-service/pkg/logging"

type TaskService struct {
	logger *logging.Logger
}

func NewTaskService(logger *logging.Logger) *TaskService {
	return &TaskService{
		logger: logger,
	}
}
