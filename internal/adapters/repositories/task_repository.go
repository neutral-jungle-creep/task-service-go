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
			return nil, domain.ErrTaskNotFound
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
LIMIT $1 OFFSET $2
`

	queryListTasksDesc = `
SELECT id, name, body, status, created_at, updated_at
FROM tasks
ORDER BY id DESC
LIMIT $1 OFFSET $2
`

	queryListTasksToIDAsc = `
SELECT id, name, body, status, created_at, updated_at
FROM tasks
WHERE id < $1
ORDER BY id ASC
LIMIT $2 OFFSET $3
`

	queryListTasksToIDDesc = `
SELECT id, name, body, status, created_at, updated_at
FROM tasks
WHERE id < $1
ORDER BY id DESC
LIMIT $2 OFFSET $3
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

	limit := uint64(defaultListLimit)
	if filter != nil && filter.Limit > 0 {
		limit = filter.Limit
	}
	var offset uint64
	if filter != nil {
		offset = filter.Offset
	}

	switch {
	case hasToID && desc:
		return queryListTasksToIDDesc, []any{filter.ToID, limit, offset}
	case hasToID:
		return queryListTasksToIDAsc, []any{filter.ToID, limit, offset}
	case desc:
		return queryListTasksDesc, []any{limit, offset}
	default:
		return queryListTasksAsc, []any{limit, offset}
	}
}

const (
	queryCountTasks     = `SELECT COUNT(*) FROM tasks`
	queryCountTasksToID = `SELECT COUNT(*) FROM tasks WHERE id < $1`
)

func (r *TaskRepository) Count(filter *ports.ListTasksFilter) (uint64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	var (
		row   *sql.Row
		total uint64
	)
	if filter != nil && filter.ToID > 0 {
		row = r.db.QueryRowContext(ctx, queryCountTasksToID, filter.ToID)
	} else {
		row = r.db.QueryRowContext(ctx, queryCountTasks)
	}
	if err := row.Scan(&total); err != nil {
		return 0, fmt.Errorf("count tasks: %w", err)
	}
	return total, nil
}

const queryUpdateTask = `
UPDATE tasks
SET name = $1, body = $2, status = $3, updated_at = $4
WHERE id = $5
`

func (r *TaskRepository) Update(task *domain.Task) error {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	res, err := r.db.ExecContext(
		ctx,
		queryUpdateTask,
		task.Name,
		task.Body,
		string(task.Status),
		task.UpdatedAt,
		task.ID,
	)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update task rows affected: %w", err)
	}
	if affected == 0 {
		return domain.ErrTaskNotFound
	}
	return nil
}

const queryDeleteTask = `DELETE FROM tasks WHERE id = $1`

func (r *TaskRepository) Delete(id uint64) error {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	res, err := r.db.ExecContext(ctx, queryDeleteTask, id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete task rows affected: %w", err)
	}
	if affected == 0 {
		return domain.ErrTaskNotFound
	}
	return nil
}
