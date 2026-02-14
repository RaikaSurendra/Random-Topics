# Chapter 07: Rate Limiting

## Learning Objectives

1. Understand why rate limiting is essential for production systems
2. Implement Token Bucket algorithm (smooth limiting with burst)
3. Implement Sliding Window algorithm (accurate per-second limiting)
4. Design thread-safe rate limiter with hash table

## Why Rate Limiting?

```
WITHOUT RATE LIMITING
═══════════════════════════════════════════════════════════════

Malicious client: 10,000 requests/second
    ↓
Backend servers: OVERWHELMED
    ↓
All users: 503 Service Unavailable


WITH RATE LIMITING
═══════════════════════════════════════════════════════════════

Malicious client: 10,000 requests/second
    ↓
Rate Limiter: Allow 100/s, deny 9,900
    ↓
Backend servers: HEALTHY
    ↓
Legitimate users: 200 OK
```

## Algorithms

### Token Bucket

```
┌─────────────────────────────────────────────────────────────┐
│                     TOKEN BUCKET                            │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Bucket Capacity: 10 tokens (burst size)                   │
│  Refill Rate: 1 token/second                               │
│                                                             │
│  Initial State:      After 3 requests:    After 10s idle:  │
│  ┌──────────┐        ┌──────────┐         ┌──────────┐     │
│  │██████████│ 10     │███████   │ 7       │██████████│ 10  │
│  └──────────┘        └──────────┘         └──────────┘     │
│                                                             │
│  Pros: Allows burst, smooth average rate                   │
│  Cons: Complex state tracking                               │
└─────────────────────────────────────────────────────────────┘
```

### Sliding Window

```
┌─────────────────────────────────────────────────────────────┐
│                    SLIDING WINDOW                           │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Window Size: 1 second                                      │
│  Limit: 10 requests/window                                  │
│                                                             │
│  Time:  |------ Window (1s) ------|                        │
│         t-1s                     t                          │
│  Reqs:  [###########]            [########]                │
│         Count: 11                Count: 8                   │
│                                                             │
│  Weighted Count = 11 * 0.3 + 8 = 11.3 (deny next)         │
│                                                             │
│  Pros: Accurate, no bursts at window boundary              │
│  Cons: Slightly more computation                            │
└─────────────────────────────────────────────────────────────┘
```

## API

```c
// Create rate limiter
rate_limiter_t* rate_limiter_create(
    RateLimitAlgorithm algorithm,  // TOKEN_BUCKET or SLIDING_WINDOW
    double rate,                    // Requests per second
    double burst                    // Burst size or window size
);

// Check if request allowed
int rate_limiter_allow(rate_limiter_t *limiter, const char *client_ip);

// Returns: 1 = allowed, 0 = rate limited (return 429)
```

## Integration with Load Balancer

```c
void handle_request(Connection *conn) {
    // Check rate limit BEFORE processing
    if (!rate_limiter_allow(lb->limiter, conn->client_ip)) {
        send_response(conn, 429, "Too Many Requests");
        return;
    }

    // Process request normally
    forward_to_backend(conn);
}
```

## Response Headers

When rate limited, include helpful headers:

```
HTTP/1.1 429 Too Many Requests
Retry-After: 1
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1612345678
```

## Exercises

1. Add `X-RateLimit-*` response headers
2. Implement per-endpoint rate limits (e.g., `/api/login` = 5/min)
3. Add IP whitelist for trusted clients
4. Implement distributed rate limiting with Redis

## Key Takeaways

1. **Token bucket** is best for APIs (allows burst)
2. **Sliding window** is best for strict limits (no bursts)
3. Always return **429** with **Retry-After** header
4. Consider **distributed limiting** for multi-instance deployments
