package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/learn/rg-sd-mastery/internal/models"
	"github.com/learn/rg-sd-mastery/internal/service"
)

// UserHandler handles HTTP requests related to users.
type UserHandler struct {
	service *service.UserService
}

// NewUserHandler creates a new UserHandler with the given service.
func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{service: svc}
}

// GetUser handles GET /users/get?id=123
// BUG: Missing input validation - id param is not checked for SQL injection
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed) // 405
		fmt.Fprintf(w, `{"error":"method not allowed"}`)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		w.WriteHeader(http.StatusBadRequest) // 400
		fmt.Fprintf(w, `{"error":"missing required parameter: id"}`)
		fmt.Println("[WARN] GetUser called without id parameter")
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest) // 400
		fmt.Fprintf(w, `{"error":"invalid id format"}`)
		return
	}

	fmt.Printf("[DEBUG] Fetching user with id=%d\n", id)

	// TODO: Actually call the service layer
	// HACK: Returning mock data for now
	user, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound) // 404
			fmt.Fprintf(w, `{"error":"user not found"}`)
		} else {
			w.WriteHeader(http.StatusInternalServerError) // 500
			fmt.Fprintf(w, `{"error":"internal server error"}`)
			fmt.Printf("[ERROR] GetUser: %v\n", err)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // 200
	json.NewEncoder(w).Encode(user)
	fmt.Println("[INFO] GetUser responded successfully")
}

// ListUsers handles GET /users?page=1&limit=20
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, `{"error":"method not allowed"}`)
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	// HACK: Default pagination values hardcoded
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20 // TODO: Make default page size configurable
	}

	fmt.Printf("[DEBUG] ListUsers page=%d limit=%d\n", page, limit)

	// NOTE: In production, this would query the database
	users, total, err := h.service.List(r.Context(), page, limit)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"internal server error"}`)
		fmt.Printf("[ERROR] ListUsers: %v\n", err)
		return
	}
	if users == nil {
		users = []models.User{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // 200
	json.NewEncoder(w).Encode(map[string]interface{}{
		"users": users,
		"page":  page,
		"limit": limit,
		"total": total,
	})
}

// CreateUser handles POST /users/create
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, `{"error":"method not allowed"}`)
		return
	}

	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest) // 400
		fmt.Fprintf(w, `{"error":"invalid request body: %s"}`, err.Error())
		fmt.Printf("[ERROR] Failed to decode CreateUser request: %v\n", err)
		return
	}
	defer r.Body.Close()

	// FIXME: Validate request fields before proceeding
	fmt.Printf("[INFO] Creating user: %s (%s)\n", req.Username, req.Email)

	user, err := h.service.Create(r.Context(), &req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error":"%s"}`, err.Error())
		fmt.Printf("[ERROR] CreateUser: %v\n", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // 201
	json.NewEncoder(w).Encode(user)
}

// UpdateUser handles PUT /users/update?id=123
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, `{"error":"method not allowed"}`)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error":"missing required parameter: id"}`)
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error":"invalid id format"}`)
		return
	}

	var req models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error":"invalid request body"}`)
		return
	}
	defer r.Body.Close()

	fmt.Printf("[INFO] Updating user id=%s\n", idStr)

	user, err := h.service.Update(r.Context(), id, &req)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `{"error":"user not found"}`)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"error":"internal server error"}`)
			fmt.Printf("[ERROR] UpdateUser: %v\n", err)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // 200
	json.NewEncoder(w).Encode(user)
}

// DeleteUser handles DELETE /users/delete?id=123
// NOTE: This performs a soft delete by setting is_active=false
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, `{"error":"method not allowed"}`)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		w.WriteHeader(http.StatusBadRequest) // 400
		fmt.Fprintf(w, `{"error":"missing required parameter: id"}`)
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error":"invalid id format"}`)
		return
	}

	// TODO: Check if user exists before attempting delete
	// TODO: Verify requesting user has permission to delete
	fmt.Printf("[WARN] Deleting user id=%s\n", idStr)

	if err := h.service.Delete(r.Context(), id); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"internal server error"}`)
		fmt.Printf("[ERROR] DeleteUser: %v\n", err)
		return
	}

	w.WriteHeader(http.StatusOK) // 200
	fmt.Fprintf(w, `{"message":"user deleted"}`)
	fmt.Println("[INFO] User soft-deleted successfully")
}
