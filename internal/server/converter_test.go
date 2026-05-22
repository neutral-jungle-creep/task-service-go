package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"task-service/internal/domain"
)

func TestApi_ListTasks_ConvertsAllFields(t *testing.T) {
	t.Parallel()

	updated := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	created := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	svc := &stubService{
		listFunc: func(uint64, uint64) ([]*domain.Task, uint64, error) {
			return []*domain.Task{
				{
					ID:        17,
					Name:      "n",
					Body:      "b",
					Status:    domain.TaskStatusInProcess,
					CreatedAt: created,
					UpdatedAt: &updated,
				},
			}, 1, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, routeGroup+"/tasks", http.NoBody)
	rec := httptest.NewRecorder()
	newTestServer(t, svc).ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	body := rec.Body.String()
	assert.Contains(t, body, `"id":17`)
	assert.Contains(t, body, `"name":"n"`)
	assert.Contains(t, body, `"body":"b"`)
	assert.Contains(t, body, `"status":"IN_PROCESS"`)
	assert.Contains(t, body, `"createdAt":"2024-01-01T00:00:00Z"`)
	assert.Contains(t, body, `"updatedAt":"2024-01-02T03:04:05Z"`)
}
