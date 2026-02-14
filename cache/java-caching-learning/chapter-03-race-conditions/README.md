# Chapter 03: Cache Race Conditions

Learn to identify and solve common cache concurrency problems including thundering herd, cache stampede, and dogpile effects.

## Learning Objectives

By the end of this chapter, you will:
- Understand the thundering herd problem
- Implement mutex-based cache locking with Redisson
- Use probabilistic early refresh (XFetch algorithm)
- Implement request coalescing (singleflight pattern)
- Compare tradeoffs between solutions

## The Problem: Thundering Herd

When a popular cache key expires, many concurrent requests may:
1. All find the cache empty simultaneously
2. All attempt to regenerate the cached value
3. All hit the database at once
4. Overwhelm backend systems

```
Time: T=0 (Cache expires)
┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐
│ Req1 │ │ Req2 │ │ Req3 │ │ ... │ │Req1000│
└──┬───┘ └──┬───┘ └──┬───┘ └──┬───┘ └──┬────┘
   │        │        │        │        │
   └────────┴────────┴────────┴────────┘
                     │
                     ▼
              ┌─────────────┐
              │   Cache     │
              │   MISS!     │
              └─────────────┘
                     │
        1000 requests hit database!
                     ▼
              ┌─────────────┐
              │  Database   │
              │  OVERLOAD!  │
              └─────────────┘
```

## Solutions Overview

| Solution | Pros | Cons | Best For |
|----------|------|------|----------|
| Mutex Lock | Guaranteed single writer | Lock wait latency | Critical data |
| Probabilistic | No locks, smooth refresh | Occasional extra queries | High traffic |
| Coalescing | Deduplicates in-flight | Memory for pending | API aggregation |

## Project Structure

```
chapter-03-race-conditions/
├── README.md
├── docs/
│   ├── HLD.md
│   └── LLD.md
├── pom.xml
└── src/main/java/com/learning/cache/race/
    ├── RaceConditionApplication.java
    ├── config/
    │   └── RedissonConfig.java
    ├── service/
    │   ├── VulnerableCacheService.java
    │   ├── MutexCacheService.java
    │   ├── ProbabilisticCacheService.java
    │   └── CoalescingCacheService.java
    ├── controller/
    │   └── RaceConditionDemoController.java
    └── demo/
        └── RaceConditionVisualizer.java
```

## Running This Chapter

```bash
docker-compose up -d postgres redis

cd chapter-03-race-conditions
../mvnw spring-boot:run
```

## API Endpoints

| Endpoint | Description |
|----------|-------------|
| `POST /api/demo/stampede` | Simulate cache stampede |
| `GET /api/demo/mutex/{sku}` | Get with mutex lock |
| `GET /api/demo/probabilistic/{sku}` | Get with early refresh |
| `GET /api/demo/coalescing/{sku}` | Get with request coalescing |
| `GET /api/demo/compare` | Compare all strategies |

## Key Concepts

### 1. Mutex Lock (Distributed Lock)

Only one thread can refresh the cache:

```java
public ProductDTO getWithMutex(String sku) {
    ProductDTO cached = cache.get(sku);
    if (cached != null) return cached;

    RLock lock = redisson.getLock("lock:product:" + sku);
    try {
        if (lock.tryLock(5, 30, TimeUnit.SECONDS)) {
            // Double-check after acquiring lock
            cached = cache.get(sku);
            if (cached != null) return cached;

            // Only this thread loads from DB
            ProductDTO product = loadFromDatabase(sku);
            cache.put(sku, product);
            return product;
        }
        // Lock not acquired - another thread is loading
        return waitForCache(sku);
    } finally {
        if (lock.isHeldByCurrentThread()) {
            lock.unlock();
        }
    }
}
```

### 2. Probabilistic Early Refresh (XFetch)

Refresh before expiration with increasing probability:

```java
public ProductDTO getWithProbabilistic(String sku) {
    CacheEntry entry = cache.getEntry(sku);

    if (entry != null) {
        // Calculate refresh probability based on remaining TTL
        double remainingTtl = entry.getExpirationTime() - System.currentTimeMillis();
        double probability = calculateRefreshProbability(remainingTtl);

        if (Math.random() < probability) {
            // Async refresh in background
            asyncRefresh(sku);
        }
        return entry.getValue();
    }

    // Cache miss - must load synchronously
    return loadAndCache(sku);
}
```

### 3. Request Coalescing (Singleflight)

Deduplicate concurrent requests for the same key:

```java
private final Map<String, CompletableFuture<ProductDTO>> inFlight = new ConcurrentHashMap<>();

public ProductDTO getWithCoalescing(String sku) {
    ProductDTO cached = cache.get(sku);
    if (cached != null) return cached;

    // All concurrent requests for same SKU share one future
    CompletableFuture<ProductDTO> future = inFlight.computeIfAbsent(sku,
        key -> loadAsync(key).whenComplete((v, e) -> inFlight.remove(key)));

    return future.join();
}
```

## Demonstrations

### Demo 1: Vulnerable Implementation

```bash
# Simulate 1000 concurrent requests for expired key
curl -X POST "http://localhost:8080/api/demo/stampede?requests=1000&strategy=vulnerable"
```

Expected output:
```json
{
  "strategy": "vulnerable",
  "requests": 1000,
  "dbQueries": 847,
  "avgLatencyMs": 156.3,
  "conclusion": "Database overwhelmed with 847 concurrent queries!"
}
```

### Demo 2: Mutex Lock

```bash
curl -X POST "http://localhost:8080/api/demo/stampede?requests=1000&strategy=mutex"
```

Expected output:
```json
{
  "strategy": "mutex",
  "requests": 1000,
  "dbQueries": 1,
  "avgLatencyMs": 45.2,
  "lockWaitMs": 32.1,
  "conclusion": "Only 1 database query! Others waited for lock."
}
```

### Demo 3: Probabilistic Refresh

```bash
curl -X POST "http://localhost:8080/api/demo/stampede?requests=1000&strategy=probabilistic"
```

Expected output:
```json
{
  "strategy": "probabilistic",
  "requests": 1000,
  "dbQueries": 3,
  "backgroundRefreshes": 2,
  "avgLatencyMs": 2.1,
  "conclusion": "Smooth refresh with minimal latency impact."
}
```

### Demo 4: Compare All Strategies

```bash
curl "http://localhost:8080/api/demo/compare?requests=500"
```

## Configuration

```yaml
race-condition:
  mutex:
    wait-time: 5000      # Max time to wait for lock (ms)
    lease-time: 30000    # Lock auto-release time (ms)

  probabilistic:
    beta: 1.0            # Refresh aggressiveness (higher = earlier refresh)
    delta: 5000          # Refresh window before expiration (ms)

  coalescing:
    timeout: 10000       # Max wait for in-flight request (ms)
```

## Verification Checklist

- [ ] Mutex: Only 1 DB query under 1000 concurrent requests
- [ ] Probabilistic: Background refreshes prevent expiration spikes
- [ ] Coalescing: Concurrent requests share single DB query
- [ ] All strategies: Response time < 100ms p99

## Key Takeaways

1. **Default caching is vulnerable** - Expiration creates race conditions
2. **Mutex guarantees single writer** - But adds lock waiting latency
3. **Probabilistic prevents spikes** - Smooth refresh curve, no locks
4. **Coalescing deduplicates** - Best when many identical requests arrive

## Next Chapter

[Chapter 04: Distributed Patterns](../chapter-04-distributed-patterns/README.md) - Implement consistent hashing and sharding.
