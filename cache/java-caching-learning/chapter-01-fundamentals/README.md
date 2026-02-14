# Chapter 01: Caching Fundamentals

Learn the core concepts of caching through hands-on implementation of the cache-aside pattern with Redis and Spring Boot.

## Learning Objectives

By the end of this chapter, you will:
- Understand why caching improves application performance
- Implement the cache-aside (lazy-loading) pattern
- Configure Redis with Spring Boot
- Use `@Cacheable`, `@CacheEvict`, and `@CachePut` annotations
- Measure and visualize cache hit/miss ratios
- Configure TTL (Time-To-Live) for cache entries

## Key Concepts

### Cache-Aside Pattern

The cache-aside pattern (also known as lazy-loading) works as follows:

```
┌──────────┐         ┌──────────┐         ┌──────────┐
│  Client  │────────▶│   App    │────────▶│  Cache   │
└──────────┘         └──────────┘         └──────────┘
                           │                    │
                           │    Cache Miss      │
                           │◀───────────────────┤
                           │                    │
                           ▼                    │
                     ┌──────────┐              │
                     │ Database │              │
                     └──────────┘              │
                           │                    │
                           │ Store in Cache     │
                           │───────────────────▶│
                           │                    │
```

1. Check cache first
2. On cache miss, query database
3. Store result in cache
4. Return result

### TTL (Time-To-Live)

TTL determines how long cached data remains valid:
- **Short TTL (seconds)**: Fresh data, more DB queries
- **Long TTL (hours/days)**: Fewer queries, potentially stale data

## Project Structure

```
chapter-01-fundamentals/
├── README.md
├── docs/
│   ├── HLD.md              # High-Level Design
│   └── LLD.md              # Low-Level Design
├── pom.xml
└── src/main/java/com/learning/cache/fundamentals/
    ├── CacheFundamentalsApplication.java
    ├── config/
    │   └── RedisConfig.java
    ├── service/
    │   ├── ProductService.java          # Uncached (baseline)
    │   └── CachedProductService.java    # With caching
    ├── controller/
    │   ├── ProductController.java
    │   └── MetricsController.java
    └── repository/
        └── ProductRepository.java
```

## Running This Chapter

### Prerequisites
- Docker running
- Java 21 installed

### Start Services

```bash
# From project root
docker-compose up -d postgres redis

# Run the application
cd chapter-01-fundamentals
../mvnw spring-boot:run
```

### API Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /api/products/{sku}` | Get product (cached) |
| `GET /api/products/uncached/{sku}` | Get product (uncached, baseline) |
| `PUT /api/products/{sku}` | Update product (evicts cache) |
| `DELETE /api/cache/{sku}` | Manually evict cache entry |
| `GET /api/metrics/cache` | View cache statistics |

## Demonstrations

### Demo 1: Cold vs Warm Cache

```bash
# First request (cold cache) - slower
time curl http://localhost:8080/api/products/SKU-001

# Second request (warm cache) - faster
time curl http://localhost:8080/api/products/SKU-001
```

### Demo 2: TTL Expiration

```bash
# Get product (caches it)
curl http://localhost:8080/api/products/SKU-001

# Wait for TTL to expire (default: 60 seconds)
sleep 65

# Next request will be a cache miss
curl http://localhost:8080/api/products/SKU-001
```

### Demo 3: Manual Eviction

```bash
# Cache the product
curl http://localhost:8080/api/products/SKU-001

# Evict from cache
curl -X DELETE http://localhost:8080/api/cache/SKU-001

# Next request will be a cache miss
curl http://localhost:8080/api/products/SKU-001
```

### Demo 4: Cache Metrics

```bash
# Run several requests
for i in {1..10}; do curl -s http://localhost:8080/api/products/SKU-001; done

# View metrics
curl http://localhost:8080/api/metrics/cache
```

Expected output:
```json
{
  "cacheName": "products",
  "hits": 9,
  "misses": 1,
  "hitRatio": 0.9,
  "hitRatioPercentage": "90.00%"
}
```

## Configuration

### application.yml

```yaml
spring:
  data:
    redis:
      host: localhost
      port: 6379

cache:
  products:
    ttl: 60  # seconds
```

## Verification Checklist

- [ ] Cached request is significantly faster than uncached
- [ ] Cache entries expire after TTL
- [ ] Manual eviction works
- [ ] Hit ratio > 90% after warmup
- [ ] Metrics endpoint shows accurate statistics

## Key Takeaways

1. **Caching reduces latency** - Cache hits are orders of magnitude faster than database queries
2. **TTL is a tradeoff** - Balance freshness vs. performance
3. **Measure everything** - Don't assume, use metrics
4. **Cache aside is simple but powerful** - Good starting point for most applications

## Next Chapter

[Chapter 02: Bloom Filters](../chapter-02-bloom-filters/README.md) - Protect your cache from penetration attacks.
