package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"task-service/pkg/http/middleware"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestIPRateLimit_Disabled_WhenRateZero(t *testing.T) {
	t.Parallel()

	h := middleware.IPRateLimit(middleware.IPRateLimitConfig{Rate: 0})(okHandler())

	for i := 0; i < 100; i++ {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody)
		req.RemoteAddr = "1.2.3.4:5678"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code, "request %d should pass when limiter is disabled", i)
	}
}

func TestIPRateLimit_BlocksAfterBurst(t *testing.T) {
	t.Parallel()

	h := middleware.IPRateLimit(middleware.IPRateLimitConfig{
		Rate:  1,
		Burst: 3,
		TTL:   time.Minute,
	})(okHandler())

	send := func() int {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody)
		req.RemoteAddr = "1.2.3.4:5678"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}

	// burst=3 → first three pass immediately
	assert.Equal(t, http.StatusOK, send())
	assert.Equal(t, http.StatusOK, send())
	assert.Equal(t, http.StatusOK, send())

	// fourth instantly → 429
	assert.Equal(t, http.StatusTooManyRequests, send())
}

func TestIPRateLimit_SeparateIPsHaveSeparateBuckets(t *testing.T) {
	t.Parallel()

	h := middleware.IPRateLimit(middleware.IPRateLimitConfig{
		Rate:  1,
		Burst: 1,
		TTL:   time.Minute,
	})(okHandler())

	send := func(ip string) int {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody)
		req.RemoteAddr = ip + ":1000"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}

	assert.Equal(t, http.StatusOK, send("1.1.1.1"))
	assert.Equal(t, http.StatusTooManyRequests, send("1.1.1.1"), "second from same IP exhausts the bucket")
	assert.Equal(t, http.StatusOK, send("2.2.2.2"), "different IP has its own bucket")
}

func TestIPRateLimit_HonoursXForwardedFor(t *testing.T) {
	t.Parallel()

	h := middleware.IPRateLimit(middleware.IPRateLimitConfig{
		Rate:  1,
		Burst: 1,
		TTL:   time.Minute,
	})(okHandler())

	send := func(xff string) int {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody)
		req.RemoteAddr = "10.0.0.1:1000" // proxy address, ignored when XFF is present
		req.Header.Set("X-Forwarded-For", xff)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}

	assert.Equal(t, http.StatusOK, send("203.0.113.5"))
	assert.Equal(t, http.StatusTooManyRequests, send("203.0.113.5, 10.0.0.1"),
		"left-most XFF token must be used as the client identity")
}
