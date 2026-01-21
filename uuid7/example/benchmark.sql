-- ============================================
-- UUID v4 vs v7 Performance Benchmark
-- Run this after docker-compose is up
-- ============================================

\timing on
\pset pager off

-- Configuration
\set row_count 100000
\set batch_size 1000

-- ============================================
-- PART 1: Insert Performance Test
-- ============================================

\echo ''
\echo '============================================'
\echo 'PART 1: INSERT PERFORMANCE TEST'
\echo '============================================'
\echo ''

-- Clear any existing data
TRUNCATE users_v4, users_v7, users_serial RESTART IDENTITY;

-- Analyze empty tables
ANALYZE users_v4, users_v7, users_serial;

\echo 'Inserting :row_count rows into each table...'
\echo ''

-- Test 1: UUIDv4 Insert Performance
\echo '--- UUIDv4 INSERT ---'
\timing on
INSERT INTO users_v4 (email, name, data)
SELECT
    'user' || i || '@example.com',
    'User ' || i,
    jsonb_build_object('index', i, 'random', random())
FROM generate_series(1, :row_count) AS i;
\timing off

-- Test 2: UUIDv7 Insert Performance
\echo ''
\echo '--- UUIDv7 INSERT ---'
\timing on
INSERT INTO users_v7 (email, name, data)
SELECT
    'user' || i || '@example.com',
    'User ' || i,
    jsonb_build_object('index', i, 'random', random())
FROM generate_series(1, :row_count) AS i;
\timing off

-- Test 3: BIGSERIAL Insert Performance
\echo ''
\echo '--- BIGSERIAL INSERT ---'
\timing on
INSERT INTO users_serial (email, name, data)
SELECT
    'user' || i || '@example.com',
    'User ' || i,
    jsonb_build_object('index', i, 'random', random())
FROM generate_series(1, :row_count) AS i;
\timing off

-- Update statistics
ANALYZE users_v4, users_v7, users_serial;

-- ============================================
-- PART 2: Table & Index Size Analysis
-- ============================================

\echo ''
\echo '============================================'
\echo 'PART 2: TABLE & INDEX SIZE ANALYSIS'
\echo '============================================'
\echo ''

SELECT
    'users_v4 (UUIDv4)' AS table_name,
    pg_size_pretty(pg_table_size('users_v4')) AS heap_size,
    pg_size_pretty(pg_indexes_size('users_v4')) AS index_size,
    pg_size_pretty(pg_total_relation_size('users_v4')) AS total_size
UNION ALL
SELECT
    'users_v7 (UUIDv7)',
    pg_size_pretty(pg_table_size('users_v7')),
    pg_size_pretty(pg_indexes_size('users_v7')),
    pg_size_pretty(pg_total_relation_size('users_v7'))
UNION ALL
SELECT
    'users_serial (BIGSERIAL)',
    pg_size_pretty(pg_table_size('users_serial')),
    pg_size_pretty(pg_indexes_size('users_serial')),
    pg_size_pretty(pg_total_relation_size('users_serial'));

-- ============================================
-- PART 3: B-Tree Index Statistics
-- ============================================

\echo ''
\echo '============================================'
\echo 'PART 3: B-TREE INDEX STATISTICS'
\echo '============================================'
\echo ''

-- Enable pageinspect extension
CREATE EXTENSION IF NOT EXISTS pageinspect;

-- Get B-tree metadata for primary key indexes
SELECT
    indexrelid::regclass AS index_name,
    (bt_metap(indexrelid::regclass::text)).*
FROM pg_index
WHERE indrelid IN ('users_v4'::regclass, 'users_v7'::regclass, 'users_serial'::regclass)
AND indisprimary;

-- ============================================
-- PART 4: Point Lookup Performance
-- ============================================

\echo ''
\echo '============================================'
\echo 'PART 4: POINT LOOKUP PERFORMANCE'
\echo '============================================'
\echo ''

-- Get sample IDs for testing
\echo 'Getting sample IDs...'

CREATE TEMP TABLE sample_ids AS
SELECT
    (SELECT id FROM users_v4 ORDER BY random() LIMIT 1) AS v4_id,
    (SELECT id FROM users_v7 ORDER BY random() LIMIT 1) AS v7_id,
    (SELECT id FROM users_serial ORDER BY random() LIMIT 1) AS serial_id;

-- Clear buffer cache (requires superuser - skip if not available)
-- DISCARD ALL;

\echo ''
\echo '--- UUIDv4 POINT LOOKUP ---'
\timing on
EXPLAIN (ANALYZE, BUFFERS)
SELECT * FROM users_v4 WHERE id = (SELECT v4_id FROM sample_ids);
\timing off

\echo ''
\echo '--- UUIDv7 POINT LOOKUP ---'
\timing on
EXPLAIN (ANALYZE, BUFFERS)
SELECT * FROM users_v7 WHERE id = (SELECT v7_id FROM sample_ids);
\timing off

\echo ''
\echo '--- BIGSERIAL POINT LOOKUP ---'
\timing on
EXPLAIN (ANALYZE, BUFFERS)
SELECT * FROM users_serial WHERE id = (SELECT serial_id FROM sample_ids);
\timing off

-- ============================================
-- PART 5: Range Query Performance (Time-based)
-- ============================================

\echo ''
\echo '============================================'
\echo 'PART 5: RANGE QUERY ANALYSIS'
\echo '============================================'
\echo ''

-- For UUIDv7, we can do range queries on the ID itself!
\echo '--- UUIDv7 Range Query (by ID - includes time) ---'
\echo 'Note: UUIDv7 IDs are time-ordered, so we can query ranges!'
\echo ''

-- Get min and max UUIDv7 to show time extraction
SELECT
    'First ID' AS position,
    id,
    uuid_v7_to_timestamp(id) AS extracted_timestamp
FROM users_v7
ORDER BY id
LIMIT 1;

SELECT
    'Last ID' AS position,
    id,
    uuid_v7_to_timestamp(id) AS extracted_timestamp
FROM users_v7
ORDER BY id DESC
LIMIT 1;

-- ============================================
-- PART 6: Sequential vs Random Access Pattern
-- ============================================

\echo ''
\echo '============================================'
\echo 'PART 6: BUFFER HIT RATIO TEST'
\echo '============================================'
\echo ''

-- Reset statistics
SELECT pg_stat_reset();

-- Perform multiple lookups to test buffer caching
\echo 'Performing 100 random lookups on each table...'

-- UUIDv4 lookups
DO $$
DECLARE
    sample_id UUID;
BEGIN
    FOR i IN 1..100 LOOP
        SELECT id INTO sample_id FROM users_v4 OFFSET floor(random() * 100000) LIMIT 1;
        PERFORM * FROM users_v4 WHERE id = sample_id;
    END LOOP;
END $$;

-- UUIDv7 lookups
DO $$
DECLARE
    sample_id UUID;
BEGIN
    FOR i IN 1..100 LOOP
        SELECT id INTO sample_id FROM users_v7 OFFSET floor(random() * 100000) LIMIT 1;
        PERFORM * FROM users_v7 WHERE id = sample_id;
    END LOOP;
END $$;

-- BIGSERIAL lookups
DO $$
DECLARE
    sample_id BIGINT;
BEGIN
    FOR i IN 1..100 LOOP
        SELECT id INTO sample_id FROM users_serial OFFSET floor(random() * 100000) LIMIT 1;
        PERFORM * FROM users_serial WHERE id = sample_id;
    END LOOP;
END $$;

-- Check buffer hit ratios
\echo ''
\echo 'Buffer Hit Ratios (higher is better):'
SELECT
    relname AS table_name,
    heap_blks_read,
    heap_blks_hit,
    CASE
        WHEN heap_blks_read + heap_blks_hit = 0 THEN 0
        ELSE round(100.0 * heap_blks_hit / (heap_blks_read + heap_blks_hit), 2)
    END AS heap_hit_ratio,
    idx_blks_read,
    idx_blks_hit,
    CASE
        WHEN idx_blks_read + idx_blks_hit = 0 THEN 0
        ELSE round(100.0 * idx_blks_hit / (idx_blks_read + idx_blks_hit), 2)
    END AS idx_hit_ratio
FROM pg_statio_user_tables
WHERE relname IN ('users_v4', 'users_v7', 'users_serial')
ORDER BY relname;

-- ============================================
-- PART 7: Page Split Analysis
-- ============================================

\echo ''
\echo '============================================'
\echo 'PART 7: INDEX PAGE UTILIZATION'
\echo '============================================'
\echo ''

-- This shows how well-packed the index pages are
SELECT
    indexrelname,
    pg_relation_size(indexrelid) AS index_bytes,
    pg_relation_size(indexrelid) / 8192 AS total_pages,
    round(
        100.0 * (SELECT relpages FROM pg_class WHERE oid = indexrelid) /
        NULLIF((SELECT reltuples FROM pg_class WHERE relname = t.relname)::BIGINT / 400, 0),
        2
    ) AS estimated_fill_pct
FROM pg_stat_user_indexes t
WHERE relname IN ('users_v4', 'users_v7', 'users_serial')
ORDER BY indexrelname;

-- ============================================
-- SUMMARY
-- ============================================

\echo ''
\echo '============================================'
\echo 'SUMMARY'
\echo '============================================'
\echo ''
\echo 'Key Observations:'
\echo '1. Insert times: UUIDv7 ≈ BIGSERIAL < UUIDv4'
\echo '2. Index sizes: BIGSERIAL < UUIDv7 ≤ UUIDv4'
\echo '3. B-tree depth: Should be similar for same row count'
\echo '4. Buffer hits: UUIDv7 typically better than UUIDv4 for sequential inserts'
\echo ''
\echo 'The difference becomes MORE pronounced with:'
\echo '- Larger datasets (millions of rows)'
\echo '- Smaller buffer pool (memory pressure)'
\echo '- Concurrent inserts (lock contention on page splits)'
\echo ''
