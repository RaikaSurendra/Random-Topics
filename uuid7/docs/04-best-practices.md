# Best Practices for UUIDs in Databases

This guide covers strategies for implementing performant UUID-based primary keys.

## Strategy 1: Use UUIDv7 (Recommended)

### PostgreSQL 17+ (Native Support)

```sql
-- PostgreSQL 17 introduced native UUIDv7
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    email VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

### PostgreSQL < 17 (Extension)

```sql
-- Install pg_uuidv7 extension
CREATE EXTENSION IF NOT EXISTS pg_uuidv7;

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    email VARCHAR(255) NOT NULL
);
```

### Application-Level Generation

Generate UUIDv7 in your application for better control:

```python
# Python with uuid7 package
import uuid7

new_id = uuid7.uuid7()  # Time-ordered UUID
```

```javascript
// Node.js with uuid package (v9+)
import { v7 as uuidv7 } from 'uuid';

const newId = uuidv7();
```

```go
// Go with google/uuid
import "github.com/google/uuid"

id, _ := uuid.NewV7()
```

## Strategy 2: Prefix/Composite Keys (ULID-Style)

If UUIDv7 isn't available, create time-prefixed IDs:

```sql
-- Custom time-prefixed UUID function
CREATE OR REPLACE FUNCTION time_ordered_uuid()
RETURNS UUID AS $$
DECLARE
    time_part BIGINT;
    uuid_text TEXT;
BEGIN
    -- Get milliseconds since epoch
    time_part := (EXTRACT(EPOCH FROM clock_timestamp()) * 1000)::BIGINT;

    -- Combine time prefix with random suffix
    uuid_text := lpad(to_hex(time_part), 12, '0') ||
                 replace(gen_random_uuid()::TEXT, '-', '');

    -- Format as UUID
    RETURN (
        substring(uuid_text, 1, 8) || '-' ||
        substring(uuid_text, 9, 4) || '-' ||
        substring(uuid_text, 13, 4) || '-' ||
        substring(uuid_text, 17, 4) || '-' ||
        substring(uuid_text, 21, 12)
    )::UUID;
END;
$$ LANGUAGE plpgsql;
```

## Strategy 3: Serial ID + UUID (Hybrid Approach)

Use auto-increment for indexing, UUID for external reference:

```sql
CREATE TABLE orders (
    -- Internal sequential ID (clustered index)
    id BIGSERIAL PRIMARY KEY,

    -- External reference (secondary index)
    public_id UUID NOT NULL DEFAULT gen_random_uuid(),

    -- Other columns
    customer_id BIGINT NOT NULL,
    total DECIMAL(10,2),

    -- Index for external lookups
    CONSTRAINT orders_public_id_unique UNIQUE (public_id)
);

-- Fast internal joins use `id`
-- API exposes `public_id` (no sequential exposure)
```

### When to Use This Pattern

| Use Case | Recommendation |
|----------|----------------|
| Internal microservices | Use `id` (BIGSERIAL) |
| External APIs | Use `public_id` (UUID) |
| Database joins | Use `id` (BIGSERIAL) |
| Customer-facing URLs | Use `public_id` (UUID) |

## Strategy 4: ULID (Universally Unique Lexicographically Sortable Identifier)

ULIDs are 128-bit identifiers that are time-ordered and URL-safe:

```
ULID Format: 01ARZ3NDEKTSV4RRFFQ69G5FAV
             |-------||------------|
              Time     Randomness
             (48 bit)  (80 bit)
```

```sql
-- PostgreSQL ULID extension
CREATE EXTENSION IF NOT EXISTS ulid;

CREATE TABLE events (
    id ULID PRIMARY KEY DEFAULT gen_ulid(),
    event_type VARCHAR(50),
    payload JSONB
);
```

### ULID vs UUIDv7

| Feature | ULID | UUIDv7 |
|---------|------|--------|
| Bits | 128 | 128 |
| Time precision | 48-bit ms | 48-bit ms |
| String format | 26 chars (Crockford Base32) | 36 chars (hex with dashes) |
| Sortable as string | Yes | Yes |
| Native PostgreSQL | Extension | Native (v17+) |
| Compatibility | Custom | UUID standard |

## Performance Tuning

### Index Fill Factor

For high-insert tables with UUIDs:

```sql
-- UUIDv7: Higher fill factor is fine (sequential inserts)
CREATE INDEX idx_users_id ON users(id) WITH (fillfactor = 90);

-- UUIDv4: Lower fill factor reduces splits (NOT recommended, use UUIDv7)
CREATE INDEX idx_legacy_id ON legacy_table(id) WITH (fillfactor = 70);
```

### BRIN Index for Time-Ordered UUIDs

UUIDv7 enables BRIN indexes (Block Range INdex):

```sql
-- Extremely small index for time-ordered UUIDs
CREATE INDEX idx_events_id_brin ON events USING BRIN (id);

-- BRIN index size comparison:
-- B-tree on 100M rows: ~2GB
-- BRIN on 100M rows:   ~100KB
```

### Partitioning by UUID Range

Time-ordered UUIDs enable efficient range partitioning:

```sql
-- Partition by UUID range (UUIDv7)
CREATE TABLE events (
    id UUID NOT NULL,
    event_data JSONB
) PARTITION BY RANGE (id);

-- Monthly partitions based on UUID timestamp prefix
CREATE TABLE events_2024_01 PARTITION OF events
    FOR VALUES FROM ('018d3c00-0000-0000-0000-000000000000')
                 TO ('018da580-0000-0000-0000-000000000000');
```

## Migration: UUIDv4 to UUIDv7

### Strategy A: New Column (Zero Downtime)

```sql
-- Step 1: Add new column
ALTER TABLE users ADD COLUMN id_v7 UUID;

-- Step 2: Backfill with time-ordered UUIDs
UPDATE users SET id_v7 = uuid_generate_v7()
WHERE id_v7 IS NULL;

-- Step 3: Add new index (concurrently)
CREATE INDEX CONCURRENTLY idx_users_id_v7 ON users(id_v7);

-- Step 4: Application switches to id_v7
-- Step 5: Eventually drop old column
```

### Strategy B: Keep UUIDv4, Add Clustering Index

```sql
-- Add created_at if not present
ALTER TABLE users ADD COLUMN created_at TIMESTAMPTZ DEFAULT NOW();

-- Create index for clustering
CREATE INDEX idx_users_created ON users(created_at);

-- Periodically cluster table (requires lock)
CLUSTER users USING idx_users_created;
```

## Anti-Patterns to Avoid

### 1. Don't Use UUIDv4 for High-Volume Tables

```sql
-- ❌ BAD: Random UUIDs as primary key
CREATE TABLE high_volume_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),  -- Causes fragmentation
    log_data TEXT
);

-- ✅ GOOD: Time-ordered UUIDs
CREATE TABLE high_volume_logs (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    log_data TEXT
);
```

### 2. Don't Index Random UUIDs with B-tree for Range Queries

```sql
-- ❌ BAD: Trying to do range queries on UUIDv4
SELECT * FROM events
WHERE id > 'a0000000-0000-0000-0000-000000000000';  -- Meaningless

-- ✅ GOOD: Range queries on UUIDv7 (time-based)
SELECT * FROM events
WHERE id > '018f0000-0000-7000-0000-000000000000';  -- Last hour's data
```

### 3. Don't Mix UUID Versions in Same Column

```sql
-- ❌ BAD: Mixing versions loses ordering benefits
INSERT INTO users (id) VALUES
    (gen_random_uuid()),  -- v4
    (uuid_generate_v7()); -- v7

-- ✅ GOOD: Consistent version throughout
INSERT INTO users (id) VALUES
    (uuid_generate_v7()),
    (uuid_generate_v7());
```

## Checklist for UUID Implementation

```
□ Choose UUIDv7 for new projects
□ Consider ULID if string sortability matters
□ Use native database functions when available
□ Avoid UUIDv4 for primary keys on large tables
□ Use BRIN indexes for time-ordered UUIDs when appropriate
□ Set appropriate fill factors
□ Monitor index bloat regularly
□ Consider hybrid approach for external APIs
□ Plan migration strategy for existing UUIDv4 columns
```

## Database-Specific Recommendations

| Database | UUID Recommendation |
|----------|---------------------|
| PostgreSQL 17+ | Native `uuidv7()` |
| PostgreSQL < 17 | `pg_uuidv7` extension |
| MySQL 8.0+ | `UUID_TO_BIN(UUID(), 1)` (swap time bits) |
| SQL Server | `NEWSEQUENTIALID()` or app-generated UUIDv7 |
| SQLite | App-generated UUIDv7 |
| MongoDB | Already uses time-ordered ObjectId |
