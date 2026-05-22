package domain

import (
	"errors"
	"time"
	"unsafe"
)

var ErrTaskNotFound = errors.New("task not found")

type Task struct {
	ID        uint64
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

func (t *Task) Size() uint64 {
	size := uint64(unsafe.Sizeof(*t))
	size += uint64(len(t.Name))
	size += uint64(len(t.Body))
	size += uint64(len(t.Status))
	if t.UpdatedAt != nil {
		size += uint64(unsafe.Sizeof(*t.UpdatedAt))
	}
	return size
}
