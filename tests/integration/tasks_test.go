//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"task-service/internal/server/dto"
)

func doJSON(t *testing.T, method, url string, body any) *http.Response {
	t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(context.Background(), method, url, reader)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

func decode(t *testing.T, resp *http.Response, target any) {
	t.Helper()
	defer resp.Body.Close()
	require.NoError(t, json.NewDecoder(resp.Body).Decode(target))
}

func TestIntegration_CreateGetList(t *testing.T) {
	env := setupTestEnv(t)
	defer env.close(t)

	base := env.server.URL + "/api/v1/task-service/tasks"

	resp := doJSON(t, http.MethodPost, base, dto.CreateTaskRequest{Name: "first", Body: "do it"})
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var created dto.CreateTaskResponse
	decode(t, resp, &created)
	require.NotZero(t, created.ID)

	resp = doJSON(t, http.MethodGet, fmt.Sprintf("%s/%d", base, created.ID), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var got dto.GetTaskResponse
	decode(t, resp, &got)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, "first", got.Name)
	assert.Equal(t, "do it", got.Body)
	assert.Equal(t, "NEW", got.Status)

	resp = doJSON(t, http.MethodGet, base, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var list dto.ListTasksResponse
	decode(t, resp, &list)
	assert.GreaterOrEqual(t, list.Total, uint64(1))
	assert.GreaterOrEqual(t, len(list.Items), 1)
}

func TestIntegration_GetTask_NotFound(t *testing.T) {
	env := setupTestEnv(t)
	defer env.close(t)

	resp := doJSON(t, http.MethodGet, env.server.URL+"/api/v1/task-service/tasks/999999", nil)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestIntegration_GetTask_BadID(t *testing.T) {
	env := setupTestEnv(t)
	defer env.close(t)

	resp := doJSON(t, http.MethodGet, env.server.URL+"/api/v1/task-service/tasks/not-a-number", nil)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestIntegration_CreateTask_BadJSON(t *testing.T) {
	env := setupTestEnv(t)
	defer env.close(t)

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		env.server.URL+"/api/v1/task-service/tasks",
		bytes.NewReader([]byte("not json")),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestIntegration_ConcurrentCreates(t *testing.T) {
	env := setupTestEnv(t)
	defer env.close(t)

	base := env.server.URL + "/api/v1/task-service/tasks"
	const n = 10

	ids := make(chan uint64, n)
	for i := 0; i < n; i++ {
		go func(i int) {
			resp := doJSON(t, http.MethodPost, base, dto.CreateTaskRequest{
				Name: "task-" + strconv.Itoa(i),
				Body: "body",
			})
			defer resp.Body.Close()
			require.Equal(t, http.StatusOK, resp.StatusCode)
			var c dto.CreateTaskResponse
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&c))
			ids <- c.ID
		}(i)
	}

	seen := make(map[uint64]struct{}, n)
	for i := 0; i < n; i++ {
		id := <-ids
		_, dup := seen[id]
		assert.False(t, dup, "duplicate id %d", id)
		seen[id] = struct{}{}
	}
	assert.Len(t, seen, n)
}
