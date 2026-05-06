package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"task-service/pkg/http/server"
)

func TestRouter_StaticPath(t *testing.T) {
	t.Parallel()

	r := server.NewRouter()
	r.Register("GET", "/ping", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", http.NoBody)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTeapot, rec.Code)
}

func TestRouter_PathParam(t *testing.T) {
	t.Parallel()

	var got string
	r := server.NewRouter()
	r.Register("GET", "/items/{id}", func(w http.ResponseWriter, req *http.Request) {
		got = server.RequestParams(req)["id"]
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/items/42", http.NoBody)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "42", got)
}

func TestRouter_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	r := server.NewRouter()
	r.Register("GET", "/x", func(http.ResponseWriter, *http.Request) {})

	req := httptest.NewRequest(http.MethodPost, "/x", http.NoBody)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestRouter_NotFound(t *testing.T) {
	t.Parallel()

	r := server.NewRouter()
	r.Register("GET", "/x", func(http.ResponseWriter, *http.Request) {})

	req := httptest.NewRequest(http.MethodDelete, "/y", http.NoBody)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}
