# Chapter 02: Bloom Filters - High Level Design

## Overview

This chapter implements Bloom filter protection to prevent cache penetration attacks, where malicious or accidental requests for non-existent keys overwhelm the database.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Client Request                          │
│                    GET /api/products/SKU-XYZ                    │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Bloom Filter Check                           │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Option A: Guava BloomFilter (In-Memory)                │   │
│  │  - Fast: ~100ns per lookup                              │   │
│  │  - Single instance only                                  │   │
│  │                                                          │   │
│  │  Option B: RedisBloom (Distributed)                      │   │
│  │  - Fast: ~1ms per lookup (network)                      │   │
│  │  - Shared across all instances                          │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                    │                           │
          NOT IN SET (100%)              MAYBE IN SET
          Definitely doesn't exist       Could exist (or false positive)
                    │                           │
                    ▼                           ▼
        ┌─────────────────┐         ┌─────────────────────────────┐
        │  Return 404     │         │     Standard Cache Flow     │
        │  Immediately    │         │  Cache → DB → Cache         │
        │  (0 DB queries) │         └─────────────────────────────┘
        └─────────────────┘
```

## Components

### 1. Bloom Filter Implementations

| Component | Type | Use Case |
|-----------|------|----------|
| InMemoryBloomFilter | Guava | Single instance, ultra-fast |
| RedisBloomFilter | RedisBloom | Multi-instance, distributed |

### 2. Protection Service

| Component | Responsibility |
|-----------|----------------|
| BloomProtectedService | Combines Bloom check + cache lookup |

### 3. Filter Management

| Operation | Description |
|-----------|-------------|
| Initialize | Load all valid keys on startup |
| Add | Add new key when product created |
| Rebuild | Periodic full rebuild from DB |

## Data Flow

### Request for Valid Key

```
1. Client → GET /products/SKU-001
2. Bloom Filter Check → MAYBE IN SET
3. Redis Cache Check → HIT
4. Return cached product
```

### Request for Invalid Key (Blocked)

```
1. Client → GET /products/FAKE-SKU
2. Bloom Filter Check → NOT IN SET
3. Return 404 immediately
4. No cache check, no DB query
```

### Request for Invalid Key (False Positive)

```
1. Client → GET /products/FAKE-BUT-PASSED
2. Bloom Filter Check → MAYBE IN SET (false positive)
3. Redis Cache Check → MISS
4. Database Query → Not found
5. Return 404
6. Note: ~1% of invalid keys will reach DB (acceptable)
```

## Bloom Filter Mathematics

### Parameters

| Parameter | Symbol | Typical Value |
|-----------|--------|---------------|
| Expected insertions | n | 100,000 |
| False positive rate | p | 0.01 (1%) |
| Bit array size | m | calculated |
| Hash functions | k | calculated |

### Formulas

```
Optimal bit array size:  m = -n * ln(p) / (ln(2))²
Optimal hash functions:  k = (m/n) * ln(2)

For n=100,000 and p=0.01:
  m ≈ 958,506 bits ≈ 117 KB
  k ≈ 7 hash functions
```

### Memory vs Accuracy Tradeoff

```
┌────────────────┬────────────────┬──────────────────┐
│ False Positive │ Memory (100K)  │ Memory (1M)      │
├────────────────┼────────────────┼──────────────────┤
│     10%        │     60 KB      │     600 KB       │
│      5%        │     78 KB      │     780 KB       │
│      1%        │    117 KB      │    1.17 MB       │
│    0.1%        │    176 KB      │    1.76 MB       │
│   0.01%        │    234 KB      │    2.34 MB       │
└────────────────┴────────────────┴──────────────────┘
```

## RedisBloom Commands

```bash
# Create filter with 1% false positive rate, 100K capacity
BF.RESERVE products-bloom 0.01 100000

# Add single item
BF.ADD products-bloom SKU-001

# Add multiple items
BF.MADD products-bloom SKU-001 SKU-002 SKU-003

# Check single item
BF.EXISTS products-bloom SKU-001
# Returns: 1 (might exist) or 0 (definitely not)

# Check multiple items
BF.MEXISTS products-bloom SKU-001 FAKE-SKU

# Get filter info
BF.INFO products-bloom
```

## Filter Lifecycle

### Initialization (Startup)

```
1. Application starts
2. Query DB for all valid SKUs
3. Populate Bloom filter with all SKUs
4. Ready to serve requests
```

### Adding New Product

```
1. Save product to database
2. Add SKU to Redis cache
3. Add SKU to Bloom filter
4. Publish invalidation event (for other instances)
```

### Periodic Rebuild

```
1. Cron job triggers rebuild (e.g., nightly)
2. Create new temporary filter
3. Load all valid SKUs from DB
4. Swap old filter with new
5. Delete old filter
```

## Failure Scenarios

### Bloom Filter Unavailable

```
If Bloom filter check fails:
  → Log warning
  → Proceed without protection
  → Fall back to normal cache flow
  → DB may receive more queries temporarily
```

### Stale Filter (Missing Keys)

```
If new product not in filter:
  → Bloom check returns "not in set"
  → Valid request incorrectly rejected
  → Solution: Always add to filter on product creation
  → Fallback: Rebuild filter periodically
```

## Metrics

| Metric | Type | Description |
|--------|------|-------------|
| bloom.checks | Counter | Total Bloom filter checks |
| bloom.rejections | Counter | Requests blocked by filter |
| bloom.false_positives | Counter | Invalid keys that passed |
| bloom.lookup_time | Timer | Filter lookup latency |

## Security Considerations

### Rate Limiting

Even with Bloom protection, implement rate limiting:
- Bloom checks are cheap but not free
- Prevent enumeration attacks (trying all possible keys)

### Filter Poisoning

Attacker could try to fill filter with fake keys:
- Solution: Only add keys from trusted sources (DB)
- Never add user-provided keys directly

## Configuration

```yaml
bloom:
  type: redis  # or 'guava' for in-memory

  redis:
    key: products-bloom-filter
    expected-insertions: 100000
    false-positive-rate: 0.01
    rebuild-cron: "0 0 3 * * *"  # 3 AM daily

  guava:
    expected-insertions: 100000
    false-positive-rate: 0.01
```

## Performance Targets

| Operation | Target |
|-----------|--------|
| Guava lookup | < 1μs |
| RedisBloom lookup | < 2ms |
| Filter rebuild (100K items) | < 5s |
| Memory usage (100K items, 1% FP) | ~120 KB |
