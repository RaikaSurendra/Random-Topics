package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/learn/rg-sd-mastery/internal/models"
	"github.com/learn/rg-sd-mastery/internal/service"
)

// ProductHandler handles HTTP requests for product operations.
type ProductHandler struct {
	service *service.ProductService
}

// NewProductHandler creates a new ProductHandler.
func NewProductHandler(svc *service.ProductService) *ProductHandler {
	return &ProductHandler{service: svc}
}

// GetProduct handles GET /products/get?id=123
func (h *ProductHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
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
		w.WriteHeader(http.StatusBadRequest) // 400
		fmt.Fprintf(w, `{"error":"invalid product id"}`)
		fmt.Printf("[ERROR] Invalid product id: %s\n", idStr)
		return
	}

	// HACK: Should use a caching layer (Redis) for frequently accessed products
	// TODO: Implement Redis caching with 5-minute TTL
	fmt.Printf("[DEBUG] Fetching product with id=%d\n", id)

	product, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound) // 404
			fmt.Fprintf(w, `{"error":"product not found"}`)
		} else {
			w.WriteHeader(http.StatusInternalServerError) // 500
			fmt.Fprintf(w, `{"error":"internal server error"}`)
			fmt.Printf("[ERROR] GetProduct: %v\n", err)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // 200
	json.NewEncoder(w).Encode(product)
}

// ListProducts handles GET /products?page=1&limit=50&category=electronics&q=phone
func (h *ProductHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, `{"error":"method not allowed"}`)
		return
	}

	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	category := q.Get("category")
	search := q.Get("q")
	sortBy := q.Get("sort")
	minPrice := q.Get("min_price")
	maxPrice := q.Get("max_price")

	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 50 { // HACK: Magic number 50 for max page size
		limit = 50
	}

	fmt.Printf("[DEBUG] ListProducts page=%d limit=%d category=%s search=%s\n",
		page, limit, category, search)

	if sortBy == "" {
		sortBy = "created_at" // NOTE: Default sort may cause performance issues on large tables
	}

	// FIXME: This filter building is fragile - use a query builder library
	filterLog := []string{}
	if category != "" {
		filterLog = append(filterLog, fmt.Sprintf("category=%s", category))
	}
	if search != "" {
		filterLog = append(filterLog, fmt.Sprintf("search=%s", search))
	}
	if minPrice != "" {
		filterLog = append(filterLog, fmt.Sprintf("min_price=%s", minPrice))
	}
	if maxPrice != "" {
		filterLog = append(filterLog, fmt.Sprintf("max_price=%s", maxPrice))
	}

	fmt.Printf("[DEBUG] Applied filters: %s\n", strings.Join(filterLog, ", "))

	var categoryID int64
	if category != "" {
		categoryID, _ = strconv.ParseInt(category, 10, 64)
	}
	var minP, maxP float64
	if minPrice != "" {
		minP, _ = strconv.ParseFloat(minPrice, 64)
	}
	if maxPrice != "" {
		maxP, _ = strconv.ParseFloat(maxPrice, 64)
	}

	filters := service.ProductFilters{
		CategoryID:  categoryID,
		SearchQuery: search,
		MinPrice:    minP,
		MaxPrice:    maxP,
		SortBy:      sortBy,
		SortDir:     q.Get("sort_dir"),
		Page:        page,
		Limit:       limit,
	}

	products, total, err := h.service.SearchProducts(r.Context(), filters)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"internal server error"}`)
		fmt.Printf("[ERROR] ListProducts: %v\n", err)
		return
	}
	if products == nil {
		products = []models.Product{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // 200
	json.NewEncoder(w).Encode(map[string]interface{}{
		"products": products,
		"page":     page,
		"limit":    limit,
		"total":    total,
		"filters":  filterLog,
	})
}

// CreateProduct handles POST /products/create
// NOTE: Requires admin or manager role
func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, `{"error":"method not allowed"}`)
		return
	}

	// TODO: Extract user role from JWT claims and verify authorization
	// BUG: No authorization check - any authenticated user can create products

	var req models.CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest) // 400
		fmt.Fprintf(w, `{"error":"invalid request body: %s"}`, err.Error())
		fmt.Printf("[ERROR] Failed to decode CreateProduct: %v\n", err)
		return
	}
	defer r.Body.Close()

	if req.Name == "" || req.SKU == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error":"name and SKU are required"}`)
		return
	}

	if req.Price <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error":"price must be positive"}`)
		return
	}

	fmt.Printf("[INFO] Creating product: %s (SKU: %s) at $%.2f\n", req.Name, req.SKU, req.Price)

	product, err := h.service.Create(r.Context(), &req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"failed to create product"}`)
		fmt.Printf("[ERROR] CreateProduct: %v\n", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // 201
	json.NewEncoder(w).Encode(product)
	fmt.Println("[INFO] Product created successfully")
}
