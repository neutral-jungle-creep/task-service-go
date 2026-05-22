package dto

import "time"

type ListTasksResponse struct {
	Items  []*GetTaskResponse `json:"items"`
	Total  uint64             `json:"total"`
	Limit  uint64             `json:"limit"`
	Offset uint64             `json:"offset"`
}

type GetTaskResponse struct {
	ID        uint64     `json:"id"`
	Name      string     `json:"name"`
	Body      string     `json:"body"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt *time.Time `json:"updatedAt"`
}

type CreateTaskRequest struct {
	Name string `json:"name"`
	Body string `json:"body"`
}

type CreateTaskResponse struct {
	ID uint64 `json:"id"`
}
