package services

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"task-service/internal/domain"
)

const (
	defaultMemoryUsageMB         = 1024
	defaultMemoryMonitorInterval = 5
)

type TaskCache struct {
	tasks                 sync.Map
	len                   atomic.Uint64
	cleanupStartMB        uint64
	memoryMonitorInterval time.Duration
	firstKey              atomic.Uint64
}

func NewTaskCache(memoryLimitMB int, memoryMonitorInterval time.Duration) *TaskCache {
	if memoryLimitMB <= 0 {
		memoryLimitMB = defaultMemoryUsageMB
	}

	if memoryMonitorInterval <= 0 {
		memoryMonitorInterval = defaultMemoryMonitorInterval
	}

	s := &TaskCache{
		tasks:                 sync.Map{},
		cleanupStartMB:        uint64(float32(memoryLimitMB) * 0.9), // когда заполнится 90% памяти, начнется чистка
		memoryMonitorInterval: memoryMonitorInterval,
	}

	go s.memoryMonitor()
	return s
}

func (t *TaskCache) memoryMonitor() {
	ticker := time.NewTicker(t.memoryMonitorInterval * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)

		if memStats.Alloc > t.cleanupStartMB/1000 {
			t.cleanup()
		}
	}
}

func (t *TaskCache) cleanup() {
	cleanupCount := t.len.Load() / 5 // удалится 20% самых старых записей
	firstStoredKey := t.firstKey.Load()
	var newFirstStoredKey uint64

	for key := firstStoredKey; key < cleanupCount; key++ {
		if _, ok := t.tasks.Load(key); ok {
			t.tasks.Delete(key)
			continue
		}
		newFirstStoredKey = key
		break
	}

	if newFirstStoredKey == 0 {
		newFirstStoredKey = cleanupCount
	}

	t.firstKey.Store(newFirstStoredKey)
}

func (t *TaskCache) Store(task *domain.Task) {
	t.tasks.Store(task.ID, task)
}

func (t *TaskCache) List() []*domain.Task {
	tasks := make([]*domain.Task, 0, t.len.Load())

	t.tasks.Range(func(key, value interface{}) bool {
		task, ok := value.(*domain.Task)
		if ok {
			tasks = append(tasks, task)
		}
		return true
	})
	return tasks
}

func (t *TaskCache) Get(id int64) (*domain.Task, bool) {
	val, ok := t.tasks.Load(id)
	if !ok {
		return nil, false
	}

	task, ok := val.(*domain.Task)
	if !ok {
		return nil, false
	}
	return task, ok
}
