package middleware_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"task-service/pkg/http/middleware"
)

func readingHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func TestMaxBodyBytes_AllowsSmallBody(t *testing.T) {
	t.Parallel()

	h := middleware.MaxBodyBytes(100)(readingHandler())

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", strings.NewReader("hello"))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestMaxBodyBytes_RejectsOversize(t *testing.T) {
	t.Parallel()

	h := middleware.MaxBodyBytes(8)(readingHandler())

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/",
		strings.NewReader(strings.Repeat("x", 100)))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	require.NotEqual(t, http.StatusOK, rec.Code, "oversize body must not reach handler success path")
}

func TestMaxBodyBytes_ZeroDisablesMiddleware(t *testing.T) {
	t.Parallel()

	called := false
	h := middleware.MaxBodyBytes(0)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/",
		strings.NewReader(strings.Repeat("x", 1024)))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, called)
}
