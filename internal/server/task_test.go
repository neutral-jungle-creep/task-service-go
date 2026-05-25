package server_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"task-service/internal/domain"
	"task-service/internal/ports"
	"task-service/internal/server"
	"task-service/internal/server/dto"
	"task-service/pkg/http/protocol"
)

type silentLogger struct{}

func (silentLogger) Error(string, error) {}

type stubService struct {
	createFunc func(context.Context, *domain.Task) (uint64, error)
	listFunc   func(ctx context.Context, limit, offset uint64) ([]*domain.Task, uint64, error)
	getFunc    func(context.Context, uint64) (*domain.Task, error)
	updateFunc func(context.Context, uint64, ports.UpdateTaskParams) (*domain.Task, error)
	deleteFunc func(context.Context, uint64) error
}

func (s *stubService) Create(ctx context.Context, t *domain.Task) (uint64, error) {
	if s.createFunc != nil {
		return s.createFunc(ctx, t)
	}
	return 1, nil
}

func (s *stubService) List(ctx context.Context, limit, offset uint64) ([]*domain.Task, uint64, error) {
	if s.listFunc != nil {
		return s.listFunc(ctx, limit, offset)
	}
	return nil, 0, nil
}

func (s *stubService) Get(ctx context.Context, id uint64) (*domain.Task, error) {
	if s.getFunc != nil {
		return s.getFunc(ctx, id)
	}
	return &domain.Task{}, nil
}

func (s *stubService) Update(ctx context.Context, id uint64, f ports.UpdateTaskParams) (*domain.Task, error) {
	if s.updateFunc != nil {
		return s.updateFunc(ctx, id, f)
	}
	return &domain.Task{}, nil
}

func (s *stubService) Delete(ctx context.Context, id uint64) error {
	if s.deleteFunc != nil {
		return s.deleteFunc(ctx, id)
	}
	return nil
}

const routeGroup = "/api/v1/task-service"

func newTestServer(t *testing.T, svc *stubService) http.Handler {
	t.Helper()
	rh := protocol.NewResponseHandler(
		silentLogger{},
		protocol.WithValidation(validator.New(validator.WithRequiredStructEnabled())),
	)
	api := server.NewAPI(rh, svc)
	return api.InitRoutes(routeGroup)
}

func TestApi_CreateTask_OK(t *testing.T) {
	t.Parallel()

	svc := &stubService{
		createFunc: func(_ context.Context, task *domain.Task) (uint64, error) {
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

func TestApi_CreateTask_EmptyName(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		routeGroup+"/tasks",
		strings.NewReader(`{"name":"","body":"b"}`),
	)
	rec := httptest.NewRecorder()
	newTestServer(t, &stubService{}).ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Name")
}

func TestApi_CreateTask_EmptyBody(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		routeGroup+"/tasks",
		strings.NewReader(`{"name":"n","body":""}`),
	)
	rec := httptest.NewRecorder()
	newTestServer(t, &stubService{}).ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Body")
}

func TestApi_CreateTask_NameTooLong(t *testing.T) {
	t.Parallel()

	longName := strings.Repeat("x", 256)
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		routeGroup+"/tasks",
		strings.NewReader(`{"name":"`+longName+`","body":"b"}`),
	)
	rec := httptest.NewRecorder()
	newTestServer(t, &stubService{}).ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Name")
}

func TestApi_CreateTask_ServiceError(t *testing.T) {
	t.Parallel()

	svc := &stubService{
		createFunc: func(context.Context, *domain.Task) (uint64, error) {
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

func TestApi_ListTasks_OK_DefaultPagination(t *testing.T) {
	t.Parallel()

	svc := &stubService{
		listFunc: func(_ context.Context, limit, offset uint64) ([]*domain.Task, uint64, error) {
			assert.Equal(t, uint64(50), limit)
			assert.Equal(t, uint64(0), offset)
			return []*domain.Task{{ID: 1, Name: "a"}, {ID: 2, Name: "b"}}, 17, nil
		},
	}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, routeGroup+"/tasks", http.NoBody)
	rec := httptest.NewRecorder()
	newTestServer(t, svc).ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp dto.ListTasksResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, uint64(17), resp.Total)
	assert.Equal(t, uint64(50), resp.Limit)
	assert.Equal(t, uint64(0), resp.Offset)
	assert.Len(t, resp.Items, 2)
}

func TestApi_ListTasks_OK_CustomPagination(t *testing.T) {
	t.Parallel()

	svc := &stubService{
		listFunc: func(_ context.Context, limit, offset uint64) ([]*domain.Task, uint64, error) {
			assert.Equal(t, uint64(10), limit)
			assert.Equal(t, uint64(20), offset)
			return []*domain.Task{}, 30, nil
		},
	}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet,
		routeGroup+"/tasks?limit=10&offset=20", http.NoBody)
	rec := httptest.NewRecorder()
	newTestServer(t, svc).ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp dto.ListTasksResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, uint64(10), resp.Limit)
	assert.Equal(t, uint64(20), resp.Offset)
}

func TestApi_ListTasks_BadLimit(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"non-numeric":   "abc",
		"zero":          "0",
		"above max 500": "501",
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet,
				routeGroup+"/tasks?limit="+raw, http.NoBody)
			rec := httptest.NewRecorder()
			newTestServer(t, &stubService{}).ServeHTTP(rec, req)
			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

func TestApi_ListTasks_BadOffset(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet,
		routeGroup+"/tasks?offset=-1", http.NoBody)
	rec := httptest.NewRecorder()
	newTestServer(t, &stubService{}).ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestApi_ListTasks_ServiceError(t *testing.T) {
	t.Parallel()

	svc := &stubService{
		listFunc: func(context.Context, uint64, uint64) ([]*domain.Task, uint64, error) {
			return nil, 0, errors.New("oops")
		},
	}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, routeGroup+"/tasks", http.NoBody)
	rec := httptest.NewRecorder()
	newTestServer(t, svc).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestApi_GetTask_OK(t *testing.T) {
	t.Parallel()

	svc := &stubService{
		getFunc: func(_ context.Context, id uint64) (*domain.Task, error) {
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
	assert.Equal(t, domain.TaskStatusNew.String(), resp.Status)
}

func TestApi_GetTask_NotFound(t *testing.T) {
	t.Parallel()

	svc := &stubService{
		getFunc: func(context.Context, uint64) (*domain.Task, error) { return nil, domain.ErrTaskNotFound },
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
		getFunc: func(context.Context, uint64) (*domain.Task, error) {
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

func TestApi_UpdateTask_OK(t *testing.T) {
	t.Parallel()

	svc := &stubService{
		updateFunc: func(_ context.Context, id uint64, f ports.UpdateTaskParams) (*domain.Task, error) {
			require.NotNil(t, f.Name)
			assert.Equal(t, "renamed", *f.Name)
			require.NotNil(t, f.Status)
			assert.Equal(t, domain.TaskStatusInProcess.String(), *f.Status)
			return &domain.Task{ID: id, Name: *f.Name, Status: domain.TaskStatus(*f.Status)}, nil
		},
	}

	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPatch,
		routeGroup+"/tasks/7",
		strings.NewReader(`{"name":"renamed","status":"IN_PROCESS"}`),
	)
	rec := httptest.NewRecorder()
	newTestServer(t, svc).ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp dto.GetTaskResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, uint64(7), resp.ID)
	assert.Equal(t, "renamed", resp.Name)
	assert.Equal(t, "IN_PROCESS", resp.Status)
}

func TestApi_UpdateTask_NotFound(t *testing.T) {
	t.Parallel()

	svc := &stubService{
		updateFunc: func(context.Context, uint64, ports.UpdateTaskParams) (*domain.Task, error) {
			return nil, domain.ErrTaskNotFound
		},
	}
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPatch,
		routeGroup+"/tasks/999", strings.NewReader(`{"name":"x"}`))
	rec := httptest.NewRecorder()
	newTestServer(t, svc).ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestApi_UpdateTask_InvalidTransition(t *testing.T) {
	t.Parallel()

	svc := &stubService{
		updateFunc: func(context.Context, uint64, ports.UpdateTaskParams) (*domain.Task, error) {
			return nil, domain.ErrInvalidStatusTransition
		},
	}
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPatch,
		routeGroup+"/tasks/1", strings.NewReader(`{"status":"NEW"}`))
	rec := httptest.NewRecorder()
	newTestServer(t, svc).ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestApi_UpdateTask_BadStatusEnum(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPatch,
		routeGroup+"/tasks/1", strings.NewReader(`{"status":"BOGUS"}`))
	rec := httptest.NewRecorder()
	newTestServer(t, &stubService{}).ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code, "validator must reject unknown status")
}

func TestApi_UpdateTask_BadJSON(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPatch,
		routeGroup+"/tasks/1", strings.NewReader(`not-json`))
	rec := httptest.NewRecorder()
	newTestServer(t, &stubService{}).ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestApi_UpdateTask_BadID(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPatch,
		routeGroup+"/tasks/not-a-number", strings.NewReader(`{"name":"x"}`))
	rec := httptest.NewRecorder()
	newTestServer(t, &stubService{}).ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestApi_DeleteTask_OK(t *testing.T) {
	t.Parallel()

	deletedID := uint64(0)
	svc := &stubService{
		deleteFunc: func(_ context.Context, id uint64) error { deletedID = id; return nil },
	}
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete,
		routeGroup+"/tasks/42", http.NoBody)
	rec := httptest.NewRecorder()
	newTestServer(t, svc).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, uint64(42), deletedID)
}

func TestApi_DeleteTask_NotFound(t *testing.T) {
	t.Parallel()

	svc := &stubService{
		deleteFunc: func(context.Context, uint64) error { return domain.ErrTaskNotFound },
	}
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete,
		routeGroup+"/tasks/999", http.NoBody)
	rec := httptest.NewRecorder()
	newTestServer(t, svc).ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestApi_DeleteTask_BadID(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete,
		routeGroup+"/tasks/not-a-number", http.NoBody)
	rec := httptest.NewRecorder()
	newTestServer(t, &stubService{}).ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
