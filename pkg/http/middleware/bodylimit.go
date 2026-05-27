package middleware

import "net/http"

// MaxBodyBytes wraps r.Body in http.MaxBytesReader so that any handler that
// reads the body will see io.EOF (and the server will respond 413
// "Request Entity Too Large") once limit bytes have been consumed.
// limit <= 0 disables the middleware.
func MaxBodyBytes(limit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if limit <= 0 {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}
