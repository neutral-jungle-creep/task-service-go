package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"task-service/internal/domain"
)

func TestNewTask(t *testing.T) {
	t.Parallel()

	before := time.Now()
	task := domain.NewTask("name", "body")
	after := time.Now()

	require.NotNil(t, task)
	assert.Equal(t, "name", task.Name)
	assert.Equal(t, "body", task.Body)
	assert.Equal(t, domain.TaskStatusNew, task.Status)
	assert.Zero(t, task.ID)
	assert.Nil(t, task.UpdatedAt)
	assert.True(t, !task.CreatedAt.Before(before) && !task.CreatedAt.After(after),
		"CreatedAt %v should be within [%v, %v]", task.CreatedAt, before, after)
}

func TestTaskStatus_String(t *testing.T) {
	t.Parallel()

	cases := map[domain.TaskStatus]string{
		domain.TaskStatusNew:       "NEW",
		domain.TaskStatusInProcess: "IN_PROCESS",
		domain.TaskStatusPause:     "PAUSE",
		domain.TaskStatusComplete:  "COMPLETE",
		domain.TaskStatusCancel:    "CANCEL",
	}
	for status, want := range cases {
		assert.Equal(t, want, status.String())
	}
}

func TestTask_Size(t *testing.T) {
	t.Parallel()

	a := &domain.Task{ID: 1}
	b := &domain.Task{ID: 2, Name: "longer name", Body: "longer body content"}

	assert.NotZero(t, a.Size())
	assert.GreaterOrEqual(t, b.Size(), a.Size(),
		"task with non-empty fields should not report smaller size than empty one")
}
