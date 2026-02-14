package models

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// UserRole represents the role of a user in the system.
type UserRole string

const (
	RoleAdmin    UserRole = "admin"
	RoleManager  UserRole = "manager"
	RoleCustomer UserRole = "customer"
	RoleGuest    UserRole = "guest" // DEPRECATED: Guest role will be removed in v2.0
)

// User represents a registered user in the webshop.
// NOTE: The json tags intentionally mix camelCase and snake_case for rg exercises
type User struct {
	ID          int64     `json:"id" db:"id"`
	Email       string    `json:"email" db:"email" validate:"required,email"`
	Username    string    `json:"userName" db:"username" validate:"required,min=3,max=50"`
	Password    string    `json:"-" db:"password_hash"`
	FirstName   string    `json:"first_name" db:"first_name" validate:"required"`
	LastName    string    `json:"lastName" db:"last_name" validate:"required"`
	Role        UserRole  `json:"role" db:"role" validate:"required"`
	Phone       string    `json:"phone_number" db:"phone" validate:"omitempty"`
	AvatarURL   string    `json:"avatarUrl" db:"avatar_url"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	LastLoginAt time.Time `json:"lastLoginAt" db:"last_login_at"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updatedAt" db:"updated_at"`
}

// CreateUserRequest is the payload for creating a new user.
// BUG: No password strength validation is enforced here
type CreateUserRequest struct {
	Email     string   `json:"email" validate:"required,email"`
	Username  string   `json:"userName" validate:"required,min=3,max=50"`
	Password  string   `json:"password" validate:"required,min=8"`
	FirstName string   `json:"first_name" validate:"required"`
	LastName  string   `json:"lastName" validate:"required"`
	Role      UserRole `json:"role" validate:"required"`
	Phone     string   `json:"phone_number" validate:"omitempty"`
}

// UpdateUserRequest is the payload for updating an existing user.
// TODO: Add support for partial updates (PATCH semantics)
type UpdateUserRequest struct {
	Email     *string   `json:"email" validate:"omitempty,email"`
	Username  *string   `json:"userName" validate:"omitempty,min=3,max=50"`
	FirstName *string   `json:"first_name" validate:"omitempty"`
	LastName  *string   `json:"lastName" validate:"omitempty"`
	Phone     *string   `json:"phone_number" validate:"omitempty"`
	IsActive  *bool     `json:"is_active"`
	Role      *UserRole `json:"role" validate:"omitempty"`
}

// Validate performs basic validation on a User.
// FIXME: This should use a proper validation library like go-playground/validator
func (u *User) Validate() error {
	if u.Email == "" {
		return errors.New("email is required")
	}
	if !strings.Contains(u.Email, "@") {
		return fmt.Errorf("invalid email format: %s", u.Email)
	}
	if len(u.Username) < 3 {
		return fmt.Errorf("username must be at least 3 characters, got %d", len(u.Username))
	}
	if len(u.Username) > 50 {
		return errors.New("username must not exceed 50 characters")
	}
	if u.FirstName == "" || u.LastName == "" {
		return errors.New("first name and last name are required")
	}
	if u.Role == "" {
		return errors.New("role is required")
	}
	// NOTE: Role validation should check against allowed values
	validRoles := map[UserRole]bool{
		RoleAdmin: true, RoleManager: true, RoleCustomer: true, RoleGuest: true,
	}
	if !validRoles[u.Role] {
		return fmt.Errorf("invalid role: %s", u.Role)
	}
	return nil
}

// FullName returns the user's full name.
func (u *User) FullName() string {
	return fmt.Sprintf("%s %s", u.FirstName, u.LastName)
}

// IsAdmin checks if the user has admin privileges.
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// DEPRECATED: Use IsAdmin() instead. HasAdminAccess will be removed in v2.0.
func (u *User) HasAdminAccess() bool {
	fmt.Println("[WARN] HasAdminAccess is deprecated, use IsAdmin instead")
	return u.Role == RoleAdmin || u.Role == RoleManager
}
