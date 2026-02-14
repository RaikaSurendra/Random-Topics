-- Database Migration Script for WebShop
-- Creates all tables, indexes, and constraints.
--
-- Usage: psql -U devuser -d webshop_dev -f scripts/migrate.sql
-- NOTE: Run this before seed.sql
-- TODO: Switch to a proper migration tool (golang-migrate, goose, or atlas)

BEGIN;

-- ============================================================
-- Users table
-- ============================================================
CREATE TABLE IF NOT EXISTS users (
    id            SERIAL PRIMARY KEY,
    email         VARCHAR(255) NOT NULL UNIQUE,
    username      VARCHAR(100) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    first_name    VARCHAR(100) NOT NULL DEFAULT '',
    last_name     VARCHAR(100) NOT NULL DEFAULT '',
    role          VARCHAR(50)  NOT NULL DEFAULT 'customer',
    phone         VARCHAR(20),
    avatar_url    TEXT,
    is_active     BOOLEAN      NOT NULL DEFAULT true,
    last_login_at TIMESTAMP,
    created_at    TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMP    NOT NULL DEFAULT NOW()
);

-- DEPRECATED: The 'phone' column was removed in v1.3; use the user_profiles table instead
-- ALTER TABLE users ADD COLUMN phone VARCHAR(20);

CREATE INDEX idx_users_email ON users (email);
CREATE INDEX idx_users_role ON users (role);
-- TODO: Add partial index for active users only: WHERE is_active = true

-- ============================================================
-- Categories table
-- ============================================================
CREATE TABLE IF NOT EXISTS categories (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL,
    slug        VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    parent_id   INTEGER REFERENCES categories(id) ON DELETE SET NULL
);

CREATE INDEX idx_categories_slug ON categories (slug);
-- NOTE: parent_id enables nested categories (tree structure)

-- ============================================================
-- Products table
-- ============================================================
CREATE TABLE IF NOT EXISTS products (
    id              SERIAL PRIMARY KEY,
    sku             VARCHAR(50) NOT NULL UNIQUE,
    name            VARCHAR(255) NOT NULL,
    description     TEXT,
    price           DECIMAL(10, 2) NOT NULL CHECK (price >= 0),
    compare_price   DECIMAL(10, 2) DEFAULT 0,
    cost_price      DECIMAL(10, 2) DEFAULT 0,
    category_id     INTEGER REFERENCES categories(id) ON DELETE SET NULL,
    stock_quantity  INTEGER NOT NULL DEFAULT 0 CHECK (stock_quantity >= 0),
    weight          DECIMAL(10, 2) DEFAULT 0,
    is_published    BOOLEAN NOT NULL DEFAULT true,
    image_url       TEXT,
    tags            TEXT,
    created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP NOT NULL DEFAULT NOW()
);

-- DEPRECATED: 'sku' column was replaced by 'slug' in v1.2
-- ALTER TABLE products ADD COLUMN sku VARCHAR(50) UNIQUE;

CREATE INDEX idx_products_sku ON products (sku);
CREATE INDEX idx_products_category ON products (category_id);
CREATE INDEX idx_products_published ON products (is_published) WHERE is_published = true;
-- FIXME: Need a GIN index on description for full-text search
-- TODO: Add a product_images table for multiple images per product

-- ============================================================
-- Orders table
-- ============================================================
CREATE TABLE IF NOT EXISTS orders (
    id               SERIAL PRIMARY KEY,
    user_id          INTEGER NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status           VARCHAR(50) NOT NULL DEFAULT 'pending',
    subtotal         DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
    tax_amount       DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
    shipping_cost    DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
    total_amount     DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
    currency         VARCHAR(10) NOT NULL DEFAULT 'USD',
    payment_method   VARCHAR(50),
    payment_status   VARCHAR(50) NOT NULL DEFAULT 'pending',
    tracking_number  VARCHAR(100),
    notes            TEXT,
    created_at       TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMP NOT NULL DEFAULT NOW(),
    shipped_at       TIMESTAMP,
    delivered_at     TIMESTAMP,
    CONSTRAINT chk_order_status CHECK (status IN ('pending', 'confirmed', 'processing', 'shipped', 'delivered', 'cancelled', 'refunded'))
);

CREATE INDEX idx_orders_user ON orders (user_id);
CREATE INDEX idx_orders_status ON orders (status);
CREATE INDEX idx_orders_created ON orders (created_at DESC);

-- ============================================================
-- Order Items table
-- ============================================================
CREATE TABLE IF NOT EXISTS order_items (
    id         SERIAL PRIMARY KEY,
    order_id   INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id INTEGER NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    sku        VARCHAR(50),
    name       VARCHAR(255),
    quantity   INTEGER NOT NULL CHECK (quantity > 0),
    unit_price DECIMAL(10, 2) NOT NULL CHECK (unit_price >= 0),
    subtotal   DECIMAL(10, 2) NOT NULL DEFAULT 0.00
);

CREATE INDEX idx_order_items_order ON order_items (order_id);
-- NOTE: unit_price is stored per-item to preserve price at time of purchase

-- ============================================================
-- Sessions table (for auth token management)
-- ============================================================
-- DEPRECATED: Sessions are now managed via JWT; this table is kept for legacy API support
CREATE TABLE IF NOT EXISTS sessions (
    id         VARCHAR(128) PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token      TEXT NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sessions_user ON sessions (user_id);
CREATE INDEX idx_sessions_expires ON sessions (expires_at);
-- HACK: Old session cleanup runs via a cron job; should use pg_cron or app-level TTL

-- ============================================================
-- Audit log (append-only)
-- ============================================================
CREATE TABLE IF NOT EXISTS audit_log (
    id         BIGSERIAL PRIMARY KEY,
    user_id    INTEGER REFERENCES users(id),
    action     VARCHAR(100) NOT NULL,
    entity     VARCHAR(100) NOT NULL,
    entity_id  INTEGER,
    details    JSONB,
    ip_address INET,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_user ON audit_log (user_id);
CREATE INDEX idx_audit_entity ON audit_log (entity, entity_id);
CREATE INDEX idx_audit_created ON audit_log (created_at DESC);
-- TODO: Add partitioning by month for large-scale deployments
-- TODO: Add a retention policy to purge logs older than 90 days

COMMIT;
