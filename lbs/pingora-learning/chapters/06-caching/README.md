# Chapter 06: Caching

## Learning Objectives

By the end of this chapter, you will:
- Understand HTTP caching headers
- Implement response caching in Pingora
- Configure cache policies
- Handle cache invalidation

---

## 6.1 Why Cache at the Proxy?

```
Without Cache:
  Request → Proxy → Backend (100ms) → Response
  Request → Proxy → Backend (100ms) → Response
  Request → Proxy → Backend (100ms) → Response

With Cache:
  Request → Proxy → Backend (100ms) → Response (cached)
  Request → Proxy → Cache Hit (1ms) → Response
  Request → Proxy → Cache Hit (1ms) → Response
```

Benefits:
- **Faster responses** - cache hits are instant
- **Reduced backend load** - fewer requests reach backend
- **Better availability** - serve stale cache if backend is down

---

## 6.2 HTTP Caching Basics

### Key Headers

| Header | Purpose | Example |
|--------|---------|---------|
| `Cache-Control` | Caching rules | `max-age=3600` |
| `ETag` | Content fingerprint | `"abc123"` |
| `Last-Modified` | When content changed | `Tue, 01 Jan 2024...` |
| `Vary` | Cache key variations | `Accept-Encoding` |

### Cache-Control Directives

```
Cache-Control: max-age=3600        # Cache for 1 hour
Cache-Control: no-cache            # Validate before using
Cache-Control: no-store            # Never cache
Cache-Control: private             # Only browser can cache
Cache-Control: public              # Proxy can cache
Cache-Control: s-maxage=600        # Proxy cache time (overrides max-age)
```

---

## 6.3 Pingora Cache Architecture

```
┌─────────────────────────────────────────────────────┐
│                   Pingora Proxy                      │
│  ┌─────────────┐    ┌──────────────────────────┐   │
│  │   Request   │───▶│    Cache Layer           │   │
│  │   Handler   │    │  ┌────────────────────┐  │   │
│  │             │◀───│  │  Memory Cache      │  │   │
│  └─────────────┘    │  │  (LRU eviction)    │  │   │
│                     │  └────────────────────┘  │   │
│                     │  ┌────────────────────┐  │   │
│                     │  │  Disk Cache        │  │   │
│                     │  │  (persistent)      │  │   │
│                     │  └────────────────────┘  │   │
│                     └──────────────────────────┘   │
└─────────────────────────────────────────────────────┘
```

Pingora uses `pingora-cache` crate:

```toml
[dependencies]
pingora-cache = "0.7"
```

---

## 6.4 Basic Cache Implementation

```rust
use pingora_cache::{
    CacheKey, CacheMeta, CachePhase,
    RespCacheable, cache_control::CacheControl,
};

impl ProxyHttp for CachingProxy {
    type CTX = CacheCtx;

    /// Determine cache key for this request
    fn cache_key(&self, session: &Session, _ctx: &mut Self::CTX) -> Option<CacheKey> {
        let req = session.req_header();

        // Only cache GET requests
        if req.method != http::Method::GET {
            return None;
        }

        // Create cache key from method + host + path
        let host = req.headers.get("Host")
            .map(|h| h.to_str().unwrap_or(""))
            .unwrap_or("");

        Some(CacheKey::new(
            host.to_string(),
            req.uri.path().to_string(),
            "".to_string(),  // query params
        ))
    }

    /// Check if response is cacheable
    fn response_cache_filter(
        &self,
        _session: &Session,
        resp: &ResponseHeader,
        _ctx: &mut Self::CTX,
    ) -> RespCacheable {
        // Parse Cache-Control header
        if let Some(cc) = resp.headers.get("Cache-Control") {
            let cc_str = cc.to_str().unwrap_or("");

            // Don't cache if explicitly told not to
            if cc_str.contains("no-store") || cc_str.contains("private") {
                return RespCacheable::Uncacheable;
            }

            // Extract max-age
            if let Some(max_age) = parse_max_age(cc_str) {
                return RespCacheable::Cacheable(CacheMeta::new(
                    Duration::from_secs(max_age),
                    Duration::from_secs(max_age),  // stale-while-revalidate
                ));
            }
        }

        // Default: don't cache
        RespCacheable::Uncacheable
    }
}

fn parse_max_age(cc: &str) -> Option<u64> {
    for directive in cc.split(',') {
        let directive = directive.trim();
        if directive.starts_with("max-age=") {
            return directive[8..].parse().ok();
        }
        if directive.starts_with("s-maxage=") {
            return directive[9..].parse().ok();
        }
    }
    None
}
```

---

## 6.5 Cache Key Design

The cache key determines what's "the same" request:

### Simple Key (path only)
```rust
CacheKey::new("", path.to_string(), "")
```
Problem: Different hosts get same cached response!

### Better Key (host + path)
```rust
CacheKey::new(host.to_string(), path.to_string(), "")
```

### Full Key (include query)
```rust
CacheKey::new(host, path, query)
```
Problem: `?utm_source=...` creates separate cache entries

### Custom Key
```rust
fn cache_key(&self, session: &Session, _ctx: &mut Self::CTX) -> Option<CacheKey> {
    let req = session.req_header();
    let path = req.uri.path();

    // Don't cache API endpoints
    if path.starts_with("/api/") {
        return None;
    }

    // Include Accept-Language for localized content
    let lang = req.headers.get("Accept-Language")
        .map(|h| h.to_str().unwrap_or("en"))
        .unwrap_or("en");

    Some(CacheKey::new(
        host.to_string(),
        format!("{}:{}", path, lang),
        "".to_string(),
    ))
}
```

---

## 6.6 Cache Policies

### What to Cache

| Cache | Don't Cache |
|-------|-------------|
| Static assets (JS, CSS, images) | User-specific data |
| Public HTML pages | API responses with auth |
| Shared API responses | POST/PUT/DELETE requests |
| CDN-like content | Real-time data |

### For How Long

| Content Type | TTL |
|--------------|-----|
| Static assets | 1 year (with versioning) |
| HTML pages | 5-15 minutes |
| API responses | 1-60 seconds |
| User avatars | 1 hour |

---

## 6.7 Stale-While-Revalidate

Serve stale cache while fetching fresh content:

```
1. Cache expires
2. Request comes in
3. Immediately serve stale response (fast!)
4. Background: fetch fresh content from backend
5. Update cache with fresh content
6. Next request gets fresh content
```

```rust
RespCacheable::Cacheable(CacheMeta::new(
    Duration::from_secs(60),      // Fresh for 60s
    Duration::from_secs(300),     // Stale-while-revalidate for 5min
))
```

---

## 6.8 Cache Invalidation

"There are only two hard things in Computer Science: cache invalidation and naming things."

### Time-Based (TTL)
- Set appropriate `max-age`
- Simple but not instant

### Tag-Based
```rust
// On write
cache.purge_by_tag("product-123");

// On read
cache_meta.add_tag("product-123");
```

### Purge Endpoint
```rust
async fn request_filter(&self, session: &mut Session, _ctx: &mut Self::CTX) -> Result<bool> {
    if session.req_header().method == "PURGE" {
        let path = session.req_header().uri.path();
        self.cache.purge(path);
        // Return 200 OK
        return Ok(true);
    }
    Ok(false)
}
```

---

## Exercises

### Exercise 6.1: Basic Cache
Implement caching for all GET requests to `/static/*` with 1 hour TTL.

### Exercise 6.2: Cache Headers
Add `X-Cache: HIT` or `X-Cache: MISS` header to responses.

### Exercise 6.3: Conditional Caching
Only cache responses with status 200.

### Exercise 6.4: Cache Stats
Track and report cache hit rate every 100 requests.

---

## Key Takeaways

1. **Cache key** determines what's considered the same request
2. **Respect Cache-Control** headers from backend
3. Use **stale-while-revalidate** for better performance
4. **Don't cache** authenticated/personalized content
5. Plan for **cache invalidation** from the start

---

## Next Chapter

Your proxy is fast! Now let's make it observable.

Continue to [Chapter 07: Observability](../07-observability/README.md)
