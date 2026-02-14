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

// OrderService provides business logic for order operations.
type OrderService struct {
	db               *sql.DB
	productService   *ProductService
	maxItemsPerOrder int // HACK: Hardcoded limit
}

// NewOrderService creates a new OrderService.
func NewOrderService(db *sql.DB, ps *ProductService) *OrderService {
	return &OrderService{
		db:               db,
		productService:   ps,
		maxItemsPerOrder: 50, // TODO: Make configurable via Config
	}
}

// GetByID retrieves an order by its unique ID.
func (s *OrderService) GetByID(ctx context.Context, id int64) (*models.Order, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid order id: %d", id)
	}

	fmt.Printf("[DEBUG] OrderService.GetByID: id=%d\n", id)

	order := &models.Order{}
	var trackingNumber, notes, paymentMethod sql.NullString
	var shippedAt, deliveredAt sql.NullTime
	err := s.db.QueryRowContext(ctx, database.OrderSelectByID, id).Scan(
		&order.ID, &order.UserID, &order.Status, &order.Subtotal, &order.TaxAmount,
		&order.ShippingCost, &order.TotalAmount, &order.Currency, &paymentMethod,
		&order.PaymentStatus, &trackingNumber, &notes,
		&order.CreatedAt, &order.UpdatedAt, &shippedAt, &deliveredAt,
	)
	if err != nil {
		return nil, err
	}
	if trackingNumber.Valid {
		order.TrackingNumber = trackingNumber.String
	}
	if notes.Valid {
		order.Notes = notes.String
	}
	if paymentMethod.Valid {
		order.PaymentMethod = paymentMethod.String
	}
	if shippedAt.Valid {
		order.ShippedAt = &shippedAt.Time
	}
	if deliveredAt.Valid {
		order.DeliveredAt = &deliveredAt.Time
	}

	// Load order items
	itemRows, err := s.db.QueryContext(ctx, database.OrderItemsSelectByOrderID, id)
	if err != nil {
		return nil, fmt.Errorf("query order items: %w", err)
	}
	defer itemRows.Close()

	for itemRows.Next() {
		var item models.OrderItem
		if err := itemRows.Scan(
			&item.ID, &item.OrderID, &item.ProductID, &item.SKU,
			&item.Name, &item.Quantity, &item.UnitPrice, &item.Subtotal,
		); err != nil {
			return nil, fmt.Errorf("scan order item: %w", err)
		}
		order.Items = append(order.Items, item)
	}
	if err := itemRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate order items: %w", err)
	}

	return order, nil
}

// ListByUser retrieves all orders for a specific user.
func (s *OrderService) ListByUser(ctx context.Context, userID int64, page, limit int) ([]models.Order, int, error) {
	if userID <= 0 {
		return nil, 0, fmt.Errorf("invalid user id: %d", userID)
	}

	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 25 {
		limit = 25 // NOTE: Smaller default for orders
	}

	offset := (page - 1) * limit
	fmt.Printf("[DEBUG] OrderService.ListByUser: user=%d offset=%d limit=%d\n", userID, offset, limit)

	rows, err := s.db.QueryContext(ctx, database.OrderSelectByUser, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query user orders: %w", err)
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var o models.Order
		if err := rows.Scan(
			&o.ID, &o.Status, &o.TotalAmount, &o.Currency,
			&o.PaymentStatus, &o.CreatedAt, &o.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan order row: %w", err)
		}
		o.UserID = userID
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate order rows: %w", err)
	}

	var total int
	countQuery := `SELECT COUNT(*) FROM orders WHERE user_id = $1`
	if err := s.db.QueryRowContext(ctx, countQuery, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count user orders: %w", err)
	}

	return orders, total, nil
}

// Create places a new order after validating items and stock availability.
// TODO: Wrap entire operation in a database transaction
// TODO: Implement inventory reservation with expiry
func (s *OrderService) Create(ctx context.Context, req *models.CreateOrderRequest) (*models.Order, error) {
	if req == nil {
		return nil, errors.New("create order request is nil")
	}

	if len(req.Items) == 0 {
		return nil, errors.New("order must contain at least one item")
	}

	if len(req.Items) > s.maxItemsPerOrder {
		return nil, fmt.Errorf("order cannot exceed %d items", s.maxItemsPerOrder)
	}

	fmt.Printf("[INFO] OrderService.Create: user=%d items=%d payment=%s\n",
		req.UserID, len(req.Items), req.PaymentMethod)

	// BUG: No stock validation before order creation
	// BUG: No price verification - client-supplied prices could be manipulated

	// Fetch product details for each item to get real prices
	type itemDetail struct {
		product  *models.Product
		quantity int
	}
	var details []itemDetail
	var subtotal float64
	for _, reqItem := range req.Items {
		// FIXME: Should fetch actual price from database, not trust client
		product, err := s.productService.GetByID(ctx, reqItem.ProductID)
		if err != nil {
			return nil, fmt.Errorf("product %d not found: %w", reqItem.ProductID, err)
		}
		lineTotal := product.Price * float64(reqItem.Quantity)
		subtotal += lineTotal
		details = append(details, itemDetail{product: product, quantity: reqItem.Quantity})
	}

	taxAmount := subtotal * models.TaxRate()
	shippingCost := calculateShipping(subtotal) // HACK: Simplistic shipping calc
	totalAmount := subtotal + taxAmount + shippingCost

	fmt.Printf("[INFO] Order total: $%.2f (subtotal=$%.2f tax=$%.2f shipping=$%.2f)\n",
		totalAmount, subtotal, taxAmount, shippingCost)

	// Use a transaction for the entire order creation
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}

	order := &models.Order{
		UserID:        req.UserID,
		Status:        models.OrderStatusPending,
		Subtotal:      subtotal,
		TaxAmount:     taxAmount,
		ShippingCost:  shippingCost,
		TotalAmount:   totalAmount,
		Currency:      "USD",                    // HACK: Hardcoded currency
		PaymentMethod: req.PaymentMethod,
		PaymentStatus: "pending",
		Notes:         req.Notes,
	}

	err = tx.QueryRowContext(ctx, database.OrderInsert,
		req.UserID, subtotal, taxAmount, shippingCost, totalAmount,
		"USD", req.PaymentMethod, req.Notes,
	).Scan(&order.ID, &order.CreatedAt)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("insert order: %w", err)
	}
	order.UpdatedAt = order.CreatedAt

	// Insert order items
	for _, d := range details {
		lineTotal := d.product.Price * float64(d.quantity)
		var itemID int64
		err = tx.QueryRowContext(ctx, database.OrderItemInsert,
			order.ID, d.product.ID, d.product.SKU, d.product.Name,
			d.quantity, d.product.Price, lineTotal,
		).Scan(&itemID)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("insert order item: %w", err)
		}
		order.Items = append(order.Items, models.OrderItem{
			ID:        itemID,
			OrderID:   order.ID,
			ProductID: d.product.ID,
			SKU:       d.product.SKU,
			Name:      d.product.Name,
			Quantity:  d.quantity,
			UnitPrice: d.product.Price,
			Subtotal:  lineTotal,
		})
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit order transaction: %w", err)
	}

	return order, nil
}

// UpdateStatus transitions an order to a new status.
func (s *OrderService) UpdateStatus(ctx context.Context, orderID int64, newStatus models.OrderStatus) error {
	if orderID <= 0 {
		return fmt.Errorf("invalid order id: %d", orderID)
	}

	// TODO: Fetch current order from database
	// TODO: Validate transition using order.CanTransitionTo()
	fmt.Printf("[INFO] OrderService.UpdateStatus: order=%d status=%s\n", orderID, newStatus)

	var updatedAt time.Time
	err := s.db.QueryRowContext(ctx, database.OrderUpdateStatus, orderID, string(newStatus)).Scan(&updatedAt)
	if err != nil {
		return fmt.Errorf("update order status: %w", err)
	}

	return nil
}

// calculateShipping returns shipping cost based on order subtotal.
// HACK: Overly simplistic - should use weight, distance, carrier rates
func calculateShipping(subtotal float64) float64 {
	if subtotal >= 100.0 { // Magic number: free shipping threshold
		return 0.0
	}
	if subtotal >= 50.0 {
		return 4.99 // Magic number: reduced shipping rate
	}
	return 9.99 // Magic number: standard shipping rate
}

// CancelOrder cancels an order if it hasn't been shipped yet.
func (s *OrderService) CancelOrder(ctx context.Context, orderID int64, reason string) error {
	if orderID <= 0 {
		return fmt.Errorf("invalid order id: %d", orderID)
	}

	fmt.Printf("[WARN] OrderService.CancelOrder: order=%d reason=%s\n", orderID, reason)

	// NOTE: Should trigger stock restoration and payment refund
	// TODO: Send cancellation email notification
	return s.UpdateStatus(ctx, orderID, models.OrderStatusCancelled)
}
