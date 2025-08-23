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

func (s *TaskService) Create(task *domain.Task) (int64, error) {
	id, err := s.repository.Store(task)
	if err != nil {
		return 0, err
	}

	task.ID = id
	s.cache.Store(task)

	return id, nil
}

func (s *TaskService) List() ([]*domain.Task, error) {
	tasks := s.cache.List()

	tasks, err := s.repository.List() // todo тут надо добавить в метод лист репозитория параметры получения задач, которых уже нет в кеше
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

func (s *TaskService) Get(id int64) (*domain.Task, error) {
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
