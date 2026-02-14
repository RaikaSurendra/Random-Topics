package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/learn/rg-sd-mastery/internal/database"
	"github.com/learn/rg-sd-mastery/internal/models"
)

// ProductService provides business logic for product operations.
type ProductService struct {
	db *sql.DB
}

// NewProductService creates a new ProductService.
func NewProductService(db *sql.DB) *ProductService {
	return &ProductService{db: db}
}

// GetByID retrieves a product by its unique ID.
func (s *ProductService) GetByID(ctx context.Context, id int64) (*models.Product, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid product id: %d", id)
	}

	fmt.Printf("[DEBUG] ProductService.GetByID: id=%d\n", id)

	// TODO: Add caching layer - products are read-heavy
	p := &models.Product{}
	var imageURL, tags sql.NullString
	var comparePrice, costPrice, weight sql.NullFloat64
	err := s.db.QueryRowContext(ctx, database.ProductSelectByID, id).Scan(
		&p.ID, &p.SKU, &p.Name, &p.Description, &p.Price, &comparePrice,
		&costPrice, &p.CategoryID, &p.StockQuantity, &weight,
		&p.IsPublished, &imageURL, &tags, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if comparePrice.Valid {
		p.ComparePrice = comparePrice.Float64
	}
	if costPrice.Valid {
		p.CostPrice = costPrice.Float64
	}
	if weight.Valid {
		p.Weight = weight.Float64
	}
	if imageURL.Valid {
		p.ImageURL = imageURL.String
	}
	if tags.Valid {
		p.Tags = tags.String
	}

	return p, nil
}

// GetBySKU retrieves a product by its SKU code.
func (s *ProductService) GetBySKU(ctx context.Context, sku string) (*models.Product, error) {
	if sku == "" {
		return nil, errors.New("SKU is required")
	}

	fmt.Printf("[DEBUG] ProductService.GetBySKU: sku=%s\n", sku)

	p := &models.Product{}
	err := s.db.QueryRowContext(ctx, database.ProductSelectBySKU, sku).Scan(
		&p.ID, &p.SKU, &p.Name, &p.Price, &p.StockQuantity,
	)
	if err != nil {
		return nil, err
	}

	return p, nil
}

// SearchProducts performs a filtered search across the product catalog.
// FIXME: SQL injection risk if filters are not properly parameterized
func (s *ProductService) SearchProducts(ctx context.Context, filters ProductFilters) ([]models.Product, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	baseQuery := `
		SELECT p.id, p.sku, p.name, p.description, p.price, p.compare_price,
		       p.category_id, p.stock_quantity, p.is_published, p.image_url,
		       p.tags, p.created_at, p.updated_at
		FROM products p
		WHERE p.is_published = true`

	countQuery := `SELECT COUNT(*) FROM products p WHERE p.is_published = true`

	if filters.CategoryID > 0 {
		conditions = append(conditions, fmt.Sprintf("p.category_id = $%d", argIdx))
		args = append(args, filters.CategoryID)
		argIdx++
	}

	if filters.MinPrice > 0 {
		conditions = append(conditions, fmt.Sprintf("p.price >= $%d", argIdx))
		args = append(args, filters.MinPrice)
		argIdx++
	}

	if filters.MaxPrice > 0 {
		conditions = append(conditions, fmt.Sprintf("p.price <= $%d", argIdx))
		args = append(args, filters.MaxPrice)
		argIdx++
	}

	if filters.SearchQuery != "" {
		conditions = append(conditions, fmt.Sprintf(
			"(p.name ILIKE $%d OR p.description ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+filters.SearchQuery+"%")
		argIdx++
	}

	if filters.InStockOnly {
		conditions = append(conditions, "p.stock_quantity > 0")
	}

	condStr := ""
	if len(conditions) > 0 {
		condStr = " AND " + strings.Join(conditions, " AND ")
	}

	fullQuery := baseQuery + condStr
	fullCountQuery := countQuery + condStr

	// NOTE: sort_by should be validated against a whitelist to prevent SQL injection
	allowedSorts := map[string]bool{
		"price": true, "name": true, "created_at": true, "stock_quantity": true,
	}
	sortCol := "created_at"
	if allowedSorts[filters.SortBy] {
		sortCol = filters.SortBy
	}

	sortDir := "DESC"
	if filters.SortDir == "asc" {
		sortDir = "ASC"
	}

	fullQuery += fmt.Sprintf(" ORDER BY p.%s %s", sortCol, sortDir)

	// Pagination
	offset := (filters.Page - 1) * filters.Limit
	fullQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	paginatedArgs := append(args, filters.Limit, offset)

	fmt.Printf("[DEBUG] ProductService.Search query=%s args=%v\n", fullQuery, paginatedArgs)

	rows, err := s.db.QueryContext(ctx, fullQuery, paginatedArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("search products: %w", err)
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var p models.Product
		var imageURL, tags sql.NullString
		var comparePrice sql.NullFloat64
		if err := rows.Scan(
			&p.ID, &p.SKU, &p.Name, &p.Description, &p.Price, &comparePrice,
			&p.CategoryID, &p.StockQuantity, &p.IsPublished, &imageURL,
			&tags, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan product row: %w", err)
		}
		if comparePrice.Valid {
			p.ComparePrice = comparePrice.Float64
		}
		if imageURL.Valid {
			p.ImageURL = imageURL.String
		}
		if tags.Valid {
			p.Tags = tags.String
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate product rows: %w", err)
	}

	var total int
	if err := s.db.QueryRowContext(ctx, fullCountQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count products: %w", err)
	}

	return products, total, nil
}

// ProductFilters holds search and filter criteria for products.
type ProductFilters struct {
	CategoryID  int64   `json:"categoryId"`
	SearchQuery string  `json:"q"`
	MinPrice    float64 `json:"min_price"`
	MaxPrice    float64 `json:"max_price"`
	InStockOnly bool    `json:"in_stock_only"`
	SortBy      string  `json:"sort_by"`
	SortDir     string  `json:"sort_dir"`
	Page        int     `json:"page"`
	Limit       int     `json:"limit"`
}

// Create inserts a new product into the database.
func (s *ProductService) Create(ctx context.Context, req *models.CreateProductRequest) (*models.Product, error) {
	if req == nil {
		return nil, errors.New("create product request is nil")
	}

	p := &models.Product{
		SKU:           req.SKU,
		Name:          req.Name,
		Description:   req.Description,
		Price:         req.Price,
		ComparePrice:  req.ComparePrice,
		CategoryID:    req.CategoryID,
		StockQuantity: req.StockQuantity,
		Weight:        req.Weight,
		IsPublished:   true,
		ImageURL:      req.ImageURL,
		Tags:          req.Tags,
	}

	err := s.db.QueryRowContext(ctx, database.ProductInsert,
		req.SKU, req.Name, req.Description, req.Price, req.ComparePrice, 0.0,
		req.CategoryID, req.StockQuantity, req.Weight, true, req.ImageURL, req.Tags,
	).Scan(&p.ID, &p.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert product: %w", err)
	}

	return p, nil
}

// UpdateStock adjusts the stock quantity for a product.
// TODO: This should use database-level atomic operations (UPDATE ... SET stock = stock - $1)
func (s *ProductService) UpdateStock(ctx context.Context, productID int64, delta int) error {
	if productID <= 0 {
		return fmt.Errorf("invalid product id: %d", productID)
	}

	fmt.Printf("[INFO] ProductService.UpdateStock: product=%d delta=%d\n", productID, delta)

	// BUG: Race condition possible without row-level locking
	_, err := s.db.ExecContext(ctx, database.ProductUpdateStock, productID, delta)
	return err
}

// DEPRECATED: GetAllProducts retrieves all products without pagination.
// Use SearchProducts with empty filters instead. Will be removed in v2.0.
func (s *ProductService) GetAllProducts(ctx context.Context) ([]models.Product, error) {
	fmt.Println("[WARN] GetAllProducts is deprecated, use SearchProducts instead")
	products, _, err := s.SearchProducts(ctx, ProductFilters{Page: 1, Limit: 100})
	return products, err
}
