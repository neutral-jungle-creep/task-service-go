package server

import (
	"task-service/internal/domain"
	"task-service/internal/server/dto"
)

func tasksFromDomain(tasks []*domain.Task) []*dto.GetTaskResponse {
	taskResponses := make([]*dto.GetTaskResponse, 0, len(tasks))
	for _, task := range tasks {
		taskResponses = append(taskResponses, taskFromDomain(task))
	}
	return taskResponses
}

func taskFromDomain(task *domain.Task) *dto.GetTaskResponse {
	return &dto.GetTaskResponse{
		ID:        task.ID,
		Name:      task.Name,
		Body:      task.Body,
		Status:    task.Status.String(),
		CreatedAt: task.CreatedAt,
		UpdatedAt: task.UpdatedAt,
	}
}
