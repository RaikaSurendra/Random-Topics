package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// TODO: Add request ID middleware for distributed tracing
// TODO: Implement rate limiting middleware
// FIXME: CORS middleware does not handle preflight caching properly

// RequestLogger logs incoming HTTP requests with timing information.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// NOTE: Using a response wrapper to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapped, r)

		elapsed := time.Since(start)
		fmt.Printf("[%s] %s %s %d %v\n",
			time.Now().Format("2006-01-02 15:04:05"),
			r.Method,
			r.URL.Path,
			wrapped.statusCode,
			elapsed,
		)
	})
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code before delegating.
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// CORS adds Cross-Origin Resource Sharing headers to responses.
// HACK: AllowedOrigins is hardcoded - should be loaded from config
// TODO: Make allowed origins configurable via environment variable
func CORS(next http.Handler) http.Handler {
	allowedOrigins := map[string]bool{
		"http://localhost:3000":  true, // React dev server
		"http://localhost:5173":  true, // Vite dev server
		"https://webshop.local":  true, // Local HTTPS
		"https://staging.webshop.com": true, // Staging
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if allowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
		w.Header().Set("Access-Control-Max-Age", "86400")           // 24 hours
		w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID, X-Total-Count")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent) // 204
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Recovery recovers from panics and returns a 500 error.
// BUG: Stack trace is logged but not in a structured format
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				fmt.Printf("[FATAL] Panic recovered in %s %s: %v\n", r.Method, r.URL.Path, err)
				http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// ContentType ensures responses have the correct Content-Type header.
func ContentType(contentType string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", contentType)
			next.ServeHTTP(w, r)
		})
	}
}

// SecurityHeaders adds common security headers to all responses.
// TODO: Make these configurable via SecurityConfig struct
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		// NOTE: Referrer-Policy added for privacy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

// StripTrailingSlash removes trailing slashes from request URLs.
func StripTrailingSlash(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && strings.HasSuffix(r.URL.Path, "/") {
			r.URL.Path = strings.TrimRight(r.URL.Path, "/")
		}
		next.ServeHTTP(w, r)
	})
}
