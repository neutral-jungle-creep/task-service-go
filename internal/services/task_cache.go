package services

import (
	"context"
	"time"

	"task-service/internal/domain"
	"task-service/internal/ports"
	"task-service/pkg/cache"
)

const (
	defaultMemoryUsageMB         = 1024
	defaultMemoryMonitorInterval = 5 * time.Second
)

// TaskCache adapts pkg/cache.Cache[uint64, *domain.Task] to ports.TaskCache
// and pre-fills itself from the repository on startup so that the most
// recent records sit in memory immediately.
type TaskCache struct {
	inner *cache.Cache[uint64, *domain.Task]
}

func NewTaskCache(memoryLimitMB int, memoryMonitorInterval time.Duration, repository ports.TaskRepository) (*TaskCache, error) {
	if memoryLimitMB <= 0 {
		memoryLimitMB = defaultMemoryUsageMB
	}
	if memoryMonitorInterval <= 0 {
		memoryMonitorInterval = defaultMemoryMonitorInterval
	}

	inner := cache.New[uint64, *domain.Task](memoryLimitMB, memoryMonitorInterval)
	tc := &TaskCache{inner: inner}

	if err := tc.fill(repository); err != nil {
		return nil, err
	}
	return tc, nil
}

// Run starts the underlying memory monitor; intended to be registered as a
// background job by the DI container.
func (t *TaskCache) Run(ctx context.Context) error {
	return t.inner.Run(ctx)
}

func (t *TaskCache) Store(task *domain.Task) {
	t.inner.Store(task.ID, task)
}

func (t *TaskCache) Get(id uint64) (*domain.Task, bool) {
	return t.inner.Get(id)
}

func (t *TaskCache) List() ([]*domain.Task, uint64) {
	return t.inner.List()
}

func (t *TaskCache) fill(repository ports.TaskRepository) error {
	tasks, err := repository.List(&ports.ListTasksFilter{ // fetch from the newest end so the cache holds the most recent records
		Sort: ports.SortDesc,
	})
	if err != nil {
		return err
	}

	// budget is 90% of the cleanup threshold, converted from MB to bytes —
	// task.Size() returns bytes, so the comparison must be in bytes too.
	budget := t.inner.CleanupStartMB() * 1024 * 1024 * 9 / 10
	var totalSize uint64
	for _, task := range tasks {
		totalSize += task.Size()
		if totalSize >= budget {
			break
		}
		t.inner.Store(task.ID, task)
		t.inner.SetFirstKey(task.ID)
	}
	return nil
}
