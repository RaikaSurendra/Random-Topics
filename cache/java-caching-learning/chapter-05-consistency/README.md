# Chapter 05: Cache Consistency

Learn how to maintain cache consistency across distributed systems using RabbitMQ for event-driven invalidation and various write patterns.

## Learning Objectives

- Understand consistency vs availability tradeoffs
- Implement event-driven cache invalidation with RabbitMQ
- Compare write-through, write-behind, and refresh-ahead patterns
- Measure inconsistency windows

## Project Structure

```
chapter-05-consistency/
├── README.md
├── docs/
│   ├── HLD.md
│   └── LLD.md
├── pom.xml
└── src/main/java/com/learning/cache/consistency/
    ├── ConsistencyApplication.java
    ├── config/
    │   └── RabbitMQConfig.java
    ├── service/
    │   ├── WriteThroughCache.java
    │   ├── WriteBehindCache.java
    │   └── RefreshAheadCache.java
    ├── messaging/
    │   ├── CacheInvalidationPublisher.java
    │   └── CacheInvalidationListener.java
    └── controller/
        └── ConsistencyDemoController.java
```

## Key Concepts

### Write-Through
Synchronously update cache and database together.

### Write-Behind (Write-Back)
Update cache immediately, queue database writes for batching.

### Refresh-Ahead
Background refresh before expiration.

## Running This Chapter

```bash
docker-compose up -d postgres redis rabbitmq

cd chapter-05-consistency
../mvnw spring-boot:run
```

## Next Chapter

[Chapter 06: NGINX Caching](../chapter-06-nginx-caching/README.md)
