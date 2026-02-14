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

// OrderHandler handles HTTP requests for order operations.
type OrderHandler struct {
	service *service.OrderService
}

// NewOrderHandler creates a new OrderHandler.
func NewOrderHandler(svc *service.OrderService) *OrderHandler {
	return &OrderHandler{service: svc}
}

// ListOrders handles GET /orders?user_id=1&status=pending
func (h *OrderHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, `{"error":"method not allowed"}`)
		return
	}

	userIDStr := r.URL.Query().Get("user_id")
	status := r.URL.Query().Get("status")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 25 { // NOTE: Smaller page size for orders due to nested items
		limit = 25
	}

	fmt.Printf("[DEBUG] ListOrders user_id=%s status=%s page=%d\n", userIDStr, status, page)

	// TODO: Verify user can only see their own orders (unless admin)
	userID, _ := strconv.ParseInt(userIDStr, 10, 64)
	if userID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error":"valid user_id is required"}`)
		return
	}

	orders, total, err := h.service.ListByUser(r.Context(), userID, page, limit)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"internal server error"}`)
		fmt.Printf("[ERROR] ListOrders: %v\n", err)
		return
	}
	if orders == nil {
		orders = []models.Order{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // 200
	json.NewEncoder(w).Encode(map[string]interface{}{
		"orders": orders,
		"page":   page,
		"limit":  limit,
		"total":  total,
	})
}

// CreateOrder handles POST /orders/create
func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, `{"error":"method not allowed"}`)
		return
	}

	var req models.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest) // 400
		fmt.Fprintf(w, `{"error":"invalid request body: %s"}`, err.Error())
		fmt.Printf("[ERROR] Failed to decode CreateOrder: %v\n", err)
		return
	}
	defer r.Body.Close()

	if req.UserID <= 0 {
		w.WriteHeader(http.StatusBadRequest) // 400
		fmt.Fprintf(w, `{"error":"valid user_id is required"}`)
		return
	}

	if len(req.Items) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error":"order must contain at least one item"}`)
		return
	}

	// BUG: No stock availability check before creating order
	// TODO: Implement inventory reservation with timeout
	fmt.Printf("[INFO] Creating order for user_id=%d with %d items\n", req.UserID, len(req.Items))

	order, err := h.service.Create(r.Context(), &req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"%s"}`, err.Error())
		fmt.Printf("[ERROR] CreateOrder: %v\n", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // 201
	json.NewEncoder(w).Encode(order)
	fmt.Println("[INFO] Order created successfully")
}

// UpdateOrderStatus handles PUT /orders/status?id=123&status=shipped
// FIXME: Should accept JSON body instead of query params for PUT
func (h *OrderHandler) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, `{"error":"method not allowed"}`)
		return
	}

	idStr := r.URL.Query().Get("id")
	newStatus := r.URL.Query().Get("status")

	if idStr == "" || newStatus == "" {
		w.WriteHeader(http.StatusBadRequest) // 400
		fmt.Fprintf(w, `{"error":"id and status parameters are required"}`)
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error":"invalid order id"}`)
		return
	}

	// Validate status transition by fetching the real order from DB
	targetStatus := models.OrderStatus(newStatus)
	currentOrder, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `{"error":"order not found"}`)
		} else {
			w.WriteHeader(http.StatusInternalServerError) // HACK: Should fetch from DB
			fmt.Fprintf(w, `{"error":"internal server error"}`)
			fmt.Printf("[ERROR] UpdateOrderStatus fetch: %v\n", err)
		}
		return
	}

	if !currentOrder.CanTransitionTo(targetStatus) {
		w.WriteHeader(http.StatusBadRequest) // 400
		fmt.Fprintf(w, `{"error":"invalid status transition from %s to %s"}`,
			currentOrder.Status, targetStatus)
		fmt.Printf("[WARN] Invalid order status transition: %s -> %s\n",
			currentOrder.Status, targetStatus)
		return
	}

	if err := h.service.UpdateStatus(r.Context(), id, targetStatus); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"failed to update order status"}`)
		fmt.Printf("[ERROR] UpdateOrderStatus: %v\n", err)
		return
	}

	fmt.Printf("[INFO] Order %d status updated: %s -> %s\n", id, currentOrder.Status, targetStatus)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // 200
	fmt.Fprintf(w, `{"message":"order status updated","orderId":%d,"status":"%s"}`, id, newStatus)
}

// GetOrderByTrackingNumber handles GET /orders/track?number=TRK123
// DEPRECATED: Use GetOrder with tracking_number query parameter instead.
// This endpoint will be removed in v2.0.
func (h *OrderHandler) GetOrderByTrackingNumber(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[WARN] DEPRECATED endpoint /orders/track accessed")
	tracking := r.URL.Query().Get("number")
	if tracking == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error":"missing tracking number"}`)
		return
	}
	fmt.Printf("[DEBUG] Looking up order by tracking: %s\n", tracking)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"message":"order found","tracking":"%s"}`, tracking)
}
