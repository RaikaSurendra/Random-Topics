# Chapter 07: Read Replicas & Two-Level Caching

Learn how to implement two-level caching (L1 + L2) and use Redis read replicas for scalability.

## Learning Objectives

- Implement L1 (Caffeine) + L2 (Redis) tiered caching
- Configure Redis read replicas
- Understand replica lag implications
- Implement near-cache pattern

## Project Structure

```
chapter-07-read-replicas/
├── README.md
├── docs/
│   ├── HLD.md
│   └── LLD.md
├── pom.xml
└── src/main/java/com/learning/cache/replicas/
    ├── ReadReplicaApplication.java
    ├── config/
    │   └── TwoLevelCacheConfig.java
    ├── service/
    │   ├── TwoLevelCache.java
    │   ├── ReadReplicaCache.java
    │   └── NearCacheService.java
    └── controller/
        └── ReplicaDemoController.java
```

## Key Concepts

### Two-Level Cache Architecture

```
         Request
            │
            ▼
    ┌──────────────┐
    │   L1 Cache   │  ← Caffeine (in-memory)
    │   ~100μs     │     - Per-instance
    └──────────────┘     - Smallest, fastest
            │ Miss
            ▼
    ┌──────────────┐
    │   L2 Cache   │  ← Redis (distributed)
    │    ~1-5ms    │     - Shared across instances
    └──────────────┘     - Larger, slower
            │ Miss
            ▼
    ┌──────────────┐
    │   Database   │  ← PostgreSQL
    │   ~10-50ms   │     - Source of truth
    └──────────────┘
```

### Performance Targets

| Layer | Latency | Size |
|-------|---------|------|
| L1 | < 100μs | ~10K entries |
| L2 | < 5ms | ~100K entries |
| DB | < 50ms | Unlimited |

## Running This Chapter

```bash
docker-compose up -d postgres redis redis-replica-1 redis-replica-2

cd chapter-07-read-replicas
../mvnw spring-boot:run
```

## Next Chapter

[Chapter 08: Advanced Patterns](../chapter-08-advanced-patterns/README.md)
