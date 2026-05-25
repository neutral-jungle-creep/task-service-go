package services

import (
	"errors"
	"fmt"
	"time"

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

func (s *TaskService) List(limit, offset uint64) ([]*domain.Task, uint64, error) {
	filter := &ports.ListTasksFilter{
		Sort:   ports.SortAsc,
		Limit:  limit,
		Offset: offset,
	}

	total, err := s.repository.Count(filter)
	if err != nil {
		s.logger.AsyncError("failed to count tasks", err)
		return nil, 0, err
	}

	tasks, err := s.repository.List(filter)
	if err != nil {
		s.logger.AsyncError("failed to list tasks", err)
		return nil, 0, err
	}
	s.logger.AsyncDebug(fmt.Sprintf("listed %d tasks (total %d, limit=%d offset=%d)", len(tasks), total, limit, offset))

	return tasks, total, nil
}

func (s *TaskService) Get(id uint64) (*domain.Task, error) {
	task, ok := s.cache.Get(id)
	if ok {
		s.logger.AsyncDebug(fmt.Sprintf("found task %d from cache", task.ID))
		return task, nil
	}

	task, err := s.repository.Get(id)
	if err != nil {
		if errors.Is(err, domain.ErrTaskNotFound) {
			s.logger.AsyncDebug(fmt.Sprintf("task %d not found", id))
			return nil, err
		}
		s.logger.AsyncError("failed to get task", err)
		return nil, err
	}
	s.logger.AsyncDebug(fmt.Sprintf("found task %d from repository", task.ID))

	return task, nil
}

func (s *TaskService) Update(id uint64, params ports.UpdateTaskParams) (*domain.Task, error) {
	task, err := s.repository.Get(id)
	if err != nil {
		if errors.Is(err, domain.ErrTaskNotFound) {
			return nil, err
		}
		s.logger.AsyncError("failed to load task for update", err)
		return nil, err
	}

	if params.Name != nil {
		task.Name = *params.Name
	}
	if params.Body != nil {
		task.Body = *params.Body
	}
	if params.Status != nil {
		next := domain.TaskStatus(*params.Status)
		if !next.IsValid() {
			return nil, domain.ErrUnknownStatus
		}
		if !task.Status.CanTransitionTo(next) {
			return nil, domain.ErrInvalidStatusTransition
		}
		task.Status = next
	}

	now := time.Now()
	task.UpdatedAt = &now

	err = s.repository.Update(task)
	if err != nil {
		if errors.Is(err, domain.ErrTaskNotFound) {
			s.cache.Delete(id)
			return nil, err
		}
		s.logger.AsyncError("failed to update task", err)
		return nil, err
	}
	s.cache.Store(task)
	s.logger.AsyncDebug(fmt.Sprintf("updated task %d", id))
	return task, nil
}

func (s *TaskService) Delete(id uint64) error {
	err := s.repository.Delete(id)
	if err != nil {
		if errors.Is(err, domain.ErrTaskNotFound) {
			s.cache.Delete(id)
			return err
		}
		s.logger.AsyncError("failed to delete task", err)
		return err
	}
	s.cache.Delete(id)
	s.logger.AsyncDebug(fmt.Sprintf("deleted task %d", id))
	return nil
}
