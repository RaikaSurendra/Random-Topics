-- ============================================
-- LARGE SCALE BENCHMARK: 1 Million Rows
-- Buffer pool restricted to 64MB
-- ============================================

\timing on
\pset pager off

\echo ''
\echo '╔════════════════════════════════════════════════════════════╗'
\echo '║     LARGE SCALE UUID BENCHMARK - 1 MILLION ROWS            ║'
\echo '║     Buffer Pool: 64MB (restricted)                         ║'
\echo '╚════════════════════════════════════════════════════════════╝'
\echo ''

-- Show current settings
SELECT name, setting, unit FROM pg_settings
WHERE name IN ('shared_buffers', 'work_mem', 'effective_cache_size');

-- ============================================
-- SETUP: Clear existing data
-- ============================================

\echo ''
\echo '=== SETUP: Clearing existing data ==='
TRUNCATE users_v4, users_v7, users_serial RESTART IDENTITY;
VACUUM FULL users_v4, users_v7, users_serial;

-- Clear filesystem cache simulation (reset stats)
SELECT pg_stat_reset();

-- ============================================
-- TEST 1: BIGSERIAL INSERT (Baseline)
-- ============================================

\echo ''
\echo '╔════════════════════════════════════════════════════════════╗'
\echo '║  TEST 1: BIGSERIAL INSERT - 1,000,000 rows                 ║'
\echo '╚════════════════════════════════════════════════════════════╝'

\timing on
INSERT INTO users_serial (email, name, data)
SELECT
    'user' || i || '@example.com',
    'User ' || i,
    jsonb_build_object('index', i, 'random', random())
FROM generate_series(1, 1000000) AS i;
\timing off

-- ============================================
-- TEST 2: UUIDv4 INSERT (Random - Expected Slow)
-- ============================================

\echo ''
\echo '╔════════════════════════════════════════════════════════════╗'
\echo '║  TEST 2: UUIDv4 INSERT - 1,000,000 rows                    ║'
\echo '║  (Random distribution - expect slower)                     ║'
\echo '╚════════════════════════════════════════════════════════════╝'

\timing on
INSERT INTO users_v4 (email, name, data)
SELECT
    'user' || i || '@example.com',
    'User ' || i,
    jsonb_build_object('index', i, 'random', random())
FROM generate_series(1, 1000000) AS i;
\timing off

-- ============================================
-- TEST 3: UUIDv7 INSERT (Time-ordered)
-- ============================================

\echo ''
\echo '╔════════════════════════════════════════════════════════════╗'
\echo '║  TEST 3: UUIDv7 INSERT - 1,000,000 rows                    ║'
\echo '║  (Time-ordered - expect faster page utilization)           ║'
\echo '╚════════════════════════════════════════════════════════════╝'

\timing on
INSERT INTO users_v7 (email, name, data)
SELECT
    'user' || i || '@example.com',
    'User ' || i,
    jsonb_build_object('index', i, 'random', random())
FROM generate_series(1, 1000000) AS i;
\timing off

-- Update statistics
ANALYZE users_v4, users_v7, users_serial;

-- ============================================
-- RESULTS: Table and Index Sizes
-- ============================================

\echo ''
\echo '╔════════════════════════════════════════════════════════════╗'
\echo '║  RESULTS: TABLE AND INDEX SIZES                            ║'
\echo '╚════════════════════════════════════════════════════════════╝'
\echo ''

SELECT
    'users_serial (BIGSERIAL)' AS table_name,
    pg_size_pretty(pg_table_size('users_serial')) AS heap_size,
    pg_size_pretty(pg_indexes_size('users_serial')) AS index_size,
    pg_size_pretty(pg_total_relation_size('users_serial')) AS total_size,
    (SELECT reltuples::BIGINT FROM pg_class WHERE relname = 'users_serial') AS row_count
UNION ALL
SELECT
    'users_v4 (UUIDv4)' AS table_name,
    pg_size_pretty(pg_table_size('users_v4')),
    pg_size_pretty(pg_indexes_size('users_v4')),
    pg_size_pretty(pg_total_relation_size('users_v4')),
    (SELECT reltuples::BIGINT FROM pg_class WHERE relname = 'users_v4')
UNION ALL
SELECT
    'users_v7 (UUIDv7)' AS table_name,
    pg_size_pretty(pg_table_size('users_v7')),
    pg_size_pretty(pg_indexes_size('users_v7')),
    pg_size_pretty(pg_total_relation_size('users_v7')),
    (SELECT reltuples::BIGINT FROM pg_class WHERE relname = 'users_v7');

-- ============================================
-- RESULTS: B-Tree Index Statistics
-- ============================================

\echo ''
\echo '╔════════════════════════════════════════════════════════════╗'
\echo '║  RESULTS: B-TREE INDEX DEPTH AND STRUCTURE                 ║'
\echo '╚════════════════════════════════════════════════════════════╝'
\echo ''

SELECT
    indexrelid::regclass AS index_name,
    pg_size_pretty(pg_relation_size(indexrelid)) AS size,
    (bt_metap(indexrelid::regclass::text)).level AS tree_depth,
    (bt_metap(indexrelid::regclass::text)).root AS root_page
FROM pg_index
WHERE indrelid IN ('users_v4'::regclass, 'users_v7'::regclass, 'users_serial'::regclass)
AND indisprimary
ORDER BY pg_relation_size(indexrelid) DESC;

-- ============================================
-- RESULTS: Primary Key Index Page Statistics
-- ============================================

\echo ''
\echo '╔════════════════════════════════════════════════════════════╗'
\echo '║  RESULTS: INDEX PAGE COUNT COMPARISON                      ║'
\echo '╚════════════════════════════════════════════════════════════╝'
\echo ''

SELECT
    indexrelname AS index_name,
    pg_relation_size(indexrelid) / 8192 AS total_pages,
    pg_size_pretty(pg_relation_size(indexrelid)) AS size
FROM pg_stat_user_indexes
WHERE relname IN ('users_v4', 'users_v7', 'users_serial')
AND indexrelname LIKE '%pkey'
ORDER BY pg_relation_size(indexrelid) DESC;

-- ============================================
-- TEST 4: Random Point Lookups (Cold Cache)
-- ============================================

\echo ''
\echo '╔════════════════════════════════════════════════════════════╗'
\echo '║  TEST 4: RANDOM POINT LOOKUPS (1000 queries each)          ║'
\echo '╚════════════════════════════════════════════════════════════╝'

-- Reset stats to measure fresh
SELECT pg_stat_reset();

-- Prepare random IDs for lookup
CREATE TEMP TABLE lookup_ids_v4 AS
SELECT id FROM users_v4 ORDER BY random() LIMIT 1000;

CREATE TEMP TABLE lookup_ids_v7 AS
SELECT id FROM users_v7 ORDER BY random() LIMIT 1000;

CREATE TEMP TABLE lookup_ids_serial AS
SELECT id FROM users_serial ORDER BY random() LIMIT 1000;

\echo ''
\echo '--- BIGSERIAL: 1000 random lookups ---'
\timing on
DO $$
DECLARE
    rec RECORD;
    result RECORD;
BEGIN
    FOR rec IN SELECT id FROM lookup_ids_serial LOOP
        SELECT * INTO result FROM users_serial WHERE id = rec.id;
    END LOOP;
END $$;
\timing off

\echo ''
\echo '--- UUIDv4: 1000 random lookups ---'
\timing on
DO $$
DECLARE
    rec RECORD;
    result RECORD;
BEGIN
    FOR rec IN SELECT id FROM lookup_ids_v4 LOOP
        SELECT * INTO result FROM users_v4 WHERE id = rec.id;
    END LOOP;
END $$;
\timing off

\echo ''
\echo '--- UUIDv7: 1000 random lookups ---'
\timing on
DO $$
DECLARE
    rec RECORD;
    result RECORD;
BEGIN
    FOR rec IN SELECT id FROM lookup_ids_v7 LOOP
        SELECT * INTO result FROM users_v7 WHERE id = rec.id;
    END LOOP;
END $$;
\timing off

-- ============================================
-- RESULTS: Buffer Cache Hit Ratios
-- ============================================

\echo ''
\echo '╔════════════════════════════════════════════════════════════╗'
\echo '║  RESULTS: BUFFER CACHE HIT RATIOS                          ║'
\echo '╚════════════════════════════════════════════════════════════╝'
\echo ''

SELECT
    relname AS table_name,
    heap_blks_read AS heap_disk_reads,
    heap_blks_hit AS heap_cache_hits,
    CASE WHEN heap_blks_read + heap_blks_hit > 0
         THEN round(100.0 * heap_blks_hit / (heap_blks_read + heap_blks_hit), 2)
         ELSE 0 END AS heap_hit_pct,
    idx_blks_read AS idx_disk_reads,
    idx_blks_hit AS idx_cache_hits,
    CASE WHEN idx_blks_read + idx_blks_hit > 0
         THEN round(100.0 * idx_blks_hit / (idx_blks_read + idx_blks_hit), 2)
         ELSE 0 END AS idx_hit_pct
FROM pg_statio_user_tables
WHERE relname IN ('users_v4', 'users_v7', 'users_serial')
ORDER BY relname;

-- ============================================
-- TEST 5: Sequential Batch Inserts (Append Pattern)
-- ============================================

\echo ''
\echo '╔════════════════════════════════════════════════════════════╗'
\echo '║  TEST 5: ADDITIONAL BATCH INSERT (100K more rows)          ║'
\echo '║  This shows append performance on existing large table     ║'
\echo '╚════════════════════════════════════════════════════════════╝'

\echo ''
\echo '--- BIGSERIAL: Append 100K rows ---'
\timing on
INSERT INTO users_serial (email, name, data)
SELECT
    'newuser' || i || '@example.com',
    'New User ' || i,
    jsonb_build_object('batch', 2, 'index', i)
FROM generate_series(1, 100000) AS i;
\timing off

\echo ''
\echo '--- UUIDv4: Append 100K rows ---'
\timing on
INSERT INTO users_v4 (email, name, data)
SELECT
    'newuser' || i || '@example.com',
    'New User ' || i,
    jsonb_build_object('batch', 2, 'index', i)
FROM generate_series(1, 100000) AS i;
\timing off

\echo ''
\echo '--- UUIDv7: Append 100K rows ---'
\timing on
INSERT INTO users_v7 (email, name, data)
SELECT
    'newuser' || i || '@example.com',
    'New User ' || i,
    jsonb_build_object('batch', 2, 'index', i)
FROM generate_series(1, 100000) AS i;
\timing off

-- ============================================
-- FINAL SUMMARY
-- ============================================

\echo ''
\echo '╔════════════════════════════════════════════════════════════╗'
\echo '║                    FINAL SUMMARY                           ║'
\echo '╚════════════════════════════════════════════════════════════╝'
\echo ''

SELECT
    'After 1.1M rows' AS test_phase,
    relname AS table_name,
    pg_size_pretty(pg_table_size(relname::regclass)) AS heap_size,
    pg_size_pretty(pg_indexes_size(relname::regclass)) AS index_size,
    (SELECT level FROM bt_metap(relname || '_pkey')) AS btree_depth
FROM (VALUES ('users_serial'), ('users_v4'), ('users_v7')) AS t(relname);

\echo ''
\echo 'Key Takeaways:'
\echo '─────────────────────────────────────────────────────────────'
\echo '1. BIGSERIAL has smallest index (8 bytes vs 16 bytes per key)'
\echo '2. UUIDv7 index should be more compact than UUIDv4'
\echo '3. UUIDv4 causes more page splits → larger index'
\echo '4. Buffer hit ratio shows cache efficiency differences'
\echo '5. Append performance shows insert pattern impact'
\echo ''
