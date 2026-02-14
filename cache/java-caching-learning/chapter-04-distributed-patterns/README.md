# Chapter 04: Distributed Caching Patterns

Learn how to distribute cache across multiple nodes using consistent hashing, handle node failures, and implement circuit breakers.

## Learning Objectives

- Understand consistent hashing algorithm
- Implement key-based routing to cache nodes
- Handle node failures gracefully with circuit breakers
- Use virtual nodes for better load distribution

## Project Structure

```
chapter-04-distributed-patterns/
├── README.md
├── docs/
│   ├── HLD.md
│   └── LLD.md
├── pom.xml
└── src/main/java/com/learning/cache/distributed/
    ├── DistributedCacheApplication.java
    ├── config/
    │   └── DistributedCacheConfig.java
    ├── service/
    │   ├── ConsistentHashRing.java
    │   ├── ShardedCacheService.java
    │   └── CircuitBreakerCache.java
    └── controller/
        └── DistributedCacheDemoController.java
```

## Key Concepts

### Consistent Hashing

```
         ┌───────────────────────────────────────┐
         │         Consistent Hash Ring          │
         │                                       │
         │          Node A (0-85)                │
         │      ╱                    ╲           │
         │     ╱                      ╲          │
         │    ╱                        ╲         │
         │   ╱                          ╲        │
         │  Node C (171-255)    Node B (86-170) │
         └───────────────────────────────────────┘
```

### Virtual Nodes

Virtual nodes improve distribution by assigning multiple positions on the ring to each physical node.

## Running This Chapter

```bash
docker-compose up -d postgres redis redis-replica-1 redis-replica-2

cd chapter-04-distributed-patterns
../mvnw spring-boot:run
```

## API Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /api/distributed/{key}` | Get with consistent hashing |
| `GET /api/distributed/node/{key}` | Show which node owns key |
| `POST /api/distributed/failover` | Simulate node failure |
| `GET /api/distributed/ring` | Visualize hash ring |

## Next Chapter

[Chapter 05: Cache Consistency](../chapter-05-consistency/README.md)
