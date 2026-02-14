# Chapter 06: NGINX Caching

Learn how to add an HTTP caching layer with NGINX proxy_cache to reduce load on your application.

## Learning Objectives

- Configure NGINX as a caching proxy
- Understand Cache-Control headers
- Implement microcaching for dynamic content
- Monitor cache performance with X-Cache-Status

## Project Structure

```
chapter-06-nginx-caching/
├── README.md
├── docs/
│   ├── HLD.md
│   └── LLD.md
├── pom.xml
└── src/main/java/com/learning/cache/nginx/
    ├── NginxCacheApplication.java
    ├── service/
    │   └── CacheHeaderService.java
    └── controller/
        ├── CacheableController.java
        └── PurgeController.java
```

## Key Concepts

### Cache-Control Headers

```
Cache-Control: public, max-age=3600      # Cache for 1 hour
Cache-Control: private, no-cache         # Don't cache
Cache-Control: public, s-maxage=60       # Proxy caches for 60s
```

### X-Cache-Status Values

| Value | Meaning |
|-------|---------|
| HIT | Served from cache |
| MISS | Not in cache, fetched from origin |
| EXPIRED | Cached but expired |
| STALE | Served stale while updating |
| BYPASS | Cache bypassed |

## Running This Chapter

```bash
docker-compose up -d postgres redis nginx

cd chapter-06-nginx-caching
../mvnw spring-boot:run
```

## NGINX Configuration

See `nginx/nginx.conf` for the complete proxy_cache configuration.

## Next Chapter

[Chapter 07: Read Replicas](../chapter-07-read-replicas/README.md)
