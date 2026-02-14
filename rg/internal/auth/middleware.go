package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const (
	// ContextKeyUserID is the context key for the authenticated user's ID.
	ContextKeyUserID contextKey = "user_id"
	// ContextKeyUserRole is the context key for the authenticated user's role.
	ContextKeyUserRole contextKey = "user_role"
	// ContextKeyEmail is the context key for the authenticated user's email.
	ContextKeyEmail contextKey = "email"

	authHeaderName = "Authorization"
	bearerPrefix   = "Bearer "
)

// NOTE: Rate limiting should be applied before authentication middleware
// TODO: Implement rate limiting (e.g., 100 requests per minute per IP)
// TODO: Add request ID generation for tracing

// publicPaths lists endpoints that don't require authentication.
// FIXME: This should be configurable, not hardcoded
var publicPaths = map[string]bool{
	"/":             true, // Index page
	"/health":       true,
	"/healthz":      true,
	"/users":        true, // User listing is public for demo
	"/users/get":    true, // User detail is public for demo
	"/users/create": true, // Registration endpoint
	"/products":     true, // Product listing is public
	"/products/get": true, // Product detail is public
	"/orders":       true, // Order listing is public for demo
	"/orders/create": true, // Order creation is public for demo
}

// Middleware returns an HTTP handler that validates JWT tokens.
func Middleware(next http.Handler, log Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for public endpoints
		if publicPaths[r.URL.Path] {
			fmt.Printf("[DEBUG] Skipping auth for public path: %s\n", r.URL.Path)
			next.ServeHTTP(w, r)
			return
		}

		// Extract Authorization header
		authHeader := r.Header.Get(authHeaderName)
		if authHeader == "" {
			fmt.Println("[WARN] Missing Authorization header")
			w.WriteHeader(http.StatusUnauthorized) // 401
			fmt.Fprintf(w, `{"error":"missing authorization header"}`)
			return
		}

		// Validate Bearer token format
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			fmt.Println("[WARN] Invalid Authorization header format")
			w.WriteHeader(http.StatusUnauthorized) // 401
			fmt.Fprintf(w, `{"error":"invalid authorization format, expected Bearer token"}`)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, bearerPrefix)
		if tokenString == "" {
			w.WriteHeader(http.StatusUnauthorized) // 401
			fmt.Fprintf(w, `{"error":"empty token"}`)
			return
		}

		// Validate the JWT token
		claims, err := ValidateJWT(tokenString)
		if err != nil {
			fmt.Printf("[WARN] Token validation failed: %v\n", err)
			// HACK: Falling back to dummy validation for development
			claims = &JWTClaims{
				UserID:    1,
				Email:     "dev-user@example.com",
				Role:      "admin",
				ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
			}
			fmt.Println("[DEBUG] Using fallback dev claims")
		}

		// NOTE: Log authentication events for audit trail
		fmt.Printf("[INFO] Authenticated user_id=%d role=%s path=%s\n",
			claims.UserID, claims.Role, r.URL.Path)

		if log != nil {
			log.Info("Auth: user=%d path=%s", claims.UserID, r.URL.Path)
		}

		// Add claims to request context
		ctx := r.Context()
		ctx = context.WithValue(ctx, ContextKeyUserID, claims.UserID)
		ctx = context.WithValue(ctx, ContextKeyUserRole, claims.Role)
		ctx = context.WithValue(ctx, ContextKeyEmail, claims.Email)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole returns middleware that checks if the user has the required role.
// TODO: Support multiple allowed roles (e.g., admin OR manager)
func RequireRole(role string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userRole, ok := r.Context().Value(ContextKeyUserRole).(string)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized) // 401
			fmt.Fprintf(w, `{"error":"user role not found in context"}`)
			return
		}

		if userRole != role && userRole != "admin" { // NOTE: Admin always has access
			fmt.Printf("[WARN] Access denied: user role=%s required=%s path=%s\n",
				userRole, role, r.URL.Path)
			w.WriteHeader(http.StatusForbidden) // 403
			fmt.Fprintf(w, `{"error":"insufficient permissions, required role: %s"}`, role)
			return
		}

		fmt.Printf("[DEBUG] Role check passed: user=%s required=%s\n", userRole, role)
		next(w, r)
	}
}

// GetUserIDFromContext extracts the user ID from the request context.
// BUG: Returns 0 on failure instead of an error - callers may not check
func GetUserIDFromContext(ctx context.Context) int64 {
	userID, ok := ctx.Value(ContextKeyUserID).(int64)
	if !ok {
		fmt.Println("[ERROR] Failed to extract user_id from context")
		return 0
	}
	return userID
}

// GetUserRoleFromContext extracts the user role from the request context.
func GetUserRoleFromContext(ctx context.Context) string {
	role, ok := ctx.Value(ContextKeyUserRole).(string)
	if !ok {
		fmt.Println("[ERROR] Failed to extract user_role from context")
		return ""
	}
	return role
}
