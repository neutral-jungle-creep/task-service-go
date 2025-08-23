package domain

import "time"

type Task struct {
	ID        int64
	Name      string
	Body      string
	Status    TaskStatus
	CreatedAt time.Time
	UpdatedAt *time.Time
}

func NewTask(name, body string) *Task {
	return &Task{
		Name:      name,
		Body:      body,
		Status:    TaskStatusNew,
		CreatedAt: time.Now(),
	}
}

type TaskStatus string

func (s TaskStatus) String() string {
	return string(s)
}

const (
	TaskStatusNew       TaskStatus = "NEW"
	TaskStatusInProcess TaskStatus = "IN_PROCESS"
	TaskStatusPause     TaskStatus = "PAUSE"
	TaskStatusComplete  TaskStatus = "COMPLETE"
	TaskStatusCancel    TaskStatus = "CANCEL"
)
