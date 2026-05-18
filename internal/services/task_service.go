package services

import (
	"fmt"

	"task-service/internal/domain"
	"task-service/internal/ports"
	"task-service/pkg/logging"
)

type TaskService struct {
	logger     *logging.AsyncLogger
	repository ports.TaskRepository
	cache      ports.TaskCache
}

func NewTaskService(
	logger *logging.AsyncLogger,
	repository ports.TaskRepository,
	cache ports.TaskCache,
) *TaskService {
	return &TaskService{
		logger:     logger,
		repository: repository,
		cache:      cache,
	}
}

// verbose debug logging is intentional here to demonstrate the cache and the async logger in action.

func (s *TaskService) Create(task *domain.Task) (uint64, error) {
	id, err := s.repository.Store(task)
	if err != nil {
		s.logger.AsyncError("failed to store task", err)
		return 0, err
	}

	s.logger.AsyncDebug(fmt.Sprintf("stored task %d to repository", id))
	task.ID = id
	s.cache.Store(task)
	s.logger.AsyncDebug(fmt.Sprintf("stored task %d to catche", id))

	return id, nil
}

func (s *TaskService) List() ([]*domain.Task, error) {
	tasksFromCache, firstTaskKey := s.cache.List()
	if firstTaskKey == 1 {
		s.logger.AsyncDebug("all tasks in cache")
		return tasksFromCache, nil
	}
	s.logger.AsyncDebug(fmt.Sprintf("list %d tasks from cache", len(tasksFromCache)))

	tasksFromDb, err := s.repository.List(&ports.ListTasksFilter{
		ToID: firstTaskKey, // ask the repository for all ids less than firstTaskKey
	})
	if err != nil {
		s.logger.AsyncError("failed to list tasks", err)
		return nil, err
	}
	s.logger.AsyncDebug(fmt.Sprintf("list %d tasks from repository", len(tasksFromDb)))

	// Cache cleanup may have moved firstKey forward between the cache snapshot
	// and the repository query, so rows in the overlapping range can appear in
	// both lists. Dedupe by id, preferring the (fresher) cache copy.
	seen := make(map[uint64]struct{}, len(tasksFromCache))
	for _, t := range tasksFromCache {
		seen[t.ID] = struct{}{}
	}
	merged := make([]*domain.Task, 0, len(tasksFromDb)+len(tasksFromCache))
	for _, t := range tasksFromDb {
		if _, dup := seen[t.ID]; !dup {
			merged = append(merged, t)
		}
	}
	merged = append(merged, tasksFromCache...)
	return merged, nil
}

func (s *TaskService) Get(id uint64) (*domain.Task, error) {
	task, ok := s.cache.Get(id)
	if ok {
		s.logger.AsyncDebug(fmt.Sprintf("found task %d from cache", task.ID))
		return task, nil
	}

	task, err := s.repository.Get(id)
	if err != nil {
		s.logger.AsyncError("failed to get task", err)
		return nil, err
	}
	s.logger.AsyncDebug(fmt.Sprintf("found task %d from repository", task.ID))

	return task, nil
}
