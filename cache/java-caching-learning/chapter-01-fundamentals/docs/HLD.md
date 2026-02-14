# Chapter 01: Caching Fundamentals - High Level Design

## Overview

This chapter implements the cache-aside pattern using Redis as a distributed cache layer between the application and PostgreSQL database.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Client Request                          │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Spring Boot Application                       │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                   ProductController                      │    │
│  │         /api/products/{sku}  /api/metrics/cache          │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                │                                 │
│                                ▼                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                 CachedProductService                     │    │
│  │           @Cacheable  @CacheEvict  @CachePut            │    │
│  └─────────────────────────────────────────────────────────┘    │
│         │                                      │                 │
│         │ Cache Check                          │ Cache Miss      │
│         ▼                                      ▼                 │
│  ┌─────────────┐                      ┌─────────────────┐       │
│  │   Redis     │                      │  ProductService │       │
│  │   Cache     │                      │   (Uncached)    │       │
│  │             │                      └─────────────────┘       │
│  │ TTL: 60s    │                               │                │
│  └─────────────┘                               ▼                │
│                                        ┌─────────────────┐       │
│                                        │ ProductRepository│      │
│                                        └─────────────────┘       │
└─────────────────────────────────────────────────────────────────┘
                                                 │
                                                 ▼
                              ┌─────────────────────────────────┐
                              │          PostgreSQL             │
                              │        (Source of Truth)        │
                              └─────────────────────────────────┘
```

## Components

### 1. Controller Layer

| Component | Responsibility |
|-----------|----------------|
| ProductController | REST endpoints for product CRUD |
| MetricsController | Cache statistics and hit/miss ratios |

### 2. Service Layer

| Component | Responsibility |
|-----------|----------------|
| ProductService | Direct database access (baseline) |
| CachedProductService | Cache-aside implementation |

### 3. Configuration

| Component | Responsibility |
|-----------|----------------|
| RedisConfig | Redis connection and cache manager setup |

### 4. Metrics

| Component | Responsibility |
|-----------|----------------|
| CacheMetrics | Hit/miss counters with Micrometer |

## Data Flow

### Read Operation (Cache Hit)

```
1. Client → GET /api/products/SKU-001
2. Controller → CachedProductService.getProduct("SKU-001")
3. @Cacheable checks Redis → FOUND
4. Return cached ProductDTO
5. Record cache HIT metric
```

**Latency**: ~1-5ms

### Read Operation (Cache Miss)

```
1. Client → GET /api/products/SKU-001
2. Controller → CachedProductService.getProduct("SKU-001")
3. @Cacheable checks Redis → NOT FOUND
4. Call ProductService.getProduct("SKU-001")
5. ProductRepository queries PostgreSQL
6. Result stored in Redis with TTL
7. Return ProductDTO
8. Record cache MISS metric
```

**Latency**: ~20-50ms

### Write Operation (Eviction)

```
1. Client → PUT /api/products/SKU-001
2. Controller → CachedProductService.updateProduct(dto)
3. ProductRepository.save() to PostgreSQL
4. @CacheEvict removes from Redis
5. Record cache EVICTION metric
```

## Caching Strategy

### Cache Configuration

| Setting | Value | Rationale |
|---------|-------|-----------|
| TTL | 60 seconds | Balance freshness vs. hit ratio |
| Cache Name | "products" | Single cache for all products |
| Key | SKU | Unique product identifier |
| Serialization | JSON | Human-readable, debuggable |

### Eviction Triggers

1. **TTL Expiration** - Automatic after 60 seconds
2. **Explicit Eviction** - On product update/delete
3. **Manual Eviction** - Via admin endpoint

## Metrics

### Tracked Metrics

| Metric | Type | Description |
|--------|------|-------------|
| cache.hits | Counter | Successful cache lookups |
| cache.misses | Counter | Failed cache lookups |
| cache.evictions | Counter | Cache entry removals |
| cache.latency | Timer | Operation timing |

### Calculated Metrics

- **Hit Ratio** = hits / (hits + misses)
- **Miss Ratio** = 1 - hit_ratio
- **Throughput** = requests / time

## Non-Functional Requirements

### Performance Targets

| Metric | Target |
|--------|--------|
| Cache Hit Latency | < 5ms (p99) |
| Cache Miss Latency | < 50ms (p99) |
| Hit Ratio | > 90% (after warmup) |

### Availability

- Redis down: Fall back to database (graceful degradation)
- Database down: Serve cached data (stale reads)

## Failure Scenarios

### Redis Unavailable

```
1. Application attempts Redis operation
2. Connection timeout/failure detected
3. Fallback to direct database query
4. Log warning for monitoring
5. Circuit breaker may activate (future chapter)
```

### PostgreSQL Unavailable

```
1. Application attempts database query
2. Connection failure detected
3. Return cached data if available (stale)
4. Return error if cache miss
5. Log error for alerting
```

## Security Considerations

- Redis authentication enabled (password)
- Network isolation (internal Docker network)
- No sensitive data in cache keys

## Future Enhancements (Other Chapters)

- Bloom filters for non-existent keys
- Distributed locking for race conditions
- Cache invalidation via events
- Two-level caching (L1 + L2)
