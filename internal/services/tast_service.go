package services

import (
	"task-service/internal/domain"
	"task-service/internal/ports"
	"task-service/pkg/logging"
)

type TaskService struct {
	logger     *logging.Logger
	repository ports.TaskRepository
	cache      ports.TaskCache
}

func NewTaskService(
	logger *logging.Logger,
	repository ports.TaskRepository,
	cache ports.TaskCache,
) *TaskService {
	return &TaskService{
		logger:     logger,
		repository: repository,
		cache:      cache,
	}
}

func (s *TaskService) Create(task *domain.Task) (uint64, error) {
	id, err := s.repository.Store(task)
	if err != nil {
		return 0, err
	}

	task.ID = id
	s.cache.Store(task)

	return id, nil
}

func (s *TaskService) List() ([]*domain.Task, error) {
	tasksFromCache, firstTaskKey := s.cache.List()
	if firstTaskKey <= 1 {
		return tasksFromCache, nil
	}

	tasksFromDb, err := s.repository.List(&ports.ListTasksFilter{
		ToId: firstTaskKey, // в репо будет запрос получения всех айдишек которые меньше firstTaskKey
	})
	if err != nil {
		return nil, err
	}

	tasksFromDb = append(tasksFromDb, tasksFromCache...)
	return tasksFromDb, nil
}

func (s *TaskService) Get(id uint64) (*domain.Task, error) {
	task, ok := s.cache.Get(id)
	if ok {
		return task, nil
	}

	task, err := s.repository.Get(id)
	if err != nil {
		return nil, err
	}
	return task, nil
}
