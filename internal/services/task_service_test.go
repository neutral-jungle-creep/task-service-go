package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"task-service/internal/domain"
	"task-service/internal/ports"
	"task-service/internal/services"
	"task-service/pkg/logging"
)

type stubRepo struct {
	storeFunc func(*domain.Task) (uint64, error)
	listFunc  func(*ports.ListTasksFilter) ([]*domain.Task, error)
	getFunc   func(uint64) (*domain.Task, error)
}

func (s *stubRepo) Store(t *domain.Task) (uint64, error) {
	if s.storeFunc != nil {
		return s.storeFunc(t)
	}
	return 0, nil
}

func (s *stubRepo) List(f *ports.ListTasksFilter) ([]*domain.Task, error) {
	if s.listFunc != nil {
		return s.listFunc(f)
	}
	return nil, nil
}

func (s *stubRepo) Get(id uint64) (*domain.Task, error) {
	if s.getFunc != nil {
		return s.getFunc(id)
	}
	return &domain.Task{}, nil
}

type stubCache struct {
	stored   []*domain.Task
	listFunc func() ([]*domain.Task, uint64)
	getFunc  func(uint64) (*domain.Task, bool)
}

func (s *stubCache) Store(t *domain.Task) {
	s.stored = append(s.stored, t)
}

func (s *stubCache) List() ([]*domain.Task, uint64) {
	if s.listFunc != nil {
		return s.listFunc()
	}
	return nil, 1
}

func (s *stubCache) Get(id uint64) (*domain.Task, bool) {
	if s.getFunc != nil {
		return s.getFunc(id)
	}
	return nil, false
}

func newAsyncLogger(t *testing.T) *logging.AsyncLogger {
	t.Helper()
	core, err := logging.NewLogger("error", "test", "test")
	require.NoError(t, err)

	async := logging.NewAsyncLogger(context.Background(), core)

	done := make(chan struct{})
	go func() {
		_ = async.Process()
		close(done)
	}()
	t.Cleanup(func() {
		async.Stop()
		<-done
	})
	return async
}

func TestTaskService_Create(t *testing.T) {
	t.Parallel()

	repo := &stubRepo{
		storeFunc: func(_ *domain.Task) (uint64, error) { return 99, nil },
	}
	cache := &stubCache{}

	svc := services.NewTaskService(newAsyncLogger(t), repo, cache)
	id, err := svc.Create(domain.NewTask("n", "b"))

	require.NoError(t, err)
	assert.Equal(t, uint64(99), id)
	require.Len(t, cache.stored, 1)
	assert.Equal(t, uint64(99), cache.stored[0].ID)
}

func TestTaskService_Create_RepoError(t *testing.T) {
	t.Parallel()

	want := errors.New("boom")
	svc := services.NewTaskService(
		newAsyncLogger(t),
		&stubRepo{storeFunc: func(_ *domain.Task) (uint64, error) { return 0, want }},
		&stubCache{},
	)

	id, err := svc.Create(domain.NewTask("n", "b"))
	require.ErrorIs(t, err, want)
	assert.Zero(t, id)
}

func TestTaskService_Get_FromCache(t *testing.T) {
	t.Parallel()

	cache := &stubCache{
		getFunc: func(id uint64) (*domain.Task, bool) {
			return &domain.Task{ID: id, Name: "cached"}, true
		},
	}
	repoCalls := 0
	repo := &stubRepo{
		getFunc: func(uint64) (*domain.Task, error) {
			repoCalls++
			return &domain.Task{}, nil
		},
	}

	svc := services.NewTaskService(newAsyncLogger(t), repo, cache)
	got, err := svc.Get(7)
	require.NoError(t, err)
	assert.Equal(t, "cached", got.Name)
	assert.Zero(t, repoCalls, "repository should not be queried when cache hits")
}

func TestTaskService_Get_FromRepo(t *testing.T) {
	t.Parallel()

	repo := &stubRepo{
		getFunc: func(id uint64) (*domain.Task, error) {
			return &domain.Task{ID: id, Name: "from-db"}, nil
		},
	}
	cache := &stubCache{}

	svc := services.NewTaskService(newAsyncLogger(t), repo, cache)
	got, err := svc.Get(11)
	require.NoError(t, err)
	assert.Equal(t, "from-db", got.Name)
	assert.Equal(t, uint64(11), got.ID)
}

func TestTaskService_List_AllInCache(t *testing.T) {
	t.Parallel()

	cached := []*domain.Task{{ID: 1}, {ID: 2}}
	cache := &stubCache{
		listFunc: func() ([]*domain.Task, uint64) {
			return cached, 1
		},
	}

	svc := services.NewTaskService(newAsyncLogger(t), &stubRepo{}, cache)

	got, err := svc.List()
	require.NoError(t, err)
	assert.Equal(t, cached, got)
}

func TestTaskService_List_MergesRepoAndCache(t *testing.T) {
	t.Parallel()

	cache := &stubCache{
		listFunc: func() ([]*domain.Task, uint64) {
			return []*domain.Task{{ID: 5}, {ID: 6}}, 5
		},
	}
	repo := &stubRepo{
		listFunc: func(filter *ports.ListTasksFilter) ([]*domain.Task, error) {
			require.NotNil(t, filter)
			assert.Equal(t, uint64(5), filter.ToID)
			return []*domain.Task{{ID: 1}, {ID: 2}}, nil
		},
	}

	svc := services.NewTaskService(newAsyncLogger(t), repo, cache)

	got, err := svc.List()
	require.NoError(t, err)
	require.Len(t, got, 4)
	assert.Equal(t, uint64(1), got[0].ID)
	assert.Equal(t, uint64(2), got[1].ID)
	assert.Equal(t, uint64(5), got[2].ID)
	assert.Equal(t, uint64(6), got[3].ID)
}
