package models

import (
	"errors"
	"fmt"
	"time"
)

// Category represents a product category.
type Category struct {
	ID          int64  `json:"id" db:"id"`
	Name        string `json:"name" db:"name" validate:"required,max=100"`
	Slug        string `json:"slug" db:"slug" validate:"required"`
	Description string `json:"description" db:"description"`
	ParentID    *int64 `json:"parentId" db:"parent_id"` // NOTE: Nullable for top-level categories
	SortOrder   int    `json:"sort_order" db:"sort_order"`
}

// Product represents an item for sale in the webshop.
// FIXME: Price should use int64 (cents) instead of float64 to avoid rounding errors.
// See: https://stackoverflow.com/questions/3730019/why-not-use-double-or-float-to-represent-currency
type Product struct {
	ID            int64     `json:"id" db:"id"`
	SKU           string    `json:"sku" db:"sku" validate:"required,max=50"`
	Name          string    `json:"name" db:"name" validate:"required,min=1,max=255"`
	Description   string    `json:"description" db:"description"`
	Price         float64   `json:"price" db:"price" validate:"required,gt=0"`          // FIXME: Use int64 cents
	ComparePrice  float64   `json:"compare_price" db:"compare_price"`                   // FIXME: Use int64 cents
	CostPrice     float64   `json:"costPrice" db:"cost_price"`                          // FIXME: Use int64 cents
	CategoryID    int64     `json:"categoryId" db:"category_id" validate:"required"`
	StockQuantity int       `json:"stock_quantity" db:"stock_quantity" validate:"gte=0"`
	Weight        float64   `json:"weight" db:"weight"`                                 // in grams
	IsPublished   bool      `json:"is_published" db:"is_published"`
	ImageURL      string    `json:"imageUrl" db:"image_url"`
	Tags          string    `json:"tags" db:"tags"`                                     // HACK: Comma-separated; should be a relation table
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updatedAt" db:"updated_at"`
}

// CreateProductRequest is the payload for adding a new product.
type CreateProductRequest struct {
	SKU           string  `json:"sku" validate:"required,max=50"`
	Name          string  `json:"name" validate:"required,min=1,max=255"`
	Description   string  `json:"description"`
	Price         float64 `json:"price" validate:"required,gt=0"`
	ComparePrice  float64 `json:"compare_price"`
	CategoryID    int64   `json:"categoryId" validate:"required"`
	StockQuantity int     `json:"stock_quantity" validate:"gte=0"`
	Weight        float64 `json:"weight"`
	ImageURL      string  `json:"imageUrl"`
	Tags          string  `json:"tags"`
}

// Validate performs basic validation on a Product.
func (p *Product) Validate() error {
	if p.SKU == "" {
		return errors.New("SKU is required")
	}
	if p.Name == "" {
		return errors.New("product name is required")
	}
	if p.Price <= 0 {
		return fmt.Errorf("price must be positive, got %.2f", p.Price)
	}
	// BUG: No check for Price > ComparePrice which would be illogical
	if p.StockQuantity < 0 {
		return fmt.Errorf("stock quantity cannot be negative, got %d", p.StockQuantity)
	}
	if p.CategoryID <= 0 {
		return errors.New("valid category ID is required")
	}
	return nil
}

// IsOnSale returns true if the product has a compare price higher than the actual price.
func (p *Product) IsOnSale() bool {
	return p.ComparePrice > 0 && p.ComparePrice > p.Price
}

// DiscountPercentage returns the discount percentage if on sale.
// FIXME: Floating point arithmetic may produce imprecise results
func (p *Product) DiscountPercentage() float64 {
	if !p.IsOnSale() {
		return 0
	}
	return ((p.ComparePrice - p.Price) / p.ComparePrice) * 100
}

// InStock returns whether the product is available.
func (p *Product) InStock() bool {
	return p.StockQuantity > 0
}

// DEPRECATED: Use InStock() instead. IsAvailable will be removed in v2.0.
func (p *Product) IsAvailable() bool {
	fmt.Println("[WARN] IsAvailable is deprecated, use InStock instead")
	return p.StockQuantity > 0 && p.IsPublished
}

// FormatPrice returns a human-readable price string.
// HACK: Hardcoded currency symbol; should come from locale settings
func (p *Product) FormatPrice() string {
	return fmt.Sprintf("$%.2f", p.Price)
}
