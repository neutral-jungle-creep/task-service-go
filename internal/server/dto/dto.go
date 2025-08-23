package dto

type ListTasksResponse struct {
	Items GetTaskResponse `json:"items"`
	Total int64           `json:"total"`
}

type GetTaskResponse struct {
}

type CreateTaskRequest struct {
}
