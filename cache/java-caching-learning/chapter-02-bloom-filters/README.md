# Chapter 02: Bloom Filters

Learn how to protect your cache and database from penetration attacks using Bloom filters.

## Learning Objectives

By the end of this chapter, you will:
- Understand the cache penetration problem
- Implement Bloom filters using Guava (in-memory)
- Implement Bloom filters using RedisBloom (distributed)
- Balance false positive rate vs memory usage
- Rebuild filters when data changes

## The Problem: Cache Penetration

Cache penetration occurs when requests for **non-existent keys** bypass the cache and hit the database:

```
Attacker → GET /products/FAKE-SKU-12345
         → Cache MISS (key doesn't exist)
         → Database query (returns nothing)
         → Repeat with millions of fake keys
         → Database overwhelmed!
```

### Why It Matters

- Malicious actors can DoS your database
- Accidental load from buggy clients
- Scrapers probing for valid IDs

## Solution: Bloom Filter

A Bloom filter is a probabilistic data structure that answers:
- **"Definitely NOT in set"** - 100% accurate
- **"Probably in set"** - May have false positives

```
┌─────────────────────────────────────────────────────────────┐
│                     Request Flow                            │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
                    ┌─────────────────┐
                    │  Bloom Filter   │
                    │  Check          │
                    └─────────────────┘
                              │
              ┌───────────────┴───────────────┐
              │                               │
        NOT IN SET                      MAYBE IN SET
              │                               │
              ▼                               ▼
      ┌─────────────┐                ┌─────────────┐
      │   Return    │                │   Check     │
      │   404       │                │   Cache     │
      │ immediately │                │   & DB      │
      └─────────────┘                └─────────────┘
```

## Project Structure

```
chapter-02-bloom-filters/
├── README.md
├── docs/
│   ├── HLD.md
│   └── LLD.md
├── pom.xml
└── src/main/java/com/learning/cache/bloom/
    ├── BloomFilterApplication.java
    ├── config/
    │   ├── BloomFilterConfig.java
    │   └── RedisBloomConfig.java
    ├── service/
    │   ├── InMemoryBloomFilter.java
    │   ├── RedisBloomFilter.java
    │   └── BloomProtectedService.java
    ├── controller/
    │   └── BloomDemoController.java
    └── demo/
        └── PenetrationAttackDemo.java
```

## Running This Chapter

```bash
# Start Redis Stack (includes RedisBloom)
docker-compose up -d redis postgres

# Run the application
cd chapter-02-bloom-filters
../mvnw spring-boot:run
```

## API Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /api/products/{sku}` | Get product (with Bloom protection) |
| `GET /api/products/unprotected/{sku}` | Get without Bloom filter |
| `POST /api/bloom/rebuild` | Rebuild Bloom filter |
| `GET /api/bloom/stats` | View filter statistics |
| `POST /api/demo/attack` | Simulate penetration attack |

## Key Concepts

### False Positives vs Memory

| Filter Size | Expected Items | False Positive Rate |
|-------------|----------------|---------------------|
| 1 MB | 100,000 | ~1% |
| 10 MB | 100,000 | ~0.01% |
| 100 MB | 100,000 | ~0.0001% |

### Guava Bloom Filter (In-Memory)

```java
BloomFilter<String> filter = BloomFilter.create(
    Funnels.stringFunnel(Charset.defaultCharset()),
    expectedInsertions,      // e.g., 100_000
    falsePositiveRate        // e.g., 0.01 (1%)
);

// Add item
filter.put("SKU-001");

// Check item
if (filter.mightContain("SKU-999")) {
    // Might be in set (or false positive)
} else {
    // Definitely NOT in set - reject immediately
}
```

### RedisBloom (Distributed)

```bash
# Create filter
BF.RESERVE products-filter 0.01 100000

# Add items
BF.ADD products-filter SKU-001

# Check item
BF.EXISTS products-filter SKU-001  # Returns 1 (might exist)
BF.EXISTS products-filter FAKE     # Returns 0 (definitely not)
```

## Demonstrations

### Demo 1: Attack Without Protection

```bash
# Simulate 1000 requests with fake SKUs (no protection)
curl -X POST "http://localhost:8080/api/demo/attack?requests=1000&protected=false"
```

Expected output:
```json
{
  "requests": 1000,
  "protected": false,
  "dbQueries": 1000,
  "avgLatencyMs": 45.2,
  "conclusion": "All requests hit database!"
}
```

### Demo 2: Attack With Bloom Protection

```bash
# Same attack with Bloom filter protection
curl -X POST "http://localhost:8080/api/demo/attack?requests=1000&protected=true"
```

Expected output:
```json
{
  "requests": 1000,
  "protected": true,
  "dbQueries": 12,
  "rejectedByBloom": 988,
  "avgLatencyMs": 0.5,
  "conclusion": "99%+ requests blocked by Bloom filter!"
}
```

### Demo 3: False Positive Rate

```bash
# Measure actual false positive rate
curl "http://localhost:8080/api/demo/false-positives?tests=10000"
```

Expected output:
```json
{
  "tests": 10000,
  "falsePositives": 98,
  "falsePositiveRate": "0.98%",
  "configuredRate": "1.00%",
  "conclusion": "Within expected bounds"
}
```

## Configuration

```yaml
bloom:
  # In-memory (Guava)
  guava:
    expected-insertions: 100000
    false-positive-rate: 0.01

  # Distributed (RedisBloom)
  redis:
    key: products-bloom-filter
    expected-insertions: 100000
    false-positive-rate: 0.01
```

## Verification Checklist

- [ ] Bloom filter blocks 99%+ of invalid keys
- [ ] False positive rate matches configuration
- [ ] Filter rebuilds correctly on data changes
- [ ] RedisBloom works across multiple app instances
- [ ] Performance: Bloom check < 1ms

## Key Takeaways

1. **Bloom filters prevent unnecessary work** - Block invalid requests before hitting cache/DB
2. **False positives are OK** - A few extra DB queries are better than millions
3. **Choose filter type by deployment** - In-memory for single instance, Redis for distributed
4. **Rebuild periodically** - Keep filter in sync with actual data

## Next Chapter

[Chapter 03: Race Conditions](../chapter-03-race-conditions/README.md) - Handle thundering herd and cache stampede.
