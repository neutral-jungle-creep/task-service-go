package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"task-service/internal/domain"
	"task-service/internal/ports"
)

const (
	defaultListLimit = 1000
	queryTimeout     = 5 * time.Second
)

type TaskRepository struct {
	db *sql.DB
}

func NewTaskRepository(db *sql.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

const queryStoreTask = `
INSERT INTO tasks (name, body, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id
`

func (r *TaskRepository) Store(task *domain.Task) (uint64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	var id uint64
	err := r.db.QueryRowContext(
		ctx,
		queryStoreTask,
		task.Name,
		task.Body,
		string(task.Status),
		task.CreatedAt,
		task.UpdatedAt,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("store task: %w", err)
	}
	return id, nil
}

const queryGetTask = `
SELECT id, name, body, status, created_at, updated_at
FROM tasks
WHERE id = $1
`

func (r *TaskRepository) Get(id uint64) (*domain.Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	t := &domain.Task{}
	var status string
	err := r.db.QueryRowContext(ctx, queryGetTask, id).Scan(
		&t.ID, &t.Name, &t.Body, &status, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &domain.Task{}, nil
		}
		return nil, fmt.Errorf("get task: %w", err)
	}
	t.Status = domain.TaskStatus(status)
	return t, nil
}

const (
	queryListTasksAsc = `
SELECT id, name, body, status, created_at, updated_at
FROM tasks
ORDER BY id ASC
LIMIT $1
`

	queryListTasksDesc = `
SELECT id, name, body, status, created_at, updated_at
FROM tasks
ORDER BY id DESC
LIMIT $1
`

	queryListTasksToIDAsc = `
SELECT id, name, body, status, created_at, updated_at
FROM tasks
WHERE id < $1
ORDER BY id ASC
LIMIT $2
`

	queryListTasksToIDDesc = `
SELECT id, name, body, status, created_at, updated_at
FROM tasks
WHERE id < $1
ORDER BY id DESC
LIMIT $2
`
)

func (r *TaskRepository) List(filter *ports.ListTasksFilter) ([]*domain.Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	query, args := buildListQuery(filter)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer func() { _ = rows.Close() }()

	tasks := make([]*domain.Task, 0)
	for rows.Next() {
		t := &domain.Task{}
		var status string
		if err := rows.Scan(&t.ID, &t.Name, &t.Body, &status, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		t.Status = domain.TaskStatus(status)
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tasks: %w", err)
	}
	return tasks, nil
}

func buildListQuery(filter *ports.ListTasksFilter) (string, []any) {
	desc := filter != nil && filter.Sort == ports.SortDesc
	hasToID := filter != nil && filter.ToID > 0

	switch {
	case hasToID && desc:
		return queryListTasksToIDDesc, []any{filter.ToID, defaultListLimit}
	case hasToID:
		return queryListTasksToIDAsc, []any{filter.ToID, defaultListLimit}
	case desc:
		return queryListTasksDesc, []any{defaultListLimit}
	default:
		return queryListTasksAsc, []any{defaultListLimit}
	}
}
