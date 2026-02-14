-- ============================================
-- Additional Sample Data Script
-- Run this to add more test data
-- ============================================

-- Generate additional products dynamically
DO $$
DECLARE
    i INTEGER;
    categories TEXT[] := ARRAY['Electronics', 'Accessories', 'Home Office', 'Gaming', 'Software'];
    category TEXT;
BEGIN
    FOR i IN 101..200 LOOP
        category := categories[1 + (i % 5)];
        INSERT INTO products (sku, name, description, price, category, stock_quantity)
        VALUES (
            'SKU-' || LPAD(i::TEXT, 3, '0'),
            'Product ' || i,
            'Description for product ' || i || ' in category ' || category,
            ROUND((10 + RANDOM() * 190)::NUMERIC, 2),
            category,
            FLOOR(10 + RANDOM() * 500)::INTEGER
        )
        ON CONFLICT (sku) DO NOTHING;
    END LOOP;
END $$;

-- Generate sample orders
DO $$
DECLARE
    i INTEGER;
    user_id INTEGER;
    product_sku TEXT;
    qty INTEGER;
    statuses TEXT[] := ARRAY['PENDING', 'PROCESSING', 'SHIPPED', 'DELIVERED', 'CANCELLED'];
BEGIN
    FOR i IN 1..100 LOOP
        user_id := 1 + (i % 10);
        product_sku := 'SKU-' || LPAD((1 + (i % 100))::TEXT, 3, '0');
        qty := 1 + (i % 5);

        INSERT INTO orders (user_id, product_sku, quantity, total_price, status)
        SELECT
            user_id,
            product_sku,
            qty,
            p.price * qty,
            statuses[1 + (i % 5)]
        FROM products p
        WHERE p.sku = product_sku
        ON CONFLICT DO NOTHING;
    END LOOP;
END $$;

-- Update product view counts with realistic distribution (Zipf-like)
UPDATE product_views SET view_count = view_count + FLOOR(RANDOM() * 1000)::INTEGER;

-- Add more product views for new products
INSERT INTO product_views (product_sku, view_count)
SELECT sku, FLOOR(100 + RANDOM() * 5000)::INTEGER
FROM products
WHERE sku NOT IN (SELECT product_sku FROM product_views)
ON CONFLICT DO NOTHING;

-- Output summary
SELECT 'Products: ' || COUNT(*) FROM products;
SELECT 'Users: ' || COUNT(*) FROM users;
SELECT 'Orders: ' || COUNT(*) FROM orders;
SELECT 'Product Views: ' || COUNT(*) FROM product_views;
