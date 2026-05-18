package domain

import (
	"errors"
	"time"
	"unsafe"
)

// ErrTaskNotFound is returned by repositories/services when the requested
// task does not exist. HTTP handlers translate it to 404.
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
	// Struct headers (string headers store ptr+len, time.Time is fixed-size).
	size := uint64(unsafe.Sizeof(*t))

	// String backing arrays — these dominate footprint for non-trivial tasks.
	size += uint64(len(t.Name))
	size += uint64(len(t.Body))
	size += uint64(len(t.Status))

	// UpdatedAt header is already in unsafe.Sizeof(*t); add the time.Time payload it points to.
	if t.UpdatedAt != nil {
		size += uint64(unsafe.Sizeof(*t.UpdatedAt))
	}
	return size
}
