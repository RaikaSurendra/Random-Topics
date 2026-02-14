# Chapter 03: Load Balancer

## Learning Objectives

1. Extend reverse proxy to multiple backends
2. Implement round-robin load balancing
3. Add basic health checking
4. Understand load balancing algorithms

## Evolution: Reverse Proxy → Load Balancer

```
REVERSE PROXY (Chapter 02)              LOAD BALANCER (Chapter 03)
═══════════════════════════             ═══════════════════════════

┌────────┐      ┌───────┐               ┌────────┐      ┌───────────┐
│ Client │─────▶│ Proxy │──────────────▶│ Client │─────▶│    LB     │
└────────┘      └───────┘               └────────┘      └─────┬─────┘
                    │                                         │
                    ▼                            ┌────────────┼────────────┐
               ┌─────────┐                       ▼            ▼            ▼
               │ Backend │                  ┌─────────┐  ┌─────────┐  ┌─────────┐
               └─────────┘                  │Backend 1│  │Backend 2│  │Backend 3│
                                            └─────────┘  └─────────┘  └─────────┘

     ONE backend                              MULTIPLE backends
                                              + Selection algorithm
```

## Load Balancing Algorithms

### 1. Round Robin (This Chapter)

Distribute requests sequentially across backends.

```
Request 1 → Backend 1
Request 2 → Backend 2
Request 3 → Backend 3
Request 4 → Backend 1  (cycle repeats)
...
```

**Pros:** Simple, even distribution
**Cons:** Ignores server capacity and current load

### 2. Weighted Round Robin

Some servers get more requests based on weight.

```c
Backend 1: weight=3  (gets 3x traffic)
Backend 2: weight=1
Backend 3: weight=1

Request sequence: 1, 1, 1, 2, 3, 1, 1, 1, 2, 3, ...
```

### 3. Least Connections

Route to server with fewest active connections.

```c
Backend 1: 5 connections  ← skip
Backend 2: 2 connections  ← CHOOSE THIS
Backend 3: 4 connections  ← skip
```

**Pros:** Better for varying request durations
**Cons:** Requires connection tracking

### 4. IP Hash

Same client always goes to same backend (session persistence).

```c
client_ip = "192.168.1.100"
backend_index = hash(client_ip) % num_backends
```

**Pros:** Session stickiness without cookies
**Cons:** Uneven distribution if few clients

### 5. Random

Just pick randomly.

```c
backend_index = rand() % num_backends
```

**Pros:** Simple, stateless
**Cons:** Can be uneven short-term

## Data Structures

```c
// Backend server definition
typedef struct {
    char host[256];
    int port;
    int is_healthy;      // Health status
    int active_connections;
    int total_requests;
    time_t last_check;
} Backend;

// Load balancer state
typedef struct {
    Backend backends[MAX_BACKENDS];
    int num_backends;
    int current_index;   // For round-robin
    pthread_mutex_t lock;
} LoadBalancer;
```

## Building & Running

```bash
# Compile
make

# Terminal 1-3: Start 3 backend servers (use echo_server from Chapter 1)
cd ../chapter-01-fundamentals
./echo_server 9001 &
./echo_server 9002 &
./echo_server 9003 &

# Terminal 4: Start load balancer
./load_balancer 8080

# Terminal 5: Test - watch requests distribute
for i in {1..6}; do
    echo "Request $i" | nc localhost 8080
    sleep 0.5
done
```

## Configuration

The load balancer reads backends from command line or config:

```bash
# Command line
./load_balancer 8080 127.0.0.1:9001 127.0.0.1:9002 127.0.0.1:9003

# Or edit config in code (for simplicity in this chapter)
```

## Key Code Changes from Chapter 02

### 1. Backend Pool Instead of Single Backend

```c
// Before (Chapter 02)
char backend_host[256];
int backend_port;

// After (Chapter 03)
Backend backends[MAX_BACKENDS];
int num_backends;
```

### 2. Backend Selection Function

```c
Backend* select_backend(LoadBalancer *lb) {
    // Round-robin selection
    int start = lb->current_index;

    do {
        lb->current_index = (lb->current_index + 1) % lb->num_backends;
        Backend *b = &lb->backends[lb->current_index];

        if (b->is_healthy) {
            return b;
        }
    } while (lb->current_index != start);

    return NULL;  // All backends unhealthy
}
```

### 3. Connection Tracking

```c
void handle_client(...) {
    Backend *backend = select_backend(lb);
    backend->active_connections++;

    // ... relay data ...

    backend->active_connections--;
    backend->total_requests++;
}
```

## Health Checking (Basic)

```c
int check_backend_health(Backend *backend) {
    int sock = socket(AF_INET, SOCK_STREAM, 0);

    // Set short timeout
    struct timeval timeout = {1, 0};  // 1 second
    setsockopt(sock, SOL_SOCKET, SO_SNDTIMEO, &timeout, sizeof(timeout));

    // Try to connect
    int result = connect(sock, ...);
    close(sock);

    return (result == 0);
}
```

## Statistics Tracking

```c
typedef struct {
    unsigned long total_requests;
    unsigned long failed_requests;
    unsigned long bytes_transferred;
    time_t start_time;
} Stats;

void print_stats(LoadBalancer *lb) {
    printf("\n=== Load Balancer Statistics ===\n");
    for (int i = 0; i < lb->num_backends; i++) {
        Backend *b = &lb->backends[i];
        printf("Backend %s:%d - Requests: %d, Active: %d, Healthy: %s\n",
               b->host, b->port, b->total_requests,
               b->active_connections, b->is_healthy ? "Yes" : "No");
    }
}
```

## Files in This Chapter

| File | Description |
|------|-------------|
| `load_balancer.c` | Round-robin load balancer |
| `Makefile` | Build configuration |

## Testing Scenarios

### 1. Even Distribution Test

```bash
# Start 3 backends
for port in 9001 9002 9003; do
    ../chapter-01-fundamentals/echo_server $port &
done

# Start LB
./load_balancer 8080

# Send 9 requests - should be 3 per backend
for i in {1..9}; do
    echo "Request $i" | nc localhost 8080
done
```

### 2. Failover Test

```bash
# Start LB and backends
./load_balancer 8080 &

# Kill one backend
kill %2  # Kills second echo_server

# Requests should skip dead backend
for i in {1..6}; do
    echo "Request $i" | nc localhost 8080
done
```

## Exercises

1. **Weighted Round Robin**: Add weights to backends
2. **Least Connections**: Track connections, route to least busy
3. **Health Check Thread**: Background health checking
4. **Sticky Sessions**: IP hash for session persistence
5. **Statistics Endpoint**: HTTP endpoint showing backend stats

## Next Chapter

Chapter 04 adds advanced features:
- Non-blocking I/O with epoll
- Connection pooling
- More algorithms
- HTTP parsing
