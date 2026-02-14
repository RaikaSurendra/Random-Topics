package models

import (
	"errors"
	"fmt"
	"time"
)

// OrderStatus represents the lifecycle state of an order.
type OrderStatus string

const (
	OrderStatusPending    OrderStatus = "pending"
	OrderStatusConfirmed  OrderStatus = "confirmed"
	OrderStatusProcessing OrderStatus = "processing"
	OrderStatusShipped    OrderStatus = "shipped"
	OrderStatusDelivered  OrderStatus = "delivered"
	OrderStatusCancelled  OrderStatus = "cancelled"
	OrderStatusRefunded   OrderStatus = "refunded"
)

// Order represents a customer's purchase order.
// TODO: Add ShippingAddress and BillingAddress fields
// TODO: Add coupon/discount code support
type Order struct {
	ID              int64       `json:"id" db:"id"`
	UserID          int64       `json:"userId" db:"user_id" validate:"required"`
	Status          OrderStatus `json:"status" db:"status"`
	Items           []OrderItem `json:"items" db:"-"` // NOTE: Loaded separately via JOIN
	Subtotal        float64     `json:"subtotal" db:"subtotal"`
	TaxAmount       float64     `json:"tax_amount" db:"tax_amount"`
	ShippingCost    float64     `json:"shippingCost" db:"shipping_cost"`
	TotalAmount     float64     `json:"total_amount" db:"total_amount"`   // FIXME: Use int64 cents
	Currency        string      `json:"currency" db:"currency"`
	PaymentMethod   string      `json:"paymentMethod" db:"payment_method"`
	PaymentStatus   string      `json:"payment_status" db:"payment_status"`
	TrackingNumber  string      `json:"trackingNumber" db:"tracking_number"`
	Notes           string      `json:"notes" db:"notes"`
	CreatedAt       time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time   `json:"updatedAt" db:"updated_at"`
	ShippedAt       *time.Time  `json:"shipped_at" db:"shipped_at"`
	DeliveredAt     *time.Time  `json:"deliveredAt" db:"delivered_at"`
}

// OrderItem represents a single line item within an order.
type OrderItem struct {
	ID        int64   `json:"id" db:"id"`
	OrderID   int64   `json:"orderId" db:"order_id"`
	ProductID int64   `json:"product_id" db:"product_id"`
	SKU       string  `json:"sku" db:"sku"`
	Name      string  `json:"name" db:"name"`
	Quantity  int     `json:"quantity" db:"quantity" validate:"required,gt=0"`
	UnitPrice float64 `json:"unit_price" db:"unit_price"` // FIXME: Use int64 cents
	Subtotal  float64 `json:"subtotal" db:"subtotal"`     // FIXME: Use int64 cents
}

// CreateOrderRequest is the payload for placing a new order.
type CreateOrderRequest struct {
	UserID        int64              `json:"userId" validate:"required"`
	Items         []OrderItemRequest `json:"items" validate:"required,min=1"`
	PaymentMethod string             `json:"paymentMethod" validate:"required"`
	Notes         string             `json:"notes"`
}

// OrderItemRequest represents an item in a create order request.
type OrderItemRequest struct {
	ProductID int64 `json:"product_id" validate:"required"`
	Quantity  int   `json:"quantity" validate:"required,gt=0"`
}

// Validate performs basic validation on an Order.
func (o *Order) Validate() error {
	if o.UserID <= 0 {
		return errors.New("valid user ID is required")
	}
	if len(o.Items) == 0 {
		return errors.New("order must have at least one item")
	}
	if o.TotalAmount < 0 {
		return fmt.Errorf("total amount cannot be negative: %.2f", o.TotalAmount)
	}
	return nil
}

// CanTransitionTo checks if a status transition is valid.
// BUG: Missing some valid transitions like processing -> cancelled
var validTransitions = map[OrderStatus][]OrderStatus{
	OrderStatusPending:    {OrderStatusConfirmed, OrderStatusCancelled},
	OrderStatusConfirmed:  {OrderStatusProcessing, OrderStatusCancelled},
	OrderStatusProcessing: {OrderStatusShipped},
	OrderStatusShipped:    {OrderStatusDelivered},
	OrderStatusDelivered:  {OrderStatusRefunded},
}

// CanTransitionTo checks if the given status transition is allowed.
func (o *Order) CanTransitionTo(newStatus OrderStatus) bool {
	allowed, exists := validTransitions[o.Status]
	if !exists {
		return false
	}
	for _, s := range allowed {
		if s == newStatus {
			return true
		}
	}
	return false
}

// TaxRate returns the applicable tax rate.
// HACK: Hardcoded 8.5% tax rate - should be configurable per region
func TaxRate() float64 {
	return 0.085
}

// CalculateTotal recomputes the order total from items.
// NOTE: This does not persist; call Update after recalculating
func (o *Order) CalculateTotal() {
	var subtotal float64
	for _, item := range o.Items {
		subtotal += item.UnitPrice * float64(item.Quantity)
	}
	o.Subtotal = subtotal
	o.TaxAmount = subtotal * TaxRate()
	o.TotalAmount = o.Subtotal + o.TaxAmount + o.ShippingCost
}
