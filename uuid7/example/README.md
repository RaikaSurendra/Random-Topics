# PostgreSQL UUID v4 vs v7 Benchmark

A practical demonstration of the performance differences between UUIDv4 and UUIDv7 in PostgreSQL.

## Prerequisites

- Docker and Docker Compose installed
- Bash shell

## Quick Start

```bash
# 1. Start the database
./run-benchmark.sh start

# 2. Run the benchmark
./run-benchmark.sh benchmark

# 3. Clean up when done
./run-benchmark.sh clean
```

## What the Benchmark Tests

### 1. Insert Performance
Compares insert speed for 100,000 rows across:
- `users_v4` - UUIDv4 (random) primary key
- `users_v7` - UUIDv7 (time-ordered) primary key
- `users_serial` - BIGSERIAL (sequential) primary key

### 2. Table & Index Sizes
Shows storage overhead differences between UUID types.

### 3. B-Tree Index Statistics
Displays B-tree metadata including:
- Tree level (depth)
- Number of leaf pages
- Fast root information

### 4. Point Lookup Performance
Tests `SELECT * WHERE id = ?` with EXPLAIN ANALYZE.

### 5. Buffer Hit Ratio
Measures how effectively each index type uses the buffer pool.

### 6. Index Page Utilization
Shows how well-packed index pages are (higher = better).

## Manual Exploration

Connect to the database directly:

```bash
./run-benchmark.sh interactive
```

Then try these queries:

```sql
-- Check table sizes
SELECT * FROM get_table_stats('users_v4');
SELECT * FROM get_table_stats('users_v7');

-- Extract timestamp from UUIDv7
SELECT id, uuid_v7_to_timestamp(id) FROM users_v7 LIMIT 5;

-- Compare ID ordering
SELECT id FROM users_v4 ORDER BY id LIMIT 5;  -- Random order
SELECT id FROM users_v7 ORDER BY id LIMIT 5;  -- Time order!

-- See B-tree internals
SELECT * FROM bt_metap('users_v4_pkey');
SELECT * FROM bt_metap('users_v7_pkey');
```

## Expected Results

On a typical run with 100,000 rows:

| Metric | UUIDv4 | UUIDv7 | BIGSERIAL |
|--------|--------|--------|-----------|
| Insert Time | ~800ms | ~700ms | ~600ms |
| Index Size | ~5MB | ~5MB | ~2MB |
| B-tree Depth | 2 | 2 | 2 |

The differences become **dramatic** with:
- 1M+ rows
- Concurrent inserts
- Limited memory (small buffer pool)
- HDD storage (vs SSD)

## Large Scale Test

For a more dramatic demonstration, modify `benchmark.sql`:

```sql
\set row_count 1000000  -- 1 million rows
```

Then run:
```bash
./run-benchmark.sh benchmark
```

## Files

```
example/
├── docker-compose.yml   # PostgreSQL 17 container config
├── init/
│   └── 01-setup.sql     # Database initialization
├── benchmark.sql        # Benchmark queries
├── run-benchmark.sh     # Helper script
└── README.md            # This file
```

## Troubleshooting

### Container won't start
```bash
./run-benchmark.sh clean  # Remove old data
./run-benchmark.sh start  # Fresh start
```

### Permission denied on script
```bash
chmod +x run-benchmark.sh
```

### Port 5432 already in use
Edit `docker-compose.yml` and change the port mapping:
```yaml
ports:
  - "5433:5432"  # Use 5433 instead
```
