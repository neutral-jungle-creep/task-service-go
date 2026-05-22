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

// doJSON sends a JSON request, decodes the response into dst (when not nil),
// closes the body and returns the status code. Keeping the request lifecycle
// fully inside this helper lets bodyclose and errcheck see a balanced flow.
func doJSON(t *testing.T, method, url string, body, dst any) int {
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
	defer func() { _ = resp.Body.Close() }()

	if dst != nil {
		require.NoError(t, json.NewDecoder(resp.Body).Decode(dst))
	} else {
		_, _ = io.Copy(io.Discard, resp.Body)
	}

	return resp.StatusCode
}

func TestIntegration_CreateGetList(t *testing.T) {
	env := setupTestEnv(t)
	defer env.close(t)

	base := env.server.URL + "/api/v1/task-service/tasks"

	var created dto.CreateTaskResponse
	status := doJSON(t, http.MethodPost, base, dto.CreateTaskRequest{Name: "first", Body: "do it"}, &created)
	require.Equal(t, http.StatusOK, status)
	require.NotZero(t, created.ID)

	var got dto.GetTaskResponse
	status = doJSON(t, http.MethodGet, fmt.Sprintf("%s/%d", base, created.ID), nil, &got)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, "first", got.Name)
	assert.Equal(t, "do it", got.Body)
	assert.Equal(t, "NEW", got.Status)

	var list dto.ListTasksResponse
	status = doJSON(t, http.MethodGet, base, nil, &list)
	require.Equal(t, http.StatusOK, status)
	assert.GreaterOrEqual(t, list.Total, uint64(1))
	assert.GreaterOrEqual(t, len(list.Items), 1)
}

func TestIntegration_GetTask_NotFound(t *testing.T) {
	env := setupTestEnv(t)
	defer env.close(t)

	status := doJSON(t, http.MethodGet, env.server.URL+"/api/v1/task-service/tasks/999999", nil, nil)
	require.Equal(t, http.StatusNotFound, status)
}

func TestIntegration_GetTask_BadID(t *testing.T) {
	env := setupTestEnv(t)
	defer env.close(t)

	status := doJSON(t, http.MethodGet, env.server.URL+"/api/v1/task-service/tasks/not-a-number", nil, nil)
	require.Equal(t, http.StatusBadRequest, status)
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
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

type concurrentCreateResult struct {
	id     uint64
	status int
	err    error
}

// rawCreate avoids testifylint's go-require check by performing the request
// without require/assert and reporting outcomes through a channel.
func rawCreate(url string, payload dto.CreateTaskRequest) concurrentCreateResult {
	raw, err := json.Marshal(payload)
	if err != nil {
		return concurrentCreateResult{err: err}
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return concurrentCreateResult{err: err}
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return concurrentCreateResult{err: err}
	}
	defer func() { _ = resp.Body.Close() }()

	var c dto.CreateTaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&c); err != nil {
		return concurrentCreateResult{status: resp.StatusCode, err: err}
	}
	return concurrentCreateResult{id: c.ID, status: resp.StatusCode}
}

func TestIntegration_ConcurrentCreates(t *testing.T) {
	env := setupTestEnv(t)
	defer env.close(t)

	base := env.server.URL + "/api/v1/task-service/tasks"
	const n = 10

	results := make(chan concurrentCreateResult, n)
	for i := 0; i < n; i++ {
		go func(i int) {
			results <- rawCreate(base, dto.CreateTaskRequest{
				Name: "task-" + strconv.Itoa(i),
				Body: "body",
			})
		}(i)
	}

	seen := make(map[uint64]struct{}, n)
	for i := 0; i < n; i++ {
		r := <-results
		require.NoError(t, r.err)
		require.Equal(t, http.StatusOK, r.status)
		_, dup := seen[r.id]
		assert.False(t, dup, "duplicate id %d", r.id)
		seen[r.id] = struct{}{}
	}
	assert.Len(t, seen, n)
}
