package domain

import (
	"errors"
	"time"
	"unsafe"
)

var (
	ErrTaskNotFound            = errors.New("task not found")
	ErrInvalidStatusTransition = errors.New("invalid status transition")
	ErrUnknownStatus           = errors.New("unknown status")
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

func (s TaskStatus) IsValid() bool {
	switch s {
	case TaskStatusNew, TaskStatusInProcess, TaskStatusPause, TaskStatusComplete, TaskStatusCancel:
		return true
	default:
		return false
	}
}

// CanTransitionTo enforces the lifecycle:
//   - NEW       → IN_PROCESS
//   - IN_PROCESS → COMPLETE
//   - any non-terminal → PAUSE / CANCEL
//   - PAUSE → IN_PROCESS (resume)
//
// COMPLETE and CANCEL are terminal — nothing leaves them.
func (s TaskStatus) CanTransitionTo(next TaskStatus) bool {
	if !next.IsValid() {
		return false
	}
	if s == next {
		return true // idempotent
	}
	switch next {
	case TaskStatusCancel, TaskStatusPause:
		return s != TaskStatusComplete && s != TaskStatusCancel
	case TaskStatusInProcess:
		return s == TaskStatusNew || s == TaskStatusPause
	case TaskStatusComplete:
		return s == TaskStatusInProcess
	case TaskStatusNew:
		return false // can't go back to NEW
	default:
		return false
	}
}

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
