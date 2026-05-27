package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// IPRateLimitConfig configures the per-IP rate limiter.
//
// Rate <= 0 disables the middleware altogether (useful for tests / dev).
// Burst is the bucket size — instantaneous bursts above the steady Rate up
// to Burst requests are still allowed.
// TTL evicts per-IP limiter state that has been idle for at least TTL.
type IPRateLimitConfig struct {
	Rate  float64
	Burst int
	TTL   time.Duration
}

// IPRateLimit returns a middleware that rate-limits incoming requests per
// client IP. Replies with 429 Too Many Requests on rejection.
func IPRateLimit(cfg IPRateLimitConfig) func(http.Handler) http.Handler {
	if cfg.Rate <= 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	if cfg.Burst <= 0 {
		cfg.Burst = 1
	}
	if cfg.TTL <= 0 {
		cfg.TTL = 10 * time.Minute
	}

	limiters := &ipLimiterStore{
		buckets: make(map[string]*ipBucket),
		rate:    rate.Limit(cfg.Rate),
		burst:   cfg.Burst,
		ttl:     cfg.TTL,
	}
	go limiters.gc()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)
			if !limiters.get(ip).Allow() {
				w.Header().Set("Retry-After", "1")
				http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

type ipBucket struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type ipLimiterStore struct {
	mu      sync.Mutex
	buckets map[string]*ipBucket
	rate    rate.Limit
	burst   int
	ttl     time.Duration
}

func (s *ipLimiterStore) get(ip string) *rate.Limiter {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, ok := s.buckets[ip]
	if !ok {
		b = &ipBucket{limiter: rate.NewLimiter(s.rate, s.burst)}
		s.buckets[ip] = b
	}
	b.lastSeen = time.Now()
	return b.limiter
}

func (s *ipLimiterStore) gc() {
	ticker := time.NewTicker(s.ttl)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-s.ttl)
		s.mu.Lock()
		for ip, b := range s.buckets {
			if b.lastSeen.Before(cutoff) {
				delete(s.buckets, ip)
			}
		}
		s.mu.Unlock()
	}
}

// clientIP picks the most specific client identifier available.
// X-Forwarded-For wins (when set by a trusted proxy), otherwise the
// peer host from RemoteAddr is used.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// take the left-most entry — the original client
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return trimSpace(xff[:i])
			}
		}
		return trimSpace(xff)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}
