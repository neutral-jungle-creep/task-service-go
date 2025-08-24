package services

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"task-service/internal/domain"
	"task-service/internal/ports"
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

func NewTaskCache(memoryLimitMB int, memoryMonitorInterval time.Duration, repository ports.TaskRepository) (*TaskCache, error) {
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

	err := s.fill(repository)
	if err != nil {
		return nil, err
	}
	go s.memoryMonitor()
	return s, nil
}

func (t *TaskCache) fill(repository ports.TaskRepository) error {
	tasks, err := repository.List(&ports.ListTasksFilter{ // получение данных с конца, чтобы добавить в кеш новейшие
		Sort: ports.SortDesc,
	})
	if err != nil {
		return err
	}

	var totalSize uint64
	for _, task := range tasks {
		totalSize += task.Size()
		if totalSize >= uint64(float32(t.cleanupStartMB)*0.9) {
			break
		}
		t.tasks.Store(task.ID, task)
		t.firstKey.Store(task.ID)
	}
	return nil
}

func (t *TaskCache) memoryMonitor() {
	ticker := time.NewTicker(t.memoryMonitorInterval * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)

		if memStats.Alloc > t.cleanupStartMB*1024*1024 {
			t.cleanup()
		}
	}
}

func (t *TaskCache) cleanup() {
	cleanupCount := t.len.Load() / 5 // удалится 20% самых старых записей
	firstStoredKey := t.firstKey.Load()
	var newFirstStoredKey uint64

	for key := firstStoredKey; key < cleanupCount; key++ { // эта реализация актуальна только для данных у которых id - автоинкремент
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

func (t *TaskCache) List() ([]*domain.Task, uint64) {
	tasks := make([]*domain.Task, 0, t.len.Load())

	t.tasks.Range(func(key, value interface{}) bool {
		task, ok := value.(*domain.Task)
		if ok {
			tasks = append(tasks, task)
		}
		return true
	})
	return tasks, t.firstKey.Load()
}

func (t *TaskCache) Get(id uint64) (*domain.Task, bool) {
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
