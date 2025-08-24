package domain

import (
	"time"
	"unsafe"
)

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
	size := uintptr(0)

	size += unsafe.Sizeof(t.ID)
	size += unsafe.Sizeof(t.Name)
	size += unsafe.Sizeof(t.Body)
	size += unsafe.Sizeof(t.Status)
	size += unsafe.Sizeof(t.CreatedAt)
	size += unsafe.Sizeof(t.UpdatedAt)

	return uint64(size)
}
