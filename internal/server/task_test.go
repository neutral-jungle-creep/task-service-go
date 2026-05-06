package server_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"task-service/internal/domain"
	"task-service/internal/server"
	"task-service/internal/server/dto"
)

type stubService struct {
	createFunc func(*domain.Task) (uint64, error)
	listFunc   func() ([]*domain.Task, error)
	getFunc    func(uint64) (*domain.Task, error)
}

func (s *stubService) Create(t *domain.Task) (uint64, error) {
	if s.createFunc != nil {
		return s.createFunc(t)
	}
	return 1, nil
}

func (s *stubService) List() ([]*domain.Task, error) {
	if s.listFunc != nil {
		return s.listFunc()
	}
	return nil, nil
}

func (s *stubService) Get(id uint64) (*domain.Task, error) {
	if s.getFunc != nil {
		return s.getFunc(id)
	}
	return &domain.Task{}, nil
}

const routeGroup = "/api/v1/task-service"

func newTestServer(t *testing.T, svc *stubService) http.Handler {
	t.Helper()
	api := server.NewAPI(svc)
	return api.InitRoutes(routeGroup)
}

func TestApi_CreateTask_OK(t *testing.T) {
	t.Parallel()

	svc := &stubService{
		createFunc: func(task *domain.Task) (uint64, error) {
			assert.Equal(t, "n", task.Name)
			assert.Equal(t, "b", task.Body)
			return 42, nil
		},
	}

	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		routeGroup+"/tasks",
		strings.NewReader(`{"name":"n","body":"b"}`),
	)
	rec := httptest.NewRecorder()
	newTestServer(t, svc).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp dto.CreateTaskResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, uint64(42), resp.ID)
}

func TestApi_CreateTask_BadJSON(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		routeGroup+"/tasks",
		strings.NewReader(`not-json`),
	)
	rec := httptest.NewRecorder()
	newTestServer(t, &stubService{}).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestApi_CreateTask_ServiceError(t *testing.T) {
	t.Parallel()

	svc := &stubService{
		createFunc: func(*domain.Task) (uint64, error) {
			return 0, errors.New("db down")
		},
	}

	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		routeGroup+"/tasks",
		strings.NewReader(`{"name":"n","body":"b"}`),
	)
	rec := httptest.NewRecorder()
	newTestServer(t, svc).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestApi_ListTasks_OK(t *testing.T) {
	t.Parallel()

	svc := &stubService{
		listFunc: func() ([]*domain.Task, error) {
			return []*domain.Task{{ID: 1, Name: "a"}, {ID: 2, Name: "b"}}, nil
		},
	}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, routeGroup+"/tasks", http.NoBody)
	rec := httptest.NewRecorder()
	newTestServer(t, svc).ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp dto.ListTasksResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, uint64(2), resp.Total)
	assert.Len(t, resp.Items, 2)
}

func TestApi_ListTasks_ServiceError(t *testing.T) {
	t.Parallel()

	svc := &stubService{
		listFunc: func() ([]*domain.Task, error) { return nil, errors.New("oops") },
	}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, routeGroup+"/tasks", http.NoBody)
	rec := httptest.NewRecorder()
	newTestServer(t, svc).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestApi_GetTask_OK(t *testing.T) {
	t.Parallel()

	svc := &stubService{
		getFunc: func(id uint64) (*domain.Task, error) {
			return &domain.Task{ID: id, Name: "x", Body: "y", Status: domain.TaskStatusNew}, nil
		},
	}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, routeGroup+"/tasks/7", http.NoBody)
	rec := httptest.NewRecorder()
	newTestServer(t, svc).ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp dto.GetTaskResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, uint64(7), resp.ID)
	assert.Equal(t, "x", resp.Name)
	assert.Equal(t, string(domain.TaskStatusNew), resp.Status)
}

func TestApi_GetTask_NotFound(t *testing.T) {
	t.Parallel()

	svc := &stubService{
		getFunc: func(uint64) (*domain.Task, error) { return &domain.Task{}, nil },
	}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, routeGroup+"/tasks/999", http.NoBody)
	rec := httptest.NewRecorder()
	newTestServer(t, svc).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestApi_GetTask_BadID(t *testing.T) {
	t.Parallel()

	getCalled := false
	svc := &stubService{
		getFunc: func(uint64) (*domain.Task, error) {
			getCalled = true
			return &domain.Task{}, nil
		},
	}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, routeGroup+"/tasks/not-a-number", http.NoBody)
	rec := httptest.NewRecorder()
	newTestServer(t, svc).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.False(t, getCalled, "service.Get must not be called when id parsing fails")
}
