-- ============================================
-- Database Initialization Script
-- Java Caching Learning Project
-- ============================================

-- Create products table
CREATE TABLE IF NOT EXISTS products (
    id BIGSERIAL PRIMARY KEY,
    sku VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price DECIMAL(10, 2) NOT NULL,
    category VARCHAR(100),
    stock_quantity INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_products_sku ON products(sku);
CREATE INDEX IF NOT EXISTS idx_products_category ON products(category);

-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);

-- Create orders table (for advanced examples)
CREATE TABLE IF NOT EXISTS orders (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id),
    product_sku VARCHAR(50) REFERENCES products(sku),
    quantity INTEGER NOT NULL,
    total_price DECIMAL(10, 2) NOT NULL,
    status VARCHAR(50) DEFAULT 'PENDING',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);

-- Create product_views table (for hot key detection)
CREATE TABLE IF NOT EXISTS product_views (
    id BIGSERIAL PRIMARY KEY,
    product_sku VARCHAR(50) REFERENCES products(sku),
    view_count INTEGER DEFAULT 0,
    last_viewed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_product_views_sku ON product_views(product_sku);

-- ============================================
-- Insert Sample Products (100 items)
-- ============================================

INSERT INTO products (sku, name, description, price, category, stock_quantity) VALUES
-- Electronics (SKU-001 to SKU-020)
('SKU-001', 'Wireless Mouse', 'Ergonomic wireless mouse with USB receiver', 29.99, 'Electronics', 150),
('SKU-002', 'Mechanical Keyboard', 'RGB backlit mechanical keyboard with Cherry MX switches', 89.99, 'Electronics', 75),
('SKU-003', 'USB-C Hub', '7-in-1 USB-C hub with HDMI, USB 3.0, SD card', 49.99, 'Electronics', 200),
('SKU-004', 'Webcam HD', '1080p HD webcam with built-in microphone', 59.99, 'Electronics', 80),
('SKU-005', 'Bluetooth Speaker', 'Portable Bluetooth 5.0 speaker with 20hr battery', 45.99, 'Electronics', 120),
('SKU-006', 'Wireless Earbuds', 'True wireless earbuds with noise cancellation', 79.99, 'Electronics', 90),
('SKU-007', 'External SSD 1TB', 'Portable SSD with USB 3.2 Gen 2', 109.99, 'Electronics', 60),
('SKU-008', 'Gaming Headset', '7.1 surround sound gaming headset', 69.99, 'Electronics', 85),
('SKU-009', 'Smart Watch', 'Fitness tracker with heart rate monitor', 149.99, 'Electronics', 40),
('SKU-010', 'Tablet Stand', 'Adjustable tablet and phone stand', 24.99, 'Electronics', 200),
('SKU-011', 'HDMI Cable 6ft', 'High-speed HDMI 2.1 cable 4K/120Hz', 14.99, 'Electronics', 500),
('SKU-012', 'USB-C Cable Pack', '3-pack USB-C to USB-C cables', 19.99, 'Electronics', 400),
('SKU-013', 'Wireless Charger', '15W fast wireless charging pad', 29.99, 'Electronics', 180),
('SKU-014', 'Power Bank 20000mAh', 'Portable charger with fast charging', 39.99, 'Electronics', 150),
('SKU-015', 'Ring Light', '10-inch LED ring light with tripod', 34.99, 'Electronics', 100),
('SKU-016', 'Microphone USB', 'Condenser microphone for streaming', 49.99, 'Electronics', 70),
('SKU-017', 'Capture Card', '4K HDMI capture card for streaming', 129.99, 'Electronics', 45),
('SKU-018', 'Router WiFi 6', 'Dual-band WiFi 6 router', 89.99, 'Electronics', 55),
('SKU-019', 'Network Switch', '8-port gigabit ethernet switch', 29.99, 'Electronics', 120),
('SKU-020', 'Ethernet Cable 50ft', 'Cat6 ethernet cable', 19.99, 'Electronics', 300),

-- Accessories (SKU-021 to SKU-040)
('SKU-021', 'Monitor Stand', 'Adjustable monitor stand with USB ports', 39.99, 'Accessories', 100),
('SKU-022', 'Desk Lamp LED', 'LED desk lamp with adjustable brightness', 34.99, 'Accessories', 120),
('SKU-023', 'Mouse Pad XL', 'Extra large gaming mouse pad 900x400mm', 19.99, 'Accessories', 250),
('SKU-024', 'Headphone Stand', 'Aluminum headphone stand with cable holder', 24.99, 'Accessories', 90),
('SKU-025', 'Cable Organizer', 'Desktop cable management kit', 14.99, 'Accessories', 300),
('SKU-026', 'Laptop Stand', 'Foldable aluminum laptop stand', 44.99, 'Accessories', 110),
('SKU-027', 'Wrist Rest', 'Memory foam keyboard wrist rest', 16.99, 'Accessories', 180),
('SKU-028', 'Monitor Arm', 'Single monitor arm mount', 49.99, 'Accessories', 75),
('SKU-029', 'Desk Organizer', 'Wooden desk organizer with drawers', 29.99, 'Accessories', 95),
('SKU-030', 'Document Holder', 'Adjustable document holder stand', 22.99, 'Accessories', 85),
('SKU-031', 'Keyboard Cover', 'Silicone keyboard protector', 9.99, 'Accessories', 400),
('SKU-032', 'Screen Cleaner', 'Screen cleaning kit with microfiber cloth', 12.99, 'Accessories', 350),
('SKU-033', 'Laptop Sleeve 15"', 'Neoprene laptop sleeve', 19.99, 'Accessories', 200),
('SKU-034', 'Backpack', 'Laptop backpack with USB charging port', 49.99, 'Accessories', 80),
('SKU-035', 'Desk Mat', 'Leather desk mat 80x40cm', 29.99, 'Accessories', 150),
('SKU-036', 'Pen Holder', 'Ceramic pen and pencil holder', 14.99, 'Accessories', 220),
('SKU-037', 'Sticky Notes', 'Colorful sticky notes pack', 8.99, 'Accessories', 500),
('SKU-038', 'Notebook', 'Premium hardcover notebook A5', 12.99, 'Accessories', 300),
('SKU-039', 'Mouse Bungee', 'Mouse cable management bungee', 19.99, 'Accessories', 130),
('SKU-040', 'Footrest', 'Ergonomic adjustable footrest', 34.99, 'Accessories', 65),

-- Home Office (SKU-041 to SKU-060)
('SKU-041', 'Office Chair', 'Ergonomic mesh office chair', 199.99, 'Home Office', 30),
('SKU-042', 'Standing Desk', 'Electric height adjustable desk', 399.99, 'Home Office', 20),
('SKU-043', 'Desk Shelf', 'Desktop shelf organizer', 39.99, 'Home Office', 90),
('SKU-044', 'Whiteboard', 'Magnetic whiteboard 24x18', 29.99, 'Home Office', 100),
('SKU-045', 'Corkboard', 'Cork bulletin board with frame', 24.99, 'Home Office', 110),
('SKU-046', 'Filing Cabinet', '3-drawer mobile filing cabinet', 79.99, 'Home Office', 45),
('SKU-047', 'Bookshelf', '5-tier bookshelf', 59.99, 'Home Office', 35),
('SKU-048', 'Plant Pot', 'Desktop plant pot with drainage', 14.99, 'Home Office', 200),
('SKU-049', 'Air Purifier', 'HEPA air purifier for small rooms', 89.99, 'Home Office', 50),
('SKU-050', 'Humidifier', 'USB desk humidifier', 24.99, 'Home Office', 120),
('SKU-051', 'Fan USB', 'Quiet USB desk fan', 19.99, 'Home Office', 180),
('SKU-052', 'Heater', 'Personal ceramic space heater', 39.99, 'Home Office', 70),
('SKU-053', 'Clock Digital', 'LED digital desk clock', 16.99, 'Home Office', 150),
('SKU-054', 'Timer', 'Pomodoro timer for productivity', 12.99, 'Home Office', 200),
('SKU-055', 'Calendar', 'Desktop flip calendar', 9.99, 'Home Office', 250),
('SKU-056', 'Picture Frame', 'Digital picture frame 10"', 49.99, 'Home Office', 60),
('SKU-057', 'Coaster Set', 'Cork coaster set of 6', 14.99, 'Home Office', 180),
('SKU-058', 'Mug Warmer', 'USB powered mug warmer', 19.99, 'Home Office', 140),
('SKU-059', 'Trash Can', 'Mini desktop trash can', 8.99, 'Home Office', 300),
('SKU-060', 'Paper Shredder', 'Personal paper shredder', 39.99, 'Home Office', 55),

-- Gaming (SKU-061 to SKU-080)
('SKU-061', 'Gaming Mouse', 'RGB gaming mouse 16000 DPI', 49.99, 'Gaming', 100),
('SKU-062', 'Gaming Keyboard', 'Mechanical gaming keyboard RGB', 79.99, 'Gaming', 80),
('SKU-063', 'Gaming Chair', 'Racing style gaming chair', 249.99, 'Gaming', 25),
('SKU-064', 'Gaming Desk', 'Carbon fiber gaming desk', 179.99, 'Gaming', 30),
('SKU-065', 'Controller', 'Wireless game controller', 59.99, 'Gaming', 90),
('SKU-066', 'Mouse Pad RGB', 'LED gaming mouse pad', 29.99, 'Gaming', 150),
('SKU-067', 'Headset Stand RGB', 'RGB headset stand with USB hub', 34.99, 'Gaming', 85),
('SKU-068', 'Stream Deck', '15-key LCD stream controller', 149.99, 'Gaming', 40),
('SKU-069', 'Webcam 4K', '4K streaming webcam', 129.99, 'Gaming', 35),
('SKU-070', 'Green Screen', 'Collapsible green screen', 79.99, 'Gaming', 45),
('SKU-071', 'Boom Arm', 'Microphone boom arm', 39.99, 'Gaming', 95),
('SKU-072', 'Pop Filter', 'Microphone pop filter', 14.99, 'Gaming', 200),
('SKU-073', 'Shock Mount', 'Microphone shock mount', 24.99, 'Gaming', 120),
('SKU-074', 'Audio Interface', 'USB audio interface', 99.99, 'Gaming', 50),
('SKU-075', 'Monitor 27"', '27" 144Hz gaming monitor', 299.99, 'Gaming', 20),
('SKU-076', 'Monitor 32"', '32" 4K gaming monitor', 449.99, 'Gaming', 15),
('SKU-077', 'Desk Pad XXL', 'Full desk gaming pad', 39.99, 'Gaming', 100),
('SKU-078', 'Cable Sleeve', 'Braided cable sleeve 10ft', 12.99, 'Gaming', 250),
('SKU-079', 'LED Strip', 'RGB LED strip for desk', 24.99, 'Gaming', 180),
('SKU-080', 'GPU Holder', 'Graphics card support bracket', 19.99, 'Gaming', 130),

-- Software & Services (SKU-081 to SKU-100)
('SKU-081', 'Antivirus 1yr', 'Premium antivirus subscription 1 year', 39.99, 'Software', 1000),
('SKU-082', 'VPN 1yr', 'VPN service subscription 1 year', 49.99, 'Software', 1000),
('SKU-083', 'Cloud Storage', '1TB cloud storage annual', 99.99, 'Software', 1000),
('SKU-084', 'Password Manager', 'Password manager annual subscription', 35.99, 'Software', 1000),
('SKU-085', 'Office Suite', 'Office productivity suite annual', 69.99, 'Software', 1000),
('SKU-086', 'Photo Editor', 'Professional photo editing software', 89.99, 'Software', 500),
('SKU-087', 'Video Editor', 'Video editing software license', 149.99, 'Software', 300),
('SKU-088', 'Music Software', 'Digital audio workstation', 199.99, 'Software', 200),
('SKU-089', 'Design Tool', 'Graphic design software annual', 119.99, 'Software', 400),
('SKU-090', 'Dev Tools', 'Developer tools subscription', 99.99, 'Software', 500),
('SKU-091', 'Learning Platform', 'Online learning platform annual', 149.99, 'Software', 800),
('SKU-092', 'Project Manager', 'Project management tool annual', 79.99, 'Software', 600),
('SKU-093', 'Time Tracker', 'Time tracking software annual', 49.99, 'Software', 700),
('SKU-094', 'Note Taking', 'Note-taking app premium annual', 34.99, 'Software', 900),
('SKU-095', 'Email Service', 'Premium email service annual', 59.99, 'Software', 750),
('SKU-096', 'Backup Service', 'Online backup service annual', 69.99, 'Software', 650),
('SKU-097', 'Domain Name', 'Domain registration 1 year', 14.99, 'Software', 2000),
('SKU-098', 'SSL Certificate', 'SSL certificate 1 year', 49.99, 'Software', 1500),
('SKU-099', 'Web Hosting', 'Web hosting annual plan', 99.99, 'Software', 1000),
('SKU-100', 'CDN Service', 'CDN service monthly', 29.99, 'Software', 800)

ON CONFLICT (sku) DO NOTHING;

-- ============================================
-- Insert Sample Users
-- ============================================

INSERT INTO users (username, email) VALUES
('john_doe', 'john@example.com'),
('jane_smith', 'jane@example.com'),
('bob_wilson', 'bob@example.com'),
('alice_jones', 'alice@example.com'),
('charlie_brown', 'charlie@example.com'),
('diana_prince', 'diana@example.com'),
('edward_stark', 'edward@example.com'),
('fiona_green', 'fiona@example.com'),
('george_miller', 'george@example.com'),
('hannah_white', 'hannah@example.com')
ON CONFLICT (username) DO NOTHING;

-- ============================================
-- Insert Sample Product Views (for hot key detection demo)
-- ============================================

INSERT INTO product_views (product_sku, view_count) VALUES
('SKU-001', 15000),  -- Hot product
('SKU-002', 12000),  -- Hot product
('SKU-061', 10000),  -- Hot product
('SKU-003', 5000),
('SKU-005', 4500),
('SKU-010', 3000),
('SKU-021', 2500),
('SKU-041', 2000),
('SKU-075', 1800),
('SKU-081', 1500)
ON CONFLICT DO NOTHING;

-- ============================================
-- Create helper function to update timestamps
-- ============================================

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

DROP TRIGGER IF EXISTS update_products_updated_at ON products;
CREATE TRIGGER update_products_updated_at
    BEFORE UPDATE ON products
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
