# Chapter 08: Advanced Caching Patterns

Learn production-ready caching patterns including cache warming, hot key detection, TTL jitter, and versioned caching.

## Learning Objectives

- Implement cache warming on startup
- Detect and mitigate hot keys
- Use TTL jitter to prevent synchronized expiration
- Implement versioned caching for zero-downtime deployments

## Project Structure

```
chapter-08-advanced-patterns/
├── README.md
├── docs/
│   ├── HLD.md
│   └── LLD.md
├── pom.xml
└── src/main/java/com/learning/cache/advanced/
    ├── AdvancedPatternsApplication.java
    ├── service/
    │   ├── CacheWarmer.java
    │   ├── HotKeyDetector.java
    │   ├── HotKeyMitigation.java
    │   ├── JitteredTTLService.java
    │   └── VersionedCacheService.java
    └── controller/
        └── AdvancedDemoController.java
```

## Key Concepts

### Cache Warming

Pre-load frequently accessed data on startup to avoid cold cache latency.

### Hot Key Detection

Monitor access patterns to identify keys with disproportionate traffic.

### TTL Jitter

Add randomness to TTL values to prevent thundering herd from synchronized expiration.

```java
long baseTtl = 60_000; // 60 seconds
long jitter = ThreadLocalRandom.current().nextLong(baseTtl / 10); // ±10%
long actualTtl = baseTtl + jitter - (baseTtl / 20);
```

### Versioned Cache

Prefix cache keys with version to enable zero-downtime migrations.

```
v1::products::SKU-001  ← Old version
v2::products::SKU-001  ← New version (after deployment)
```

## Running This Chapter

```bash
docker-compose up -d postgres redis

cd chapter-08-advanced-patterns
../mvnw spring-boot:run
```

## Verification

| Pattern | Verification |
|---------|--------------|
| Warming | Startup loads top 100 products |
| Hot Keys | Detected after 1000 requests |
| TTL Jitter | Expiration spread over time |
| Versioning | Zero downtime switch |
