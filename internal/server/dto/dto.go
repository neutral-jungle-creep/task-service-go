package dto

import "time"

type ListTasksResponse struct {
	Items  []*GetTaskResponse `json:"items"`
	Total  uint64             `json:"total"`
	Limit  uint64             `json:"limit"`
	Offset uint64             `json:"offset"`
}

type ListTasksQuery struct {
	Limit  uint64 `validate:"min=1,max=500"`
	Offset uint64 `validate:"gte=0"`
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
	Name string `json:"name" validate:"required,min=1,max=255"`
	Body string `json:"body" validate:"required,min=1,max=10000"`
}

type CreateTaskResponse struct {
	ID uint64 `json:"id"`
}
