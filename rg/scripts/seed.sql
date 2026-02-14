-- Seed Data for WebShop Development Database
-- Run after migrate.sql to populate tables with test data.
--
-- Usage: psql -U devuser -d webshop_dev -f scripts/seed.sql
-- TODO: Add idempotent checks (ON CONFLICT DO NOTHING) for all inserts
-- FIXME: Passwords below are bcrypt hashes of "password123" — rotate before any demo

-- ============================================================
-- Users
-- ============================================================
INSERT INTO users (id, email, username, password_hash, first_name, last_name, role, phone, created_at, updated_at) VALUES
  (1, 'admin@webshop.dev', 'admin', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'Admin', 'User', 'admin', NULL, '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
  (2, 'alice@example.com', 'alice', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'Alice', 'Smith', 'customer', '555-0102', '2025-01-15 10:30:00', '2025-01-15 10:30:00'),
  (3, 'bob@example.com', 'bob', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'Bob', 'Jones', 'customer', '555-0103', '2025-02-01 14:00:00', '2025-02-01 14:00:00'),
  (4, 'charlie@example.com', 'charlie', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'Charlie', 'Brown', 'vendor', NULL, '2025-02-10 09:15:00', '2025-02-10 09:15:00'),
  (5, 'diana@example.com', 'diana', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'Diana', 'Prince', 'customer', '555-0105', '2025-03-01 16:45:00', '2025-03-01 16:45:00');
-- TODO: Add more users with varied roles for RBAC testing
-- FIXME: User #4 vendor role is not yet fully implemented in the auth handler

-- ============================================================
-- Categories
-- ============================================================
INSERT INTO categories (id, name, slug, description) VALUES
  (1, 'Electronics', 'electronics', 'Laptops, phones, and accessories'),
  (2, 'Books', 'books', 'Physical and digital books'),
  (3, 'Clothing', 'clothing', 'Apparel and fashion items'),
  (4, 'Home & Garden', 'home-garden', 'Furniture, tools, and decor');
-- NOTE: Category IDs are referenced by products below

-- ============================================================
-- Products
-- ============================================================
INSERT INTO products (id, sku, name, description, price, compare_price, cost_price, category_id, stock_quantity, weight, is_published, image_url, tags, created_at) VALUES
  (1, 'KB-MECH-001', 'Mechanical Keyboard', 'Cherry MX Brown switches, RGB backlit, full-size layout', 89.99, 109.99, 45.00, 1, 150, 800, true, NULL, 'keyboard,mechanical,rgb', '2025-01-10 08:00:00'),
  (2, 'MS-WRLS-001', 'Wireless Mouse', 'Ergonomic design, 2.4GHz wireless, 12-month battery life', 34.50, 0, 12.00, 1, 300, 120, true, NULL, 'mouse,wireless,ergonomic', '2025-01-10 08:00:00'),
  (3, 'HB-USBC-001', 'USB-C Hub', '7-in-1 hub: HDMI, USB-A x3, SD, microSD, PD 100W', 49.99, 59.99, 20.00, 1, 200, 150, true, NULL, 'hub,usb-c,adapter', '2025-01-12 12:00:00'),
  (4, 'BK-GO-001', 'Go Programming Language', 'The Go Programming Language by Donovan & Kernighan', 39.99, 0, 15.00, 2, 50, 500, true, NULL, 'book,programming,go', '2025-01-15 09:00:00'),
  (5, 'BK-SQL-001', 'Learning SQL', 'Master SQL fundamentals and advanced queries', 29.99, 0, 12.00, 2, 75, 450, true, NULL, 'book,sql,database', '2025-01-15 09:00:00'),
  (6, 'CL-TSH-001', 'Cotton T-Shirt', '100% organic cotton, available in S/M/L/XL', 19.99, 0, 5.00, 3, 500, 200, true, NULL, 'clothing,tshirt,cotton', '2025-02-01 10:00:00'),
  (7, 'CL-DNM-001', 'Denim Jacket', 'Classic fit denim jacket with button closure', 59.99, 79.99, 25.00, 3, 120, 700, true, NULL, 'clothing,jacket,denim', '2025-02-01 10:00:00'),
  (8, 'HG-DSK-001', 'Standing Desk', 'Electric height-adjustable, 60x30 inch bamboo top', 399.00, 499.00, 180.00, 4, 40, 25000, true, NULL, 'desk,standing,furniture', '2025-02-05 11:00:00'),
  (9, 'HG-LMP-001', 'Desk Lamp', 'LED desk lamp with adjustable brightness and color temperature', 24.99, 0, 8.00, 4, 250, 600, true, NULL, 'lamp,led,desk', '2025-02-05 11:00:00'),
  (10, 'XX-DEP-001', 'DEPRECATED Product', 'This product should be removed', 0.00, 0, 0, 1, 0, 0, false, NULL, '', '2024-06-01 00:00:00');
-- DEPRECATED: Product #10 is kept for order history integrity; do not display in catalog
-- TODO: Add product images table and seed image URLs
-- HACK: Prices are stored as DECIMAL but some old code treats them as floats

-- ============================================================
-- Orders
-- ============================================================
INSERT INTO orders (id, user_id, status, subtotal, tax_amount, shipping_cost, total_amount, currency, payment_method, payment_status, notes, created_at, updated_at) VALUES
  (1, 2, 'delivered', 124.49, 10.58, 0.00, 135.07, 'USD', 'credit_card', 'paid', NULL, '2025-02-20 14:30:00', '2025-02-22 10:00:00'),
  (2, 3, 'processing', 39.99, 3.40, 9.99, 53.38, 'USD', 'paypal', 'paid', NULL, '2025-03-01 09:00:00', '2025-03-01 09:00:00'),
  (3, 5, 'pending', 463.97, 39.44, 0.00, 503.41, 'USD', 'credit_card', 'pending', 'Please gift wrap', '2025-03-05 16:00:00', '2025-03-05 16:00:00'),
  (4, 2, 'cancelled', 19.99, 1.70, 9.99, 31.68, 'USD', 'credit_card', 'refunded', NULL, '2025-03-10 11:00:00', '2025-03-10 15:00:00'),
  (5, 3, 'shipped', 89.99, 7.65, 0.00, 97.64, 'USD', 'debit_card', 'paid', NULL, '2025-03-12 08:00:00', '2025-03-13 14:00:00');
-- FIXME: Order #3 total does not match line items — recalculate trigger is broken

-- ============================================================
-- Order Items (line items)
-- ============================================================
INSERT INTO order_items (id, order_id, product_id, sku, name, quantity, unit_price, subtotal) VALUES
  (1, 1, 1, 'KB-MECH-001', 'Mechanical Keyboard', 1, 89.99, 89.99),
  (2, 1, 2, 'MS-WRLS-001', 'Wireless Mouse', 1, 34.50, 34.50),
  (3, 2, 4, 'BK-GO-001', 'Go Programming Language', 1, 39.99, 39.99),
  (4, 3, 8, 'HG-DSK-001', 'Standing Desk', 1, 399.00, 399.00),
  (5, 3, 9, 'HG-LMP-001', 'Desk Lamp', 1, 24.99, 24.99),
  (6, 3, 6, 'CL-TSH-001', 'Cotton T-Shirt', 2, 19.99, 39.98),
  (7, 4, 6, 'CL-TSH-001', 'Cotton T-Shirt', 1, 19.99, 19.99),
  (8, 5, 1, 'KB-MECH-001', 'Mechanical Keyboard', 1, 89.99, 89.99);
-- NOTE: unit_price is captured at order time to preserve historical pricing

-- ============================================================
-- Reset sequences to avoid PK conflicts on next insert
-- ============================================================
SELECT setval('users_id_seq', (SELECT MAX(id) FROM users));
SELECT setval('categories_id_seq', (SELECT MAX(id) FROM categories));
SELECT setval('products_id_seq', (SELECT MAX(id) FROM products));
SELECT setval('orders_id_seq', (SELECT MAX(id) FROM orders));
SELECT setval('order_items_id_seq', (SELECT MAX(id) FROM order_items));
-- TODO: Wrap this file in a transaction for atomicity
