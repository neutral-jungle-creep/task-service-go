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

func TestRouter_NotFound_EmptyRouter(t *testing.T) {
	t.Parallel()

	r := server.NewRouter()

	req := httptest.NewRequest(http.MethodGet, "/anything", http.NoBody)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestRouter_LowercaseMethodRegistration(t *testing.T) {
	t.Parallel()

	hit := false
	r := server.NewRouter()
	r.Register("get", "/x", func(w http.ResponseWriter, _ *http.Request) {
		hit = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/x", http.NoBody)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, hit, "Register must normalise method to uppercase")
}

func TestRouter_MultipleParams(t *testing.T) {
	t.Parallel()

	var captured map[string]string
	r := server.NewRouter()
	r.Register("GET", "/users/{uid}/posts/{pid}", func(w http.ResponseWriter, req *http.Request) {
		captured = server.RequestParams(req)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/users/u-7/posts/42", http.NoBody)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "u-7", captured["uid"])
	assert.Equal(t, "42", captured["pid"])
}

func TestRouter_RejectsDifferentStaticSegments(t *testing.T) {
	t.Parallel()

	r := server.NewRouter()
	r.Register("GET", "/x/y", func(http.ResponseWriter, *http.Request) {})

	req := httptest.NewRequest(http.MethodGet, "/a/b", http.NoBody)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code,
		"two same-length paths with different static segments must not match")
}

func TestRouter_DifferentSegmentCount(t *testing.T) {
	t.Parallel()

	r := server.NewRouter()
	r.Register("GET", "/items/{id}", func(http.ResponseWriter, *http.Request) {})

	req := httptest.NewRequest(http.MethodGet, "/items/42/extra", http.NoBody)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestRouter_MultipleMethodsSamePath(t *testing.T) {
	t.Parallel()

	r := server.NewRouter()
	r.Register("GET", "/x", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r.Register("POST", "/x", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	getReq := httptest.NewRequest(http.MethodGet, "/x", http.NoBody)
	getRec := httptest.NewRecorder()
	r.ServeHTTP(getRec, getReq)
	assert.Equal(t, http.StatusOK, getRec.Code)

	postReq := httptest.NewRequest(http.MethodPost, "/x", http.NoBody)
	postRec := httptest.NewRecorder()
	r.ServeHTTP(postRec, postReq)
	assert.Equal(t, http.StatusCreated, postRec.Code)
}

func TestRequestParams_NoContext(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/x", http.NoBody)
	assert.Nil(t, server.RequestParams(req))
}
