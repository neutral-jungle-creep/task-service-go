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

func TestTaskStatus_IsValid(t *testing.T) {
	t.Parallel()

	for _, s := range []domain.TaskStatus{
		domain.TaskStatusNew, domain.TaskStatusInProcess, domain.TaskStatusPause,
		domain.TaskStatusComplete, domain.TaskStatusCancel,
	} {
		assert.True(t, s.IsValid(), "valid status %q must be IsValid()", s)
	}
	assert.False(t, domain.TaskStatus("BOGUS").IsValid())
	assert.False(t, domain.TaskStatus("").IsValid())
}

func TestTaskStatus_CanTransitionTo(t *testing.T) {
	t.Parallel()

	type tc struct {
		from, to domain.TaskStatus
		want     bool
	}
	cases := []tc{
		// NEW
		{domain.TaskStatusNew, domain.TaskStatusInProcess, true},
		{domain.TaskStatusNew, domain.TaskStatusPause, true},
		{domain.TaskStatusNew, domain.TaskStatusCancel, true},
		{domain.TaskStatusNew, domain.TaskStatusComplete, false},
		{domain.TaskStatusNew, domain.TaskStatusNew, true},
		// IN_PROCESS
		{domain.TaskStatusInProcess, domain.TaskStatusComplete, true},
		{domain.TaskStatusInProcess, domain.TaskStatusPause, true},
		{domain.TaskStatusInProcess, domain.TaskStatusCancel, true},
		{domain.TaskStatusInProcess, domain.TaskStatusNew, false},
		// PAUSE → IN_PROCESS (resume)
		{domain.TaskStatusPause, domain.TaskStatusInProcess, true},
		{domain.TaskStatusPause, domain.TaskStatusCancel, true},
		{domain.TaskStatusPause, domain.TaskStatusComplete, false},
		// COMPLETE is terminal
		{domain.TaskStatusComplete, domain.TaskStatusPause, false},
		{domain.TaskStatusComplete, domain.TaskStatusCancel, false},
		{domain.TaskStatusComplete, domain.TaskStatusComplete, true},
		// CANCEL is terminal
		{domain.TaskStatusCancel, domain.TaskStatusPause, false},
		{domain.TaskStatusCancel, domain.TaskStatusInProcess, false},
		// unknown target
		{domain.TaskStatusNew, domain.TaskStatus("BOGUS"), false},
	}
	for _, c := range cases {
		got := c.from.CanTransitionTo(c.to)
		assert.Equalf(t, c.want, got, "%s → %s", c.from, c.to)
	}
}
