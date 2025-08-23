package dto

import "time"

type ListTasksResponse struct {
	Items []*GetTaskResponse `json:"items"`
	Total int64              `json:"total"`
}

type GetTaskResponse struct {
	ID        int64      `json:"id"`
	Name      string     `json:"name"`
	Body      string     `json:"body"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt *time.Time `json:"updatedAt"`
}

type CreateTaskRequest struct {
	Name string `json:"name" binding:"required"`
	Body string `json:"body" binding:"required"`
}

type CreateTaskResponse struct {
	ID int64 `json:"id"`
}
