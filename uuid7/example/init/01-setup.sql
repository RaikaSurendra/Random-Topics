-- ============================================
-- UUID v4 vs v7 Benchmark Setup
-- ============================================

-- Enable timing
\timing on

-- Enable pgcrypto for random bytes
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ============================================
-- Create UUIDv7 function (RFC 9562 compliant)
-- ============================================

CREATE OR REPLACE FUNCTION uuidv7() RETURNS UUID AS $$
DECLARE
    unix_ts_ms BIGINT;
    uuid_bytes BYTEA;
BEGIN
    -- Get current timestamp in milliseconds
    unix_ts_ms := (EXTRACT(EPOCH FROM clock_timestamp()) * 1000)::BIGINT;

    -- Build UUIDv7:
    -- Bytes 0-5: 48-bit timestamp (big-endian)
    -- Byte 6: version (7) in high nibble + 4 bits of random
    -- Byte 7: random
    -- Byte 8: variant (10xx) in high 2 bits + 6 bits random
    -- Bytes 9-15: random

    uuid_bytes :=
        -- Timestamp (48 bits = 6 bytes)
        set_byte(set_byte(set_byte(set_byte(set_byte(set_byte(
            gen_random_bytes(16),
            0, ((unix_ts_ms >> 40) & 255)::INT),
            1, ((unix_ts_ms >> 32) & 255)::INT),
            2, ((unix_ts_ms >> 24) & 255)::INT),
            3, ((unix_ts_ms >> 16) & 255)::INT),
            4, ((unix_ts_ms >> 8) & 255)::INT),
            5, (unix_ts_ms & 255)::INT);

    -- Set version 7 (0111 in bits 4-7 of byte 6)
    uuid_bytes := set_byte(uuid_bytes, 6, (get_byte(uuid_bytes, 6) & 15) | 112);

    -- Set variant (10xx in bits 6-7 of byte 8)
    uuid_bytes := set_byte(uuid_bytes, 8, (get_byte(uuid_bytes, 8) & 63) | 128);

    RETURN encode(uuid_bytes, 'hex')::UUID;
END;
$$ LANGUAGE plpgsql VOLATILE;

-- ============================================
-- Create tables with different UUID strategies
-- ============================================

-- Table 1: UUIDv4 (Random - BAD for performance)
CREATE TABLE users_v4 (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    data JSONB DEFAULT '{}'
);

-- Table 2: UUIDv7 (Time-ordered - GOOD for performance)
CREATE TABLE users_v7 (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    data JSONB DEFAULT '{}'
);

-- Table 3: BIGSERIAL (Sequential - Best for performance, for comparison)
CREATE TABLE users_serial (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    data JSONB DEFAULT '{}'
);

-- ============================================
-- Create indexes for analysis
-- ============================================

CREATE INDEX idx_users_v4_created ON users_v4(created_at);
CREATE INDEX idx_users_v7_created ON users_v7(created_at);
CREATE INDEX idx_users_serial_created ON users_serial(created_at);

-- ============================================
-- Helper functions for analysis
-- ============================================

-- Function to get table and index sizes
CREATE OR REPLACE FUNCTION get_table_stats(table_name TEXT)
RETURNS TABLE (
    table_size TEXT,
    index_size TEXT,
    total_size TEXT,
    row_count BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        pg_size_pretty(pg_table_size(table_name::regclass)) AS table_size,
        pg_size_pretty(pg_indexes_size(table_name::regclass)) AS index_size,
        pg_size_pretty(pg_total_relation_size(table_name::regclass)) AS total_size,
        (SELECT reltuples::BIGINT FROM pg_class WHERE relname = table_name) AS row_count;
END;
$$ LANGUAGE plpgsql;

-- Function to check index bloat (simplified)
CREATE OR REPLACE FUNCTION get_index_stats(idx_name TEXT)
RETURNS TABLE (
    index_name TEXT,
    index_size TEXT,
    leaf_pages BIGINT,
    tree_level INTEGER
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        idx_name,
        pg_size_pretty(pg_relation_size(idx_name::regclass)),
        (pg_relation_size(idx_name::regclass) / 8192)::BIGINT,
        (SELECT level FROM bt_metap(idx_name));
END;
$$ LANGUAGE plpgsql;

-- Function to extract timestamp from UUIDv7
CREATE OR REPLACE FUNCTION uuid_v7_to_timestamp(uuid_val UUID)
RETURNS TIMESTAMPTZ AS $$
DECLARE
    uuid_hex TEXT;
    time_ms BIGINT;
BEGIN
    uuid_hex := replace(uuid_val::TEXT, '-', '');
    time_ms := ('x' || substring(uuid_hex, 1, 12))::BIT(48)::BIGINT;
    RETURN to_timestamp(time_ms / 1000.0);
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- Display setup completion
-- ============================================

SELECT 'Setup complete! Tables created:' AS status;
SELECT '- users_v4 (UUIDv4 - random)' AS table_info
UNION ALL
SELECT '- users_v7 (UUIDv7 - time-ordered)'
UNION ALL
SELECT '- users_serial (BIGSERIAL - sequential)';
