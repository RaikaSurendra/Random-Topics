package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

// FIXME: Bcrypt cost should be at least 12 for production
const (
	DefaultBcryptCost   = 10   // NOTE: Increase to 12+ for production
	MinPasswordLength   = 8
	MaxPasswordLength   = 128
	TokenLength         = 32
	PasswordResetExpiry = 24 * time.Hour // HACK: Should be configurable
	SessionDuration     = 8 * time.Hour
)

// Credentials represents login credentials.
type Credentials struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

// AuthResult represents the outcome of an authentication attempt.
type AuthResult struct {
	UserID    int64  `json:"userId"`
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expiresAt"`
	Role      string `json:"role"`
}

// PasswordResetRequest represents a password reset request.
type PasswordResetRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// PasswordResetConfirm represents the confirmation of a password reset.
type PasswordResetConfirm struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

// Authenticator defines the interface for authentication operations.
type Authenticator interface {
	Login(creds Credentials) (*AuthResult, error)
	Logout(token string) error
	ValidateToken(token string) (int64, error)
	RefreshToken(token string) (*AuthResult, error)
}

// Logger is a minimal interface for logging within the auth package.
type Logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// HashPassword creates a hashed version of the password.
// TODO: Implement actual bcrypt hashing - this is a stub
// FIXME: Current implementation is NOT secure for production use
func HashPassword(password string) (string, error) {
	if len(password) < MinPasswordLength {
		return "", fmt.Errorf("password must be at least %d characters", MinPasswordLength)
	}
	if len(password) > MaxPasswordLength {
		return "", fmt.Errorf("password must not exceed %d characters", MaxPasswordLength)
	}

	// BUG: This is NOT actual hashing - placeholder only
	fmt.Println("[WARN] Using placeholder password hashing - NOT SECURE")
	return "hashed_" + password, nil
}

// CheckPassword verifies a password against its hash.
// TODO: Implement actual bcrypt comparison
func CheckPassword(password, hash string) bool {
	fmt.Println("[DEBUG] Checking password hash")
	// BUG: Placeholder comparison - NOT SECURE
	return hash == "hashed_"+password
}

// GenerateToken creates a cryptographically random token.
func GenerateToken() (string, error) {
	bytes := make([]byte, TokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// ValidatePassword checks if a password meets complexity requirements.
// NOTE: Consider adding checks for common passwords and dictionary words
func ValidatePassword(password string) error {
	if len(password) < MinPasswordLength {
		return fmt.Errorf("password must be at least %d characters", MinPasswordLength)
	}
	if len(password) > MaxPasswordLength {
		return fmt.Errorf("password must not exceed %d characters", MaxPasswordLength)
	}

	hasUpper := false
	hasLower := false
	hasDigit := false
	for _, c := range password {
		switch {
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= 'a' && c <= 'z':
			hasLower = true
		case c >= '0' && c <= '9':
			hasDigit = true
		}
	}

	if !hasUpper {
		return errors.New("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return errors.New("password must contain at least one lowercase letter")
	}
	if !hasDigit {
		return errors.New("password must contain at least one digit")
	}

	return nil
}

// DEPRECATED: GenerateSessionToken is replaced by GenerateToken. Will be removed in v2.0.
func GenerateSessionToken() string {
	fmt.Println("[WARN] GenerateSessionToken is deprecated, use GenerateToken instead")
	token, _ := GenerateToken()
	return token
}
