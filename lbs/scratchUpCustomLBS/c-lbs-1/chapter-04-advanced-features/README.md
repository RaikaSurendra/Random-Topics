# Chapter 04: Advanced Features

## Learning Objectives

1. Implement non-blocking I/O with epoll
2. Add weighted load balancing
3. Implement least-connections algorithm
4. Add HTTP-aware load balancing
5. Connection pooling

## Advanced Load Balancing Algorithms

### Weighted Round Robin

```c
typedef struct {
    char host[256];
    int port;
    int weight;           // Server capacity (1-10)
    int current_weight;   // Current position in weighted cycle
} WeightedBackend;

Backend* weighted_round_robin(LoadBalancer *lb) {
    int total_weight = 0;
    Backend *best = NULL;

    for (int i = 0; i < lb->num_backends; i++) {
        Backend *b = &lb->backends[i];
        if (!b->is_healthy) continue;

        b->current_weight += b->weight;
        total_weight += b->weight;

        if (best == NULL || b->current_weight > best->current_weight) {
            best = b;
        }
    }

    if (best) {
        best->current_weight -= total_weight;
    }

    return best;
}
```

### Least Connections

```c
Backend* least_connections(LoadBalancer *lb) {
    Backend *best = NULL;
    int min_connections = INT_MAX;

    for (int i = 0; i < lb->num_backends; i++) {
        Backend *b = &lb->backends[i];
        if (!b->is_healthy) continue;

        // Weight-adjusted connections
        int adjusted = b->active_connections * 100 / b->weight;

        if (adjusted < min_connections) {
            min_connections = adjusted;
            best = b;
        }
    }

    return best;
}
```

### IP Hash (Sticky Sessions)

```c
Backend* ip_hash(LoadBalancer *lb, const char *client_ip) {
    unsigned int hash = 0;

    // Simple hash function
    for (const char *p = client_ip; *p; p++) {
        hash = hash * 31 + *p;
    }

    // Find healthy backend using consistent hashing
    int start = hash % lb->num_backends;
    int index = start;

    do {
        if (lb->backends[index].is_healthy) {
            return &lb->backends[index];
        }
        index = (index + 1) % lb->num_backends;
    } while (index != start);

    return NULL;
}
```

## Non-Blocking I/O with epoll

### Why epoll?

```
BLOCKING (Chapter 01-03)               NON-BLOCKING (This Chapter)
════════════════════════               ═══════════════════════════

Thread 1:                              Single Thread (Event Loop):
  wait for client 1...                   - poll all connections
  handle client 1...                     - handle ready ones
  wait for client 2...                   - continue loop
  handle client 2...

Scalability: Limited                   Scalability: Thousands of connections
Memory: Thread per connection          Memory: Minimal
```

### epoll API

```c
// Create epoll instance
int epoll_fd = epoll_create1(0);

// Add socket to watch list
struct epoll_event ev;
ev.events = EPOLLIN | EPOLLET;  // Edge-triggered read
ev.data.fd = socket_fd;
epoll_ctl(epoll_fd, EPOLL_CTL_ADD, socket_fd, &ev);

// Wait for events
struct epoll_event events[MAX_EVENTS];
int n = epoll_wait(epoll_fd, events, MAX_EVENTS, timeout_ms);

for (int i = 0; i < n; i++) {
    if (events[i].data.fd == server_fd) {
        // New connection
        accept(...);
    } else {
        // Data ready on existing connection
        handle_data(events[i].data.fd);
    }
}
```

## Connection Pooling

### Why Pool Connections?

```
WITHOUT POOLING                        WITH POOLING
═══════════════                        ═══════════════

Request 1:                             Request 1:
  connect() → use → close()              connect() → use → return to pool

Request 2:                             Request 2:
  connect() → use → close()              get from pool → use → return

Request 3:                             Request 3:
  connect() → use → close()              get from pool → use → return

Time: 3× connection overhead           Time: 1× connection overhead
```

### Connection Pool Implementation

```c
typedef struct {
    int fd;
    Backend *backend;
    time_t last_used;
    int in_use;
} PooledConnection;

typedef struct {
    PooledConnection connections[POOL_SIZE];
    int size;
    pthread_mutex_t lock;
} ConnectionPool;

int get_pooled_connection(ConnectionPool *pool, Backend *backend) {
    pthread_mutex_lock(&pool->lock);

    for (int i = 0; i < pool->size; i++) {
        PooledConnection *pc = &pool->connections[i];
        if (!pc->in_use && pc->backend == backend) {
            // Check if still valid
            if (is_connection_alive(pc->fd)) {
                pc->in_use = 1;
                pthread_mutex_unlock(&pool->lock);
                return pc->fd;
            } else {
                close(pc->fd);
                pc->fd = -1;
            }
        }
    }

    pthread_mutex_unlock(&pool->lock);
    return -1;  // No pooled connection, create new
}
```

## HTTP-Aware Features

### Request Routing by Path

```c
typedef struct {
    char path_prefix[256];
    int backend_group;  // Which group of backends handles this path
} Route;

Route routes[] = {
    {"/api/",    0},   // API servers
    {"/static/", 1},   // Static file servers
    {"/",        2},   // Default servers
};

int select_backend_group(const char *request) {
    // Parse path from request
    char path[256];
    sscanf(request, "%*s %255s", path);

    for (int i = 0; i < num_routes; i++) {
        if (strncmp(path, routes[i].path_prefix,
                    strlen(routes[i].path_prefix)) == 0) {
            return routes[i].backend_group;
        }
    }

    return default_group;
}
```

### Header Injection

```c
void inject_headers(char *request, size_t max_size,
                   const char *client_ip, const char *original_host) {
    // Find end of first line
    char *header_start = strstr(request, "\r\n");
    if (!header_start) return;
    header_start += 2;

    char new_headers[512];
    snprintf(new_headers, sizeof(new_headers),
             "X-Forwarded-For: %s\r\n"
             "X-Real-IP: %s\r\n"
             "X-Forwarded-Host: %s\r\n",
             client_ip, client_ip, original_host);

    // Insert new headers
    size_t new_len = strlen(new_headers);
    size_t remaining = strlen(header_start);

    if (header_start - request + new_len + remaining < max_size) {
        memmove(header_start + new_len, header_start, remaining + 1);
        memcpy(header_start, new_headers, new_len);
    }
}
```

## Building & Running

```bash
# Compile
make

# Start with weighted backends
./advanced_lb 8080 \
    --backend 127.0.0.1:9001:3 \
    --backend 127.0.0.1:9002:2 \
    --backend 127.0.0.1:9003:1 \
    --algorithm weighted

# Start with least-connections
./advanced_lb 8080 \
    --backend 127.0.0.1:9001 \
    --backend 127.0.0.1:9002 \
    --algorithm least-conn
```

## Files in This Chapter

| File | Description |
|------|-------------|
| `advanced_lb.c` | Full-featured load balancer |
| `Makefile` | Build configuration |

## Performance Comparison

| Feature | Chapter 03 | Chapter 04 |
|---------|------------|------------|
| I/O Model | Blocking | epoll (non-blocking) |
| Connections | Sequential | Concurrent |
| Memory | ~1MB/conn (thread) | ~1KB/conn (event) |
| Max Clients | ~1000 | ~100,000 |
| Algorithms | Round-robin | Multiple |
| Health Check | On-failure | Background thread |

## Production Considerations

### What's Still Missing

1. **SSL/TLS Termination** - Use OpenSSL
2. **HTTP/2 Support** - Complex framing
3. **Rate Limiting** - Token bucket algorithm
4. **Logging/Metrics** - Prometheus export
5. **Configuration Reload** - Hot config changes
6. **Clustering** - Multiple LB instances

### Recommended Next Steps

1. Study HAProxy source code
2. Learn about io_uring (newer than epoll)
3. Implement WebSocket support
4. Add Lua scripting for custom routing

## Summary: Building Blocks of a Load Balancer

```
┌─────────────────────────────────────────────────────────────────┐
│                     PRODUCTION LOAD BALANCER                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐ │
│  │   Socket    │  │   Event     │  │   Connection            │ │
│  │   Layer     │  │   Loop      │  │   Management            │ │
│  │  (Ch. 01)   │  │  (epoll)    │  │   (pooling)             │ │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘ │
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐ │
│  │   Proxy     │  │   Load      │  │   Health                │ │
│  │   Logic     │  │   Balancing │  │   Checking              │ │
│  │  (Ch. 02)   │  │  (Ch. 03)   │  │   (background)          │ │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘ │
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐ │
│  │   HTTP      │  │   Config    │  │   Metrics &             │ │
│  │   Parsing   │  │   System    │  │   Logging               │ │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘ │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```
