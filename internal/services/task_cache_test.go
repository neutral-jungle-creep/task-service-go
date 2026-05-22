package services_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"task-service/internal/domain"
	"task-service/internal/ports"
	"task-service/internal/services"
)

type fakeRepo struct {
	listFunc func(filter *ports.ListTasksFilter) ([]*domain.Task, error)
}

func (f *fakeRepo) Store(_ *domain.Task) (uint64, error) { return 0, nil }
func (f *fakeRepo) Get(_ uint64) (*domain.Task, error)   { return &domain.Task{}, nil }
func (f *fakeRepo) List(filter *ports.ListTasksFilter) ([]*domain.Task, error) {
	if f.listFunc != nil {
		return f.listFunc(filter)
	}
	return nil, nil
}

func newCache(t *testing.T, repo ports.TaskRepository) *services.TaskCache {
	t.Helper()
	c, err := services.NewTaskCache(1, time.Hour, repo)
	require.NoError(t, err)
	return c
}

func TestTaskCache_StoreAndGet(t *testing.T) {
	t.Parallel()

	c := newCache(t, &fakeRepo{})
	task := &domain.Task{ID: 42, Name: "n", Body: "b"}

	c.Store(task)

	got, ok := c.Get(42)
	require.True(t, ok)
	assert.Equal(t, task, got)

	_, ok = c.Get(7)
	assert.False(t, ok)
}

func TestTaskCache_List(t *testing.T) {
	t.Parallel()

	c := newCache(t, &fakeRepo{})
	c.Store(&domain.Task{ID: 1})
	c.Store(&domain.Task{ID: 2})
	c.Store(&domain.Task{ID: 3})

	tasks, _ := c.List()
	assert.Len(t, tasks, 3)
}

func TestTaskCache_FillFromRepository(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{
		listFunc: func(filter *ports.ListTasksFilter) ([]*domain.Task, error) {
			require.NotNil(t, filter)
			assert.Equal(t, ports.SortDesc, filter.Sort)
			return []*domain.Task{
				{ID: 5, Name: "five"},
				{ID: 4, Name: "four"},
			}, nil
		},
	}

	c, err := services.NewTaskCache(1024, time.Hour, repo)
	require.NoError(t, err)

	got, ok := c.Get(5)
	require.True(t, ok)
	assert.Equal(t, "five", got.Name)

	got, ok = c.Get(4)
	require.True(t, ok)
	assert.Equal(t, "four", got.Name)
}
