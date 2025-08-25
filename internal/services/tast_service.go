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

// понятно, излишнее логирование затормаживает программу, для примера работоспособности кеша и асинхронного логирования
// добавила много дебаг логов

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
		ToId: firstTaskKey, // в репо будет запрос получения всех айдишек которые меньше firstTaskKey
	})
	if err != nil {
		s.logger.AsyncError("failed to list tasks", err)
		return nil, err
	}
	s.logger.AsyncDebug(fmt.Sprintf("list %d tasks from repository", len(tasksFromDb)))

	tasksFromDb = append(tasksFromDb, tasksFromCache...)
	return tasksFromDb, nil
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
