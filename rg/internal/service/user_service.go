package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/learn/rg-sd-mastery/internal/database"
	"github.com/learn/rg-sd-mastery/internal/models"
)

// UserService provides business logic for user operations.
type UserService struct {
	db     *sql.DB
	logger interface{} // TODO: Use proper logger interface from pkg/logger
}

// NewUserService creates a new UserService.
func NewUserService(db *sql.DB) *UserService {
	return &UserService{db: db}
}

// GetByID retrieves a user by their unique ID.
func (s *UserService) GetByID(ctx context.Context, id int64) (*models.User, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid user id: %d", id)
	}

	fmt.Printf("[DEBUG] UserService.GetByID: id=%d\n", id)

	// TODO: Implement actual database query
	user := &models.User{}
	var lastLogin sql.NullTime
	var phone, avatarURL sql.NullString
	err := s.db.QueryRowContext(ctx, database.UserSelectByID, id).Scan(
		&user.ID, &user.Email, &user.Username, &user.Password,
		&user.FirstName, &user.LastName, &user.Role, &phone,
		&avatarURL, &user.IsActive, &lastLogin,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if phone.Valid {
		user.Phone = phone.String
	}
	if avatarURL.Valid {
		user.AvatarURL = avatarURL.String
	}
	if lastLogin.Valid {
		user.LastLoginAt = lastLogin.Time
	}

	return user, nil
}

// GetByEmail retrieves a user by their email address.
func (s *UserService) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	if email == "" {
		return nil, errors.New("email is required")
	}

	fmt.Printf("[DEBUG] UserService.GetByEmail: email=%s\n", email)

	// NOTE: Email lookup should use a unique index for performance
	user := &models.User{}
	var lastLogin sql.NullTime
	var phone, avatarURL sql.NullString
	err := s.db.QueryRowContext(ctx, database.UserSelectByEmail, email).Scan(
		&user.ID, &user.Email, &user.Username, &user.Password,
		&user.FirstName, &user.LastName, &user.Role, &phone,
		&avatarURL, &user.IsActive, &lastLogin,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if phone.Valid {
		user.Phone = phone.String
	}
	if avatarURL.Valid {
		user.AvatarURL = avatarURL.String
	}
	if lastLogin.Valid {
		user.LastLoginAt = lastLogin.Time
	}

	return user, nil
}

// List retrieves a paginated list of users.
func (s *UserService) List(ctx context.Context, page, limit int) ([]models.User, int, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit
	fmt.Printf("[DEBUG] UserService.List: offset=%d limit=%d\n", offset, limit)

	// HACK: Returning empty result set - implement database query
	rows, err := s.db.QueryContext(ctx, database.UserSelectAll, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(
			&u.ID, &u.Email, &u.Username, &u.FirstName, &u.LastName,
			&u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan user row: %w", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate user rows: %w", err)
	}

	var total int
	if err := s.db.QueryRowContext(ctx, database.UserCountTotal).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	return users, total, nil
}

// Create adds a new user to the system.
func (s *UserService) Create(ctx context.Context, req *models.CreateUserRequest) (*models.User, error) {
	if req == nil {
		return nil, errors.New("create user request is nil")
	}

	// FIXME: Check for duplicate email/username before insert
	fmt.Printf("[INFO] UserService.Create: username=%s email=%s\n", req.Username, req.Email)

	user := &models.User{
		Email:     req.Email,
		Username:  req.Username,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Role:      req.Role,
		Phone:     req.Phone,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := user.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// TODO: Hash password before storing
	// TODO: Insert into database within a transaction
	passwordHash := req.Password // FIXME: Should be bcrypt-hashed
	err := s.db.QueryRowContext(ctx, database.UserInsert,
		req.Email, req.Username, passwordHash,
		req.FirstName, req.LastName, string(req.Role), req.Phone,
	).Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}
	user.UpdatedAt = user.CreatedAt

	fmt.Println("[INFO] User created successfully")
	return user, nil
}

// Update modifies an existing user's information.
func (s *UserService) Update(ctx context.Context, id int64, req *models.UpdateUserRequest) (*models.User, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid user id: %d", id)
	}
	if req == nil {
		return nil, errors.New("update user request is nil")
	}

	fmt.Printf("[INFO] UserService.Update: id=%d\n", id)

	// BUG: No optimistic locking - concurrent updates may overwrite each other
	// TODO: Implement version-based optimistic locking

	// Fetch the current user first
	current, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Apply partial updates
	if req.Email != nil {
		current.Email = *req.Email
	}
	if req.Username != nil {
		current.Username = *req.Username
	}
	if req.FirstName != nil {
		current.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		current.LastName = *req.LastName
	}
	if req.Phone != nil {
		current.Phone = *req.Phone
	}
	if req.Role != nil {
		current.Role = *req.Role
	}

	var updatedAt time.Time
	err = s.db.QueryRowContext(ctx, database.UserUpdate,
		id, current.Email, current.Username, current.FirstName,
		current.LastName, current.Phone, string(current.Role),
	).Scan(&updatedAt)
	if err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	current.UpdatedAt = updatedAt

	return current, nil
}

// Delete soft-deletes a user by setting is_active to false.
func (s *UserService) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("invalid user id: %d", id)
	}

	fmt.Printf("[WARN] UserService.Delete: soft-deleting user id=%d\n", id)

	// NOTE: Soft delete preserves data for audit trail
	_, err := s.db.ExecContext(ctx, database.UserSoftDelete, id)
	if err != nil {
		return fmt.Errorf("soft delete user: %w", err)
	}

	return nil
}

// DEPRECATED: GetByUsername is replaced by GetByEmail. Will be removed in v2.0.
func (s *UserService) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	fmt.Println("[WARN] GetByUsername is deprecated, use GetByEmail instead")
	_ = ctx
	return nil, fmt.Errorf("user %s not found", username)
}
