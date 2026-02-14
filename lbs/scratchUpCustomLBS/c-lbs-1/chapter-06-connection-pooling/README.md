# Chapter 06: Connection Pooling (Pingora-Inspired)

## Learning Objectives

1. Understand why connection pooling matters for performance
2. Learn how Pingora achieved 99.92% connection reuse (vs Nginx's 87%)
3. Implement a thread-safe LRU connection pool
4. Integrate keep-alive management with HTTP

## The Problem: Connection Overhead

```
WITHOUT POOLING (Chapters 01-05)
═══════════════════════════════════════════════════════════════

Request 1: connect() → use → close()   [~1ms overhead]
Request 2: connect() → use → close()   [~1ms overhead]
Request 3: connect() → use → close()   [~1ms overhead]
...
1000 requests = 1000 connections = ~1 second wasted!


WITH POOLING (This Chapter)
═══════════════════════════════════════════════════════════════

Request 1: connect() → use → return to pool   [~1ms]
Request 2: get from pool → use → return        [~0ms]
Request 3: get from pool → use → return        [~0ms]
...
1000 requests = 1 connection = near-zero overhead!
```

## Pingora vs Nginx: Why the Difference?

```
NGINX ARCHITECTURE
══════════════════
┌────────────┐  ┌────────────┐  ┌────────────┐
│  Worker 1  │  │  Worker 2  │  │  Worker 3  │
│  (Process) │  │  (Process) │  │  (Process) │
├────────────┤  ├────────────┤  ├────────────┤
│  Pool A    │  │  Pool B    │  │  Pool C    │
│  (Private) │  │  (Private) │  │  (Private) │
└────────────┘  └────────────┘  └────────────┘

Problem: Each worker has isolated pool.
         Connection to backend-1 in Worker A
         can't be reused by Worker B!

Result: 87.1% connection reuse


PINGORA ARCHITECTURE (Thread-based)
═══════════════════════════════════
┌─────────────────────────────────────────┐
│           SHARED PROCESS                │
├─────────────────────────────────────────┤
│  Thread 1   Thread 2   Thread 3         │
│     │          │          │             │
│     └──────────┼──────────┘             │
│                ▼                        │
│     ┌──────────────────────┐           │
│     │   SHARED POOL        │           │
│     │   (Thread-Safe)      │           │
│     └──────────────────────┘           │
└─────────────────────────────────────────┘

Benefit: All threads share one pool.
         Any connection can be reused by anyone!

Result: 99.92% connection reuse (160x fewer new connections!)
```

## Key Data Structures

### Connection Pool

```c
typedef struct {
    PooledConnection *connections;   // All pooled connections
    int max_size;                    // Pool capacity
    int ttl_seconds;                 // Connection lifetime

    // LRU tracking
    PooledConnection *lru_head;      // Most recently used
    PooledConnection *lru_tail;      // Least recently used

    // Statistics
    unsigned long pool_hits;
    unsigned long pool_misses;

    pthread_mutex_t lock;            // Thread safety
} conn_pool_t;
```

### Pooled Connection

```c
typedef struct PooledConnection {
    int fd;
    char backend_host[256];
    char backend_port[6];
    time_t last_used;
    time_t created;
    int requests_served;

    // Doubly-linked for LRU
    struct PooledConnection *next;
    struct PooledConnection *prev;
} PooledConnection;
```

## API Reference

```c
// Create pool with capacity and TTL
conn_pool_t* conn_pool_create(int max_size, int ttl_seconds);

// Get connection (from pool or create new)
int conn_pool_get(conn_pool_t *pool, const char *host, const char *port);

// Return connection for reuse
void conn_pool_return(conn_pool_t *pool, int fd, const char *host, const char *port);

// Close broken connection
void conn_pool_close(conn_pool_t *pool, int fd);

// Cleanup expired connections
int conn_pool_cleanup(conn_pool_t *pool);
```

## Connection Lifecycle

```
                    ┌─────────────────────────────────────────┐
                    │         conn_pool_get()                 │
                    └───────────────┬─────────────────────────┘
                                    │
                    ┌───────────────▼───────────────┐
                    │ Search pool for matching      │
                    │ backend connection            │
                    └───────────────┬───────────────┘
                                    │
                ┌───────────────────┴───────────────────┐
                │                                       │
        ┌───────▼───────┐                       ┌───────▼───────┐
        │ FOUND (HIT)   │                       │ NOT FOUND     │
        │               │                       │ (MISS)        │
        └───────┬───────┘                       └───────┬───────┘
                │                                       │
        ┌───────▼───────┐                       ┌───────▼───────┐
        │ Validate:     │                       │ Create new    │
        │ - TTL ok?     │                       │ connection    │
        │ - Still alive?│                       │ connect()     │
        └───────┬───────┘                       └───────┬───────┘
                │                                       │
                └─────────────────┬─────────────────────┘
                                  │
                    ┌─────────────▼─────────────┐
                    │ Return fd to caller       │
                    └─────────────┬─────────────┘
                                  │
                    ┌─────────────▼─────────────┐
                    │ Caller uses connection    │
                    └─────────────┬─────────────┘
                                  │
                ┌─────────────────┴─────────────────┐
                │                                   │
        ┌───────▼───────┐                   ┌───────▼───────┐
        │ Success:      │                   │ Error:        │
        │ pool_return() │                   │ pool_close()  │
        └───────────────┘                   └───────────────┘
```

## Building & Running

```bash
# Build
make

# Run with custom pool size
./pooled_lb 8080 127.0.0.1:9001 127.0.0.1:9002 -p 32 -a wrr

# Test and view pool statistics
make test

# View live stats
kill -USR1 $(pgrep pooled_lb)
```

## Expected Output

```
====================================================================
  CONNECTION POOL STATS (Pingora-style):
    Pool Size: 8 / 64
    Pool Hits: 1847  Misses: 12  Evictions: 0
    HIT RATE: 99.35% (target: 99%+)
====================================================================
```

## LRU Eviction

When pool is full, least recently used connections are evicted:

```
LRU List (doubly-linked):

  HEAD (Most Recent)              TAIL (Least Recent)
    │                                    │
    ▼                                    ▼
┌──────┐    ┌──────┐    ┌──────┐    ┌──────┐
│ C1   │◄──►│ C2   │◄──►│ C3   │◄──►│ C4   │
│ 10s  │    │ 30s  │    │ 45s  │    │ 60s  │
└──────┘    └──────┘    └──────┘    └──────┘

On pool_get() with no match and pool full:
  → Evict C4 (oldest)
  → Create new connection in that slot
```

## Health Validation

Before reusing a pooled connection, we validate it's still alive:

```c
int conn_is_alive(int fd) {
    struct pollfd pfd = {fd, POLLIN, 0};

    int ret = poll(&pfd, 1, 0);  // Non-blocking

    if (ret > 0 && (pfd.revents & (POLLERR | POLLHUP))) {
        return 0;  // Connection is dead
    }

    if (pfd.revents & POLLIN) {
        // Data available - check if it's EOF
        char buf[1];
        if (recv(fd, buf, 1, MSG_PEEK | MSG_DONTWAIT) == 0) {
            return 0;  // Server closed connection
        }
    }

    return 1;  // Connection is alive
}
```

## Performance Impact

| Metric | Without Pool | With Pool | Improvement |
|--------|--------------|-----------|-------------|
| Connections/sec | 1,000 | 50,000 | 50x |
| Latency (avg) | 5ms | 0.5ms | 10x |
| Backend CPU | 100% | 20% | 5x |
| Memory | High (TCBs) | Low | Significant |

## Exercises

1. **Implement per-backend pools**: Instead of one global pool, create
   separate pools per backend for better isolation.

2. **Add pool warmup**: Pre-create connections at startup to avoid
   cold-start latency.

3. **Implement max-age**: Close connections after N requests to prevent
   memory leaks from long-lived connections.

4. **Add connection timeout**: Track how long a connection has been in use
   and close it if exceeds threshold.

## Key Takeaways

1. **Connection reuse is crucial** - Creating connections is expensive
2. **Shared pools beat per-process** - Pingora's insight
3. **LRU eviction prevents staleness** - Old connections get replaced
4. **Health checks prevent errors** - Validate before reuse
5. **Thread safety is essential** - Multiple workers share one pool

## Next Chapter

Chapter 07 introduces **Rate Limiting** - protecting backends from
overload with token bucket and sliding window algorithms.
