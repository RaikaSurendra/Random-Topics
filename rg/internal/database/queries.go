package database

// This file contains raw SQL query constants used throughout the application.
// NOTE: Using raw SQL for learning purposes - consider using a query builder in production.

// ============================================================
// User Queries
// ============================================================

// UserSelectByID fetches a single user by primary key.
const UserSelectByID = `
	SELECT id, email, username, password_hash, first_name, last_name,
	       role, phone, avatar_url, is_active, last_login_at,
	       created_at, updated_at
	FROM users
	WHERE id = $1`

// UserSelectByEmail fetches a user by their email address.
const UserSelectByEmail = `
	SELECT id, email, username, password_hash, first_name, last_name,
	       role, phone, avatar_url, is_active, last_login_at,
	       created_at, updated_at
	FROM users
	WHERE email = $1 AND is_active = true`

// UserSelectAll returns a paginated list of active users.
const UserSelectAll = `
	SELECT id, email, username, first_name, last_name, role, is_active,
	       created_at, updated_at
	FROM users
	WHERE is_active = true
	ORDER BY created_at DESC
	LIMIT $1 OFFSET $2`

// UserInsert creates a new user record.
const UserInsert = `
	INSERT INTO users (email, username, password_hash, first_name, last_name, role, phone, is_active, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, true, NOW(), NOW())
	RETURNING id, created_at`

// UserUpdate modifies an existing user.
const UserUpdate = `
	UPDATE users
	SET email = $2, username = $3, first_name = $4, last_name = $5,
	    phone = $6, role = $7, updated_at = NOW()
	WHERE id = $1
	RETURNING updated_at`

// UserSoftDelete deactivates a user without removing their data.
const UserSoftDelete = `
	UPDATE users
	SET is_active = false, updated_at = NOW()
	WHERE id = $1`

// DEPRECATED: UserSelectByUsername is replaced by UserSelectByEmail.
// Use email-based lookups for consistency. Will be removed in v2.0.
const UserSelectByUsername = `
	SELECT id, email, username, first_name, last_name, role
	FROM users
	WHERE username = $1 AND is_active = true`

// UserCountTotal returns the total number of active users.
const UserCountTotal = `SELECT COUNT(*) FROM users WHERE is_active = true`

// ============================================================
// Product Queries
// ============================================================

// ProductSelectByID fetches a single product.
const ProductSelectByID = `
	SELECT p.id, p.sku, p.name, p.description, p.price, p.compare_price,
	       p.cost_price, p.category_id, p.stock_quantity, p.weight,
	       p.is_published, p.image_url, p.tags, p.created_at, p.updated_at
	FROM products p
	WHERE p.id = $1`

// ProductSelectBySKU fetches a product by SKU code.
const ProductSelectBySKU = `
	SELECT id, sku, name, price, stock_quantity
	FROM products
	WHERE sku = $1 AND is_published = true`

// ProductSearch performs a filtered product search.
// TODO: Add full-text search using tsvector instead of ILIKE
const ProductSearch = `
	SELECT p.id, p.sku, p.name, p.description, p.price, p.compare_price,
	       p.category_id, p.stock_quantity, p.is_published, p.image_url,
	       p.tags, p.created_at, p.updated_at
	FROM products p
	WHERE p.is_published = true
	  AND (p.name ILIKE $1 OR p.description ILIKE $1)
	ORDER BY p.created_at DESC
	LIMIT $2 OFFSET $3`

// ProductInsert creates a new product.
const ProductInsert = `
	INSERT INTO products (sku, name, description, price, compare_price, cost_price,
	                      category_id, stock_quantity, weight, is_published, image_url, tags,
	                      created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW(), NOW())
	RETURNING id, created_at`

// ProductUpdateStock atomically updates stock quantity.
// BUG: Should use FOR UPDATE to prevent race conditions
const ProductUpdateStock = `
	UPDATE products
	SET stock_quantity = stock_quantity + $2, updated_at = NOW()
	WHERE id = $1 AND stock_quantity + $2 >= 0
	RETURNING stock_quantity`

// ============================================================
// Order Queries
// ============================================================

// OrderSelectByID fetches a single order with its items.
// FIXME: This doesn't actually JOIN items - need a separate query or lateral join
const OrderSelectByID = `
	SELECT o.id, o.user_id, o.status, o.subtotal, o.tax_amount,
	       o.shipping_cost, o.total_amount, o.currency, o.payment_method,
	       o.payment_status, o.tracking_number, o.notes,
	       o.created_at, o.updated_at, o.shipped_at, o.delivered_at
	FROM orders o
	WHERE o.id = $1`

// OrderItemsSelectByOrderID fetches all items for an order.
const OrderItemsSelectByOrderID = `
	SELECT oi.id, oi.order_id, oi.product_id, oi.sku, oi.name,
	       oi.quantity, oi.unit_price, oi.subtotal
	FROM order_items oi
	WHERE oi.order_id = $1
	ORDER BY oi.id`

// OrderSelectByUser fetches orders for a specific user.
const OrderSelectByUser = `
	SELECT o.id, o.status, o.total_amount, o.currency,
	       o.payment_status, o.created_at, o.updated_at
	FROM orders o
	WHERE o.user_id = $1
	ORDER BY o.created_at DESC
	LIMIT $2 OFFSET $3`

// OrderInsert creates a new order.
const OrderInsert = `
	INSERT INTO orders (user_id, status, subtotal, tax_amount, shipping_cost,
	                    total_amount, currency, payment_method, payment_status,
	                    notes, created_at, updated_at)
	VALUES ($1, 'pending', $2, $3, $4, $5, $6, $7, 'pending', $8, NOW(), NOW())
	RETURNING id, created_at`

// OrderItemInsert creates a line item for an order.
const OrderItemInsert = `
	INSERT INTO order_items (order_id, product_id, sku, name, quantity, unit_price, subtotal)
	VALUES ($1, $2, $3, $4, $5, $6, $7)
	RETURNING id`

// OrderUpdateStatus changes the status of an order.
const OrderUpdateStatus = `
	UPDATE orders
	SET status = $2, updated_at = NOW()
	WHERE id = $1
	RETURNING updated_at`

// DEPRECATED: OrderSelectAll fetches all orders without pagination.
// Use OrderSelectByUser with pagination instead. Will be removed in v2.0.
const OrderSelectAll = `
	SELECT id, user_id, status, total_amount, created_at
	FROM orders
	ORDER BY created_at DESC`

// OrderWithItemsJoin fetches an order with items in a single query.
// NOTE: Results need to be grouped/deduplicated in application code
const OrderWithItemsJoin = `
	SELECT o.id, o.user_id, o.status, o.total_amount, o.currency,
	       o.created_at, oi.id AS item_id, oi.product_id, oi.sku,
	       oi.name AS item_name, oi.quantity, oi.unit_price, oi.subtotal
	FROM orders o
	LEFT JOIN order_items oi ON o.id = oi.order_id
	WHERE o.id = $1
	ORDER BY oi.id`
