package repositories_test

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"task-service/internal/adapters/repositories"
	"task-service/internal/domain"
	"task-service/internal/ports"
)

func newMock(t *testing.T) (sqlmock.Sqlmock, *repositories.TaskRepository) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return mock, repositories.NewTaskRepository(db)
}

func TestTaskRepository_Store_OK(t *testing.T) {
	t.Parallel()

	mock, repo := newMock(t)

	created := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	task := &domain.Task{
		Name:      "n",
		Body:      "b",
		Status:    domain.TaskStatusNew,
		CreatedAt: created,
	}

	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO tasks")).
		WithArgs("n", "b", domain.TaskStatusNew.String(), created, task.UpdatedAt).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))

	id, err := repo.Store(context.Background(), task)
	require.NoError(t, err)
	assert.Equal(t, uint64(42), id)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTaskRepository_Store_DBError(t *testing.T) {
	t.Parallel()

	mock, repo := newMock(t)

	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO tasks")).
		WillReturnError(errors.New("connection refused"))

	id, err := repo.Store(context.Background(), &domain.Task{
		Name:      "x",
		Body:      "y",
		Status:    domain.TaskStatusNew,
		CreatedAt: time.Now(),
	})
	require.Error(t, err)
	assert.Zero(t, id)
	assert.Contains(t, err.Error(), "store task")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTaskRepository_Get_OK(t *testing.T) {
	t.Parallel()

	mock, repo := newMock(t)

	created := time.Date(2024, 2, 3, 4, 5, 6, 0, time.UTC)
	updated := time.Date(2024, 3, 4, 5, 6, 7, 0, time.UTC)

	rows := sqlmock.NewRows([]string{"id", "name", "body", "status", "created_at", "updated_at"}).
		AddRow(int64(7), "n", "b", "IN_PROCESS", created, updated)

	mock.ExpectQuery(`SELECT .* FROM tasks WHERE id = \$1`).
		WithArgs(uint64(7)).
		WillReturnRows(rows)

	task, err := repo.Get(context.Background(), 7)
	require.NoError(t, err)
	require.NotNil(t, task)
	assert.Equal(t, uint64(7), task.ID)
	assert.Equal(t, "n", task.Name)
	assert.Equal(t, "b", task.Body)
	assert.Equal(t, domain.TaskStatusInProcess, task.Status)
	assert.Equal(t, created, task.CreatedAt)
	require.NotNil(t, task.UpdatedAt)
	assert.Equal(t, updated, *task.UpdatedAt)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTaskRepository_Get_NotFound(t *testing.T) {
	t.Parallel()

	mock, repo := newMock(t)

	mock.ExpectQuery(`SELECT .* FROM tasks WHERE id = \$1`).
		WithArgs(uint64(999)).
		WillReturnError(sql.ErrNoRows)

	task, err := repo.Get(context.Background(), 999)
	require.ErrorIs(t, err, domain.ErrTaskNotFound)
	assert.Nil(t, task)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTaskRepository_Get_DBError(t *testing.T) {
	t.Parallel()

	mock, repo := newMock(t)

	mock.ExpectQuery(`SELECT .* FROM tasks WHERE id = \$1`).
		WithArgs(uint64(1)).
		WillReturnError(errors.New("timeout"))

	task, err := repo.Get(context.Background(), 1)
	require.Error(t, err)
	require.NotErrorIs(t, err, domain.ErrTaskNotFound)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "get task")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTaskRepository_List_NoFilter_ASC(t *testing.T) {
	t.Parallel()

	mock, repo := newMock(t)

	rows := sqlmock.NewRows([]string{"id", "name", "body", "status", "created_at", "updated_at"}).
		AddRow(int64(1), "a", "aa", "NEW", time.Now(), nil).
		AddRow(int64(2), "b", "bb", "COMPLETE", time.Now(), nil)

	mock.ExpectQuery(`SELECT .* FROM tasks\s+ORDER BY id ASC\s+LIMIT \$1`).
		WithArgs(uint64(1000), uint64(0)).
		WillReturnRows(rows)

	tasks, err := repo.List(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, tasks, 2)
	assert.Equal(t, uint64(1), tasks[0].ID)
	assert.Equal(t, uint64(2), tasks[1].ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTaskRepository_List_Desc(t *testing.T) {
	t.Parallel()

	mock, repo := newMock(t)

	rows := sqlmock.NewRows([]string{"id", "name", "body", "status", "created_at", "updated_at"})

	mock.ExpectQuery(`SELECT .* FROM tasks\s+ORDER BY id DESC\s+LIMIT \$1`).
		WithArgs(uint64(1000), uint64(0)).
		WillReturnRows(rows)

	tasks, err := repo.List(context.Background(), &ports.ListTasksFilter{Sort: ports.SortDesc})
	require.NoError(t, err)
	assert.Empty(t, tasks)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTaskRepository_List_ToID_ASC(t *testing.T) {
	t.Parallel()

	mock, repo := newMock(t)

	rows := sqlmock.NewRows([]string{"id", "name", "body", "status", "created_at", "updated_at"}).
		AddRow(int64(5), "a", "aa", "NEW", time.Now(), nil)

	mock.ExpectQuery(`SELECT .* FROM tasks\s+WHERE id < \$1\s+ORDER BY id ASC\s+LIMIT \$2`).
		WithArgs(uint64(10), uint64(1000), uint64(0)).
		WillReturnRows(rows)

	tasks, err := repo.List(context.Background(), &ports.ListTasksFilter{ToID: 10})
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	assert.Equal(t, uint64(5), tasks[0].ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTaskRepository_List_ToID_Desc(t *testing.T) {
	t.Parallel()

	mock, repo := newMock(t)

	rows := sqlmock.NewRows([]string{"id", "name", "body", "status", "created_at", "updated_at"})

	mock.ExpectQuery(`SELECT .* FROM tasks\s+WHERE id < \$1\s+ORDER BY id DESC\s+LIMIT \$2`).
		WithArgs(uint64(10), uint64(1000), uint64(0)).
		WillReturnRows(rows)

	_, err := repo.List(context.Background(), &ports.ListTasksFilter{Sort: ports.SortDesc, ToID: 10})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTaskRepository_List_QueryError(t *testing.T) {
	t.Parallel()

	mock, repo := newMock(t)

	mock.ExpectQuery(`SELECT .* FROM tasks`).
		WillReturnError(errors.New("boom"))

	tasks, err := repo.List(context.Background(), nil)
	require.Error(t, err)
	assert.Nil(t, tasks)
	assert.Contains(t, err.Error(), "list tasks")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTaskRepository_List_ScanError(t *testing.T) {
	t.Parallel()

	mock, repo := newMock(t)

	// id column has a value that cannot scan into uint64
	rows := sqlmock.NewRows([]string{"id", "name", "body", "status", "created_at", "updated_at"}).
		AddRow("not-a-number", "n", "b", "NEW", time.Now(), nil)

	mock.ExpectQuery(`SELECT .* FROM tasks`).
		WithArgs(uint64(1000), uint64(0)).
		WillReturnRows(rows)

	tasks, err := repo.List(context.Background(), nil)
	require.Error(t, err)
	assert.Nil(t, tasks)
	assert.Contains(t, err.Error(), "scan task")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTaskRepository_List_HonoursLimitOffset(t *testing.T) {
	t.Parallel()

	mock, repo := newMock(t)

	rows := sqlmock.NewRows([]string{"id", "name", "body", "status", "created_at", "updated_at"})

	mock.ExpectQuery(`SELECT .* FROM tasks\s+ORDER BY id ASC\s+LIMIT \$1 OFFSET \$2`).
		WithArgs(uint64(25), uint64(50)).
		WillReturnRows(rows)

	_, err := repo.List(context.Background(), &ports.ListTasksFilter{Limit: 25, Offset: 50})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTaskRepository_Count_NoFilter(t *testing.T) {
	t.Parallel()

	mock, repo := newMock(t)

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM tasks$`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(42)))

	total, err := repo.Count(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, uint64(42), total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTaskRepository_Count_WithToID(t *testing.T) {
	t.Parallel()

	mock, repo := newMock(t)

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM tasks WHERE id < \$1`).
		WithArgs(uint64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(7)))

	total, err := repo.Count(context.Background(), &ports.ListTasksFilter{ToID: 10})
	require.NoError(t, err)
	assert.Equal(t, uint64(7), total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTaskRepository_Count_QueryError(t *testing.T) {
	t.Parallel()

	mock, repo := newMock(t)

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM tasks`).
		WillReturnError(errors.New("nope"))

	_, err := repo.Count(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "count tasks")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTaskRepository_List_RowsErr(t *testing.T) {
	t.Parallel()

	mock, repo := newMock(t)

	rows := sqlmock.NewRows([]string{"id", "name", "body", "status", "created_at", "updated_at"}).
		AddRow(int64(1), "n", "b", "NEW", time.Now(), nil).
		RowError(0, errors.New("row iteration failed"))

	mock.ExpectQuery(`SELECT .* FROM tasks`).
		WithArgs(uint64(1000), uint64(0)).
		WillReturnRows(rows)

	tasks, err := repo.List(context.Background(), nil)
	require.Error(t, err)
	assert.Nil(t, tasks)
	assert.Contains(t, err.Error(), "iterate tasks")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTaskRepository_Update_OK(t *testing.T) {
	t.Parallel()

	mock, repo := newMock(t)

	created := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	updated := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	task := &domain.Task{
		ID:        7,
		Name:      "n",
		Body:      "b",
		Status:    domain.TaskStatusInProcess,
		CreatedAt: created,
		UpdatedAt: &updated,
	}

	mock.ExpectExec(regexp.QuoteMeta("UPDATE tasks")).
		WithArgs("n", "b", "IN_PROCESS", &updated, uint64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.Update(context.Background(), task))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTaskRepository_Update_NotFound(t *testing.T) {
	t.Parallel()

	mock, repo := newMock(t)

	mock.ExpectExec(regexp.QuoteMeta("UPDATE tasks")).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.Update(context.Background(), &domain.Task{ID: 999, Status: domain.TaskStatusNew})
	require.ErrorIs(t, err, domain.ErrTaskNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTaskRepository_Update_DBError(t *testing.T) {
	t.Parallel()

	mock, repo := newMock(t)

	mock.ExpectExec(regexp.QuoteMeta("UPDATE tasks")).
		WillReturnError(errors.New("boom"))

	err := repo.Update(context.Background(), &domain.Task{ID: 1})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "update task")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTaskRepository_Delete_OK(t *testing.T) {
	t.Parallel()

	mock, repo := newMock(t)

	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM tasks WHERE id = $1")).
		WithArgs(uint64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.Delete(context.Background(), 7))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTaskRepository_Delete_NotFound(t *testing.T) {
	t.Parallel()

	mock, repo := newMock(t)

	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM tasks")).
		WithArgs(uint64(999)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.Delete(context.Background(), 999)
	require.ErrorIs(t, err, domain.ErrTaskNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTaskRepository_Delete_DBError(t *testing.T) {
	t.Parallel()

	mock, repo := newMock(t)

	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM tasks")).
		WillReturnError(errors.New("nope"))

	err := repo.Delete(context.Background(), 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "delete task")
	require.NoError(t, mock.ExpectationsWereMet())
}
