# System-Wide High Level Design (HLD)

## Overview

This document describes the overall architecture of the Java Caching Learning Project, including all components, their interactions, and design decisions.

## System Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Client Requests                                 │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              NGINX (Port 80)                                 │
│                         HTTP Caching Proxy Layer                             │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │  proxy_cache    │    Cache-Control    │    X-Cache-Status           │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Spring Boot Application (8080)                       │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │   Chapter   │  │   Chapter   │  │   Chapter   │  │    ...      │        │
│  │     01      │  │     02      │  │     03      │  │             │        │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘        │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                         L1 Cache (Caffeine)                          │    │
│  │                      In-Memory, Microsecond Access                   │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────┘
           │                    │                    │
           ▼                    ▼                    ▼
┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│  Redis Primary   │  │ Redis Replica 1  │  │ Redis Replica 2  │
│    (6379)        │  │     (6380)       │  │     (6381)       │
│                  │  │                  │  │                  │
│  RedisBloom      │  │   Read-Only      │  │   Read-Only      │
│  RedisJSON       │  │   Replica        │  │   Replica        │
└──────────────────┘  └──────────────────┘  └──────────────────┘
           │                    ▲                    ▲
           │                    │    Replication     │
           └────────────────────┴────────────────────┘
           │
           ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         RabbitMQ (5672/15672)                                │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │  cache.invalidation.exchange  →  cache.invalidation.queue           │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      │ Fanout to all instances
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                       PostgreSQL (5432)                                      │
│                       Source of Truth                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │  products    │    users    │    (other tables)                      │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                         Observability Stack                                  │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐             │
│  │   Prometheus    │  │     Grafana     │  │  RedisInsight   │             │
│  │     (9090)      │  │     (3000)      │  │     (8001)      │             │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Component Descriptions

### 1. NGINX (HTTP Caching Proxy)

**Purpose**: First line of caching, reduces load on application servers.

**Key Features**:
- `proxy_cache` for response caching
- `proxy_cache_lock` to prevent stampede
- `X-Cache-Status` header for debugging
- Microcaching for semi-dynamic content

**Configuration Highlights**:
```nginx
proxy_cache_path /var/cache/nginx levels=1:2 keys_zone=api_cache:10m max_size=1g;
proxy_cache_lock on;
proxy_cache_use_stale error timeout updating;
```

### 2. Spring Boot Application

**Purpose**: Main application logic with multi-level caching.

**Key Features**:
- L1 Cache: Caffeine (in-memory, microsecond access)
- L2 Cache: Redis (distributed, millisecond access)
- Bloom filters for penetration protection
- Race condition mitigation patterns
- Metrics with Micrometer

**Cache Layers**:
```
Request → L1 (Caffeine) → L2 (Redis) → Database (PostgreSQL)
            ~100μs          ~1-5ms          ~10-50ms
```

### 3. Redis Primary + Replicas

**Purpose**: Distributed caching with high availability.

**Topology**:
- 1 Primary (read/write)
- 2 Replicas (read-only)
- Asynchronous replication

**Features Used**:
- String operations for simple caching
- RedisBloom for Bloom filters
- Pub/Sub for invalidation (alternative to RabbitMQ)
- Distributed locks (Redisson)

### 4. RabbitMQ

**Purpose**: Event-driven cache invalidation across instances.

**Exchange/Queue Setup**:
```
Exchange: cache.invalidation.exchange (fanout)
    └── Queue: cache.invalidation.queue.{instance-id}
```

**Message Format**:
```json
{
  "type": "INVALIDATE",
  "cacheName": "products",
  "key": "SKU-001",
  "timestamp": 1704067200000
}
```

### 5. PostgreSQL

**Purpose**: Source of truth, persistent storage.

**Tables**:
- `products` - Main demo entity
- `users` - Additional examples

### 6. Observability

**Prometheus**: Scrapes metrics from:
- Application (`/actuator/prometheus`)
- Redis (via exporter)
- NGINX (stub_status)

**Grafana**: Dashboards for:
- Cache hit/miss ratios
- Latency percentiles
- Request rates
- Error rates

## Data Flow Patterns

### Pattern 1: Cache-Aside (Read)

```
1. Client requests product SKU-001
2. NGINX checks proxy_cache → MISS
3. Application checks L1 (Caffeine) → MISS
4. Application checks L2 (Redis) → MISS
5. Application queries PostgreSQL
6. Application stores in Redis (L2)
7. Application stores in Caffeine (L1)
8. Response returned, NGINX caches
9. Next request: NGINX HIT
```

### Pattern 2: Cache Invalidation (Write)

```
1. Client updates product SKU-001
2. Application updates PostgreSQL
3. Application deletes from L1 (Caffeine)
4. Application deletes from L2 (Redis)
5. Application publishes to RabbitMQ
6. Other instances receive event
7. Other instances delete from their L1
```

### Pattern 3: Bloom Filter Protection

```
1. Client requests non-existent SKU-999
2. Bloom filter check → NOT IN SET
3. Request rejected immediately
4. No cache lookup, no DB query
```

### Pattern 4: Mutex for Race Condition

```
1. Cache expires for hot key
2. 1000 concurrent requests arrive
3. First request acquires lock
4. Other 999 wait on lock
5. First request loads from DB, caches result
6. Lock released
7. 999 requests get cached value
8. Result: 1 DB query instead of 1000
```

## Consistency Model

### Eventual Consistency

- Default mode for performance
- Inconsistency window: typically < 100ms
- Acceptable for most read-heavy workloads

### Strong Consistency Options

- Write-through: Sync write to cache + DB
- Read-through with mutex: One reader populates cache
- TTL = 0: No caching (for critical operations)

## Failure Handling

### Redis Primary Failure

1. Circuit breaker activates
2. Fallback to local cache (Caffeine)
3. If local miss, query DB directly
4. Log alert for operations

### RabbitMQ Failure

1. Local invalidation continues
2. Messages queued (if persistent)
3. Eventual consistency restored on recovery
4. Fallback: TTL-based expiration

### PostgreSQL Failure

1. Application serves cached data (stale)
2. Writes fail immediately
3. Health check reports unhealthy
4. Alert operations team

## Security Considerations

### Network Isolation
- All services on internal Docker network
- Only NGINX and Grafana exposed externally
- Redis and PostgreSQL not directly accessible

### Credentials
- Environment variables for secrets
- Different credentials per environment
- No hardcoded passwords in code

### Rate Limiting
- NGINX rate limiting for public endpoints
- Application-level throttling for sensitive operations

## Scalability

### Horizontal Scaling

```
                    Load Balancer
                          │
        ┌─────────────────┼─────────────────┐
        ▼                 ▼                 ▼
   ┌─────────┐       ┌─────────┐       ┌─────────┐
   │  App 1  │       │  App 2  │       │  App 3  │
   └─────────┘       └─────────┘       └─────────┘
        │                 │                 │
        └─────────────────┼─────────────────┘
                          ▼
                    Redis Cluster
```

### Redis Cluster (Production)
- 6+ nodes for high availability
- Automatic sharding
- Failover within seconds

## Performance Targets

| Metric | Target | Measurement |
|--------|--------|-------------|
| L1 Hit Latency | < 100μs | p99 |
| L2 Hit Latency | < 5ms | p99 |
| DB Query Latency | < 50ms | p99 |
| Cache Hit Ratio | > 90% | After warmup |
| Invalidation Propagation | < 100ms | p99 |

## Monitoring Alerts

| Alert | Condition | Severity |
|-------|-----------|----------|
| Low Hit Ratio | < 80% for 5 min | Warning |
| High Latency | p99 > 100ms | Warning |
| Redis Down | Health check fails | Critical |
| Cache Miss Spike | > 50% increase | Warning |
| Invalidation Lag | > 500ms | Warning |
