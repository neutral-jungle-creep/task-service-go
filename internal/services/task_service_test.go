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
	storeFunc  func(context.Context, *domain.Task) (uint64, error)
	listFunc   func(context.Context, *ports.ListTasksFilter) ([]*domain.Task, error)
	countFunc  func(context.Context, *ports.ListTasksFilter) (uint64, error)
	getFunc    func(context.Context, uint64) (*domain.Task, error)
	updateFunc func(context.Context, *domain.Task) error
	deleteFunc func(context.Context, uint64) error
}

func (s *stubRepo) Update(ctx context.Context, t *domain.Task) error {
	if s.updateFunc != nil {
		return s.updateFunc(ctx, t)
	}
	return nil
}

func (s *stubRepo) Delete(ctx context.Context, id uint64) error {
	if s.deleteFunc != nil {
		return s.deleteFunc(ctx, id)
	}
	return nil
}

func (s *stubRepo) Store(ctx context.Context, t *domain.Task) (uint64, error) {
	if s.storeFunc != nil {
		return s.storeFunc(ctx, t)
	}
	return 0, nil
}

func (s *stubRepo) List(ctx context.Context, f *ports.ListTasksFilter) ([]*domain.Task, error) {
	if s.listFunc != nil {
		return s.listFunc(ctx, f)
	}
	return nil, nil
}

func (s *stubRepo) Count(ctx context.Context, f *ports.ListTasksFilter) (uint64, error) {
	if s.countFunc != nil {
		return s.countFunc(ctx, f)
	}
	return 0, nil
}

func (s *stubRepo) Get(ctx context.Context, id uint64) (*domain.Task, error) {
	if s.getFunc != nil {
		return s.getFunc(ctx, id)
	}
	return &domain.Task{}, nil
}

type stubCache struct {
	stored   []*domain.Task
	deleted  []uint64
	listFunc func() ([]*domain.Task, uint64)
	getFunc  func(uint64) (*domain.Task, bool)
}

func (s *stubCache) Delete(id uint64) {
	s.deleted = append(s.deleted, id)
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
		storeFunc: func(_ context.Context, _ *domain.Task) (uint64, error) { return 99, nil },
	}
	cache := &stubCache{}

	svc := services.NewTaskService(newAsyncLogger(t), repo, cache)
	id, err := svc.Create(context.Background(), domain.NewTask("n", "b"))

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
		&stubRepo{storeFunc: func(_ context.Context, _ *domain.Task) (uint64, error) { return 0, want }},
		&stubCache{},
	)

	id, err := svc.Create(context.Background(), domain.NewTask("n", "b"))
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
		getFunc: func(context.Context, uint64) (*domain.Task, error) {
			repoCalls++
			return &domain.Task{}, nil
		},
	}

	svc := services.NewTaskService(newAsyncLogger(t), repo, cache)
	got, err := svc.Get(context.Background(), 7)
	require.NoError(t, err)
	assert.Equal(t, "cached", got.Name)
	assert.Zero(t, repoCalls, "repository should not be queried when cache hits")
}

func TestTaskService_Get_FromRepo(t *testing.T) {
	t.Parallel()

	repo := &stubRepo{
		getFunc: func(_ context.Context, id uint64) (*domain.Task, error) {
			return &domain.Task{ID: id, Name: "from-db"}, nil
		},
	}
	cache := &stubCache{}

	svc := services.NewTaskService(newAsyncLogger(t), repo, cache)
	got, err := svc.Get(context.Background(), 11)
	require.NoError(t, err)
	assert.Equal(t, "from-db", got.Name)
	assert.Equal(t, uint64(11), got.ID)
}

func TestTaskService_List_PassesPaginationToRepo(t *testing.T) {
	t.Parallel()

	repo := &stubRepo{
		listFunc: func(_ context.Context, f *ports.ListTasksFilter) ([]*domain.Task, error) {
			require.NotNil(t, f)
			assert.Equal(t, uint64(25), f.Limit)
			assert.Equal(t, uint64(50), f.Offset)
			return []*domain.Task{{ID: 1}, {ID: 2}}, nil
		},
		countFunc: func(_ context.Context, f *ports.ListTasksFilter) (uint64, error) {
			require.NotNil(t, f)
			return 123, nil
		},
	}

	svc := services.NewTaskService(newAsyncLogger(t), repo, &stubCache{})

	tasks, total, err := svc.List(context.Background(), 25, 50)
	require.NoError(t, err)
	require.Len(t, tasks, 2)
	assert.Equal(t, uint64(123), total)
}

func TestTaskService_List_CountError(t *testing.T) {
	t.Parallel()

	repo := &stubRepo{
		countFunc: func(context.Context, *ports.ListTasksFilter) (uint64, error) { return 0, errors.New("count failed") },
	}

	svc := services.NewTaskService(newAsyncLogger(t), repo, &stubCache{})

	_, _, err := svc.List(context.Background(), 10, 0)
	require.Error(t, err)
}

func TestTaskService_List_ListError(t *testing.T) {
	t.Parallel()

	repo := &stubRepo{
		countFunc: func(context.Context, *ports.ListTasksFilter) (uint64, error) { return 5, nil },
		listFunc: func(context.Context, *ports.ListTasksFilter) ([]*domain.Task, error) {
			return nil, errors.New("list failed")
		},
	}

	svc := services.NewTaskService(newAsyncLogger(t), repo, &stubCache{})

	_, _, err := svc.List(context.Background(), 10, 0)
	require.Error(t, err)
}

func TestTaskService_Update_OK(t *testing.T) {
	t.Parallel()

	repo := &stubRepo{
		getFunc: func(_ context.Context, id uint64) (*domain.Task, error) {
			return &domain.Task{ID: id, Name: "old", Body: "old-body", Status: domain.TaskStatusNew}, nil
		},
		updateFunc: func(_ context.Context, task *domain.Task) error {
			assert.Equal(t, "new-name", task.Name)
			assert.Equal(t, domain.TaskStatusInProcess, task.Status)
			require.NotNil(t, task.UpdatedAt)
			return nil
		},
	}
	cache := &stubCache{}

	newName := "new-name"
	newStatus := domain.TaskStatusInProcess.String()

	svc := services.NewTaskService(newAsyncLogger(t), repo, cache)
	updated, err := svc.Update(context.Background(), 1, ports.UpdateTaskParams{
		Name:   &newName,
		Status: &newStatus,
	})
	require.NoError(t, err)
	assert.Equal(t, "new-name", updated.Name)
	assert.Equal(t, domain.TaskStatusInProcess, updated.Status)
	require.Len(t, cache.stored, 1, "updated task must be re-stored in cache")
}

func TestTaskService_Update_NotFound(t *testing.T) {
	t.Parallel()

	repo := &stubRepo{
		getFunc: func(context.Context, uint64) (*domain.Task, error) { return nil, domain.ErrTaskNotFound },
	}
	svc := services.NewTaskService(newAsyncLogger(t), repo, &stubCache{})
	_, err := svc.Update(context.Background(), 1, ports.UpdateTaskParams{})
	require.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestTaskService_Update_InvalidTransition(t *testing.T) {
	t.Parallel()

	repo := &stubRepo{
		getFunc: func(context.Context, uint64) (*domain.Task, error) {
			return &domain.Task{Status: domain.TaskStatusComplete}, nil
		},
	}
	svc := services.NewTaskService(newAsyncLogger(t), repo, &stubCache{})

	status := domain.TaskStatusInProcess.String()
	_, err := svc.Update(context.Background(), 1, ports.UpdateTaskParams{Status: &status})
	require.ErrorIs(t, err, domain.ErrInvalidStatusTransition)
}

func TestTaskService_Update_UnknownStatus(t *testing.T) {
	t.Parallel()

	repo := &stubRepo{
		getFunc: func(context.Context, uint64) (*domain.Task, error) {
			return &domain.Task{Status: domain.TaskStatusNew}, nil
		},
	}
	svc := services.NewTaskService(newAsyncLogger(t), repo, &stubCache{})

	bogus := "BOGUS"
	_, err := svc.Update(context.Background(), 1, ports.UpdateTaskParams{Status: &bogus})
	require.ErrorIs(t, err, domain.ErrUnknownStatus)
}

func TestTaskService_Delete_OK(t *testing.T) {
	t.Parallel()

	repo := &stubRepo{
		deleteFunc: func(context.Context, uint64) error { return nil },
	}
	cache := &stubCache{}
	svc := services.NewTaskService(newAsyncLogger(t), repo, cache)

	require.NoError(t, svc.Delete(context.Background(), 42))
	assert.Equal(t, []uint64{42}, cache.deleted)
}

func TestTaskService_Delete_NotFound(t *testing.T) {
	t.Parallel()

	repo := &stubRepo{
		deleteFunc: func(context.Context, uint64) error { return domain.ErrTaskNotFound },
	}
	cache := &stubCache{}
	svc := services.NewTaskService(newAsyncLogger(t), repo, cache)

	err := svc.Delete(context.Background(), 99)
	require.ErrorIs(t, err, domain.ErrTaskNotFound)
	assert.Equal(t, []uint64{99}, cache.deleted, "cache eviction should still run on not-found")
}
