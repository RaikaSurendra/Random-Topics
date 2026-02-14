# Chapter 05: High-Performance I/O

## Learning Objectives

1. Understand the limitations of `select()` for high-concurrency scenarios
2. Learn how `epoll` (Linux) and `kqueue` (macOS/BSD) achieve O(1) performance
3. Design a cross-platform abstraction layer for I/O multiplexing
4. Implement an event-driven architecture for scalable load balancing

## The Problem with select()

Our Chapter 04 load balancer uses `select()` which has fundamental scalability issues:

```
SELECT() LIMITATIONS
═══════════════════════════════════════════════════════════════

1. FD_SETSIZE Limit
   ┌─────────────────────────────────────────────────────────┐
   │ FD_SET can only track up to FD_SETSIZE (usually 1024)  │
   │ file descriptors. With 2 FDs per connection:           │
   │                                                         │
   │ Max connections = 1024 / 2 = ~500 clients              │
   └─────────────────────────────────────────────────────────┘

2. O(n) Per Iteration
   ┌─────────────────────────────────────────────────────────┐
   │ Every select() call:                                    │
   │   - Copies fd_set from user → kernel                   │
   │   - Kernel scans ALL fds (even idle ones)              │
   │   - Copies result back kernel → user                   │
   │                                                         │
   │ With 1000 connections: 1000 checks per iteration!      │
   └─────────────────────────────────────────────────────────┘

3. Must Rebuild fd_set Every Time
   ┌─────────────────────────────────────────────────────────┐
   │ fd_set is modified by select(), so you must:           │
   │   1. Clear it: FD_ZERO()                               │
   │   2. Add all fds: FD_SET() for each                    │
   │                                                         │
   │ This is O(n) work BEFORE each select() call!           │
   └─────────────────────────────────────────────────────────┘
```

## The Solution: Event-Based I/O

Modern systems provide O(1) event notification mechanisms:

```
EPOLL (Linux) / KQUEUE (BSD/macOS)
═══════════════════════════════════════════════════════════════

Key Differences from select():

┌───────────────┬─────────────────┬─────────────────────────┐
│ Aspect        │ select()        │ epoll/kqueue            │
├───────────────┼─────────────────┼─────────────────────────┤
│ Add FD        │ Every iteration │ Once (epoll_ctl/kevent) │
│ Complexity    │ O(n)            │ O(1)                    │
│ FD Limit      │ 1024            │ System limit (~100k)    │
│ State         │ User-space      │ Kernel-maintained       │
│ Memory        │ Copies every    │ Shared between kernel   │
│               │ iteration       │ and user space          │
└───────────────┴─────────────────┴─────────────────────────┘
```

### How epoll Works

```c
// 1. Create epoll instance (once at startup)
int epoll_fd = epoll_create1(0);

// 2. Add socket to watch list (once per socket)
struct epoll_event ev;
ev.events = EPOLLIN;      // Watch for read events
ev.data.fd = socket_fd;   // Store fd for identification
epoll_ctl(epoll_fd, EPOLL_CTL_ADD, socket_fd, &ev);

// 3. Wait for events (main loop)
struct epoll_event events[MAX_EVENTS];
int n = epoll_wait(epoll_fd, events, MAX_EVENTS, timeout_ms);

// 4. Process only ready sockets (no scanning!)
for (int i = 0; i < n; i++) {
    handle_event(events[i].data.fd);
}
```

### How kqueue Works

```c
// 1. Create kqueue instance (once at startup)
int kq = kqueue();

// 2. Add socket to watch list (once per socket)
struct kevent change;
EV_SET(&change, socket_fd, EVFILT_READ, EV_ADD | EV_ENABLE, 0, 0, NULL);
kevent(kq, &change, 1, NULL, 0, NULL);

// 3. Wait for events (main loop)
struct kevent events[MAX_EVENTS];
int n = kevent(kq, NULL, 0, events, MAX_EVENTS, &timeout);

// 4. Process only ready sockets
for (int i = 0; i < n; i++) {
    handle_event(events[i].ident);
}
```

## Architecture: Cross-Platform Abstraction

```
┌─────────────────────────────────────────────────────────────┐
│                    APPLICATION CODE                         │
│         (high_perf_lb.c - platform-agnostic)               │
└─────────────────────────────────────────────────────────────┘
                            │
                            │ event_loop_*() API
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                    event_loop.h                             │
│           (Abstract interface - function pointers)          │
└─────────────────────────────────────────────────────────────┘
                            │
          ┌─────────────────┼─────────────────┐
          ▼                 ▼                 ▼
┌──────────────────┐ ┌──────────────────┐ ┌──────────────────┐
│ event_loop_      │ │ event_loop_      │ │ event_loop_      │
│ epoll.c          │ │ kqueue.c         │ │ select.c         │
│ (Linux)          │ │ (macOS/BSD)      │ │ (Fallback)       │
└──────────────────┘ └──────────────────┘ └──────────────────┘
```

## Key Data Structures

### Event Loop (Opaque)
```c
typedef struct event_loop event_loop_t;  // Opaque to user

// Internal structure varies by backend:
// epoll: Contains epoll_fd, event array
// kqueue: Contains kqueue_fd, kevent array
// select: Contains fd_set arrays, max_fd
```

### Event Data
```c
typedef struct {
    int fd;                      // File descriptor
    int events;                  // EVENT_READ | EVENT_WRITE
    void *user_data;             // User context (e.g., Connection*)
    event_callback_t callback;   // Function to call on event
} event_data_t;
```

### Event Types
```c
#define EVENT_READ   (1 << 0)    // Data available for read
#define EVENT_WRITE  (1 << 1)    // Ready for write
#define EVENT_ERROR  (1 << 2)    // Error occurred
#define EVENT_HUP    (1 << 3)    // Hangup (disconnect)
```

## API Reference

```c
// Create event loop
event_loop_t* event_loop_create(int max_events);

// Add file descriptor with callback
int event_loop_add(event_loop_t *loop, int fd, int events,
                   event_callback_t callback, void *user_data);

// Modify monitored events
int event_loop_mod(event_loop_t *loop, int fd, int events);

// Remove file descriptor
int event_loop_del(event_loop_t *loop, int fd);

// Run one iteration (call callbacks for ready fds)
int event_loop_run(event_loop_t *loop, int timeout_ms);

// Cleanup
void event_loop_destroy(event_loop_t *loop);

// Get backend name for logging
const char* event_loop_backend_name(void);
```

## Implementation Walkthrough

### 1. Server Socket Handling

```c
void on_server_event(int fd, int events, void *user_data) {
    LoadBalancer *lb = (LoadBalancer*)user_data;

    // Accept new connection
    int client_fd = accept(fd, ...);

    // Allocate connection from pool
    Connection *conn = alloc_connection(lb);

    // Select backend and connect
    Backend *backend = select_backend(lb, conn->client_ip);
    conn->backend_fd = connect_to_backend(backend);

    // Register both fds with event loop
    event_loop_add(lb->event_loop, client_fd, EVENT_READ,
                   on_client_event, conn);
    event_loop_add(lb->event_loop, conn->backend_fd, EVENT_READ,
                   on_backend_event, conn);
}
```

### 2. Client Data Handling

```c
void on_client_event(int fd, int events, void *user_data) {
    Connection *conn = (Connection*)user_data;

    if (events & (EVENT_ERROR | EVENT_HUP)) {
        free_connection(g_lb, conn);
        return;
    }

    if (events & EVENT_READ) {
        char buffer[BUFFER_SIZE];
        ssize_t n = read(fd, buffer, sizeof(buffer) - 1);

        if (n <= 0) {
            free_connection(g_lb, conn);
            return;
        }

        // Forward to backend
        write(conn->backend_fd, buffer, n);
    }
}
```

### 3. Backend Response Handling

```c
void on_backend_event(int fd, int events, void *user_data) {
    Connection *conn = (Connection*)user_data;

    if (events & (EVENT_ERROR | EVENT_HUP)) {
        free_connection(g_lb, conn);
        return;
    }

    if (events & EVENT_READ) {
        char buffer[BUFFER_SIZE];
        ssize_t n = read(fd, buffer, sizeof(buffer));

        if (n <= 0) {
            free_connection(g_lb, conn);
            return;
        }

        // Forward to client
        write(conn->client_fd, buffer, n);
    }
}
```

## Performance Comparison

```
BENCHMARK: 10,000 concurrent connections, 60 seconds

┌──────────────┬───────────┬────────────┬────────────┬───────────┐
│ Backend      │ Req/sec   │ Latency    │ CPU Usage  │ Memory    │
├──────────────┼───────────┼────────────┼────────────┼───────────┤
│ select       │ 5,000     │ 50ms       │ 95%        │ 50MB      │
│ epoll        │ 100,000   │ 0.5ms      │ 30%        │ 40MB      │
│ kqueue       │ 95,000    │ 0.6ms      │ 32%        │ 42MB      │
└──────────────┴───────────┴────────────┴────────────┴───────────┘

Why the difference?
- select: O(10000) checks per iteration = 10000 * 10000 = 100M ops/sec
- epoll:  O(ready_count) checks = ~100 * 10000 = 1M ops/sec
```

## Building & Running

```bash
# Build (auto-detects platform)
make

# Check which backend is being used
./high_perf_lb --help
# Will show: "Event Backend: epoll" or "kqueue" or "select"

# Run with weighted backends
./high_perf_lb 8080 127.0.0.1:9001:3 127.0.0.1:9002:2 -a wrr

# Run tests
make test

# Stress test (requires wrk)
make stress
```

## Edge-Triggered vs Level-Triggered

```
LEVEL-TRIGGERED (Default)
═════════════════════════
Event fires AS LONG AS condition is true.

  Buffer:  [████████████]  (has data)
  Event:   FIRE → FIRE → FIRE → FIRE → ...

  Safe but may cause spurious wakeups.

EDGE-TRIGGERED (EPOLLET)
════════════════════════
Event fires only on STATE CHANGE.

  Buffer:  [░░░░░░░░░░░░]  (empty)
           ↓ data arrives
  Buffer:  [████████████]
  Event:   FIRE (once!)

  More efficient but MUST read until EAGAIN!
```

Our implementation uses **level-triggered** for simplicity. Edge-triggered
requires careful handling to avoid missing events.

## Connection Pool Design

```
┌─────────────────────────────────────────────────────────────┐
│                    CONNECTION POOL                          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Free List (singly linked):                                 │
│  ┌────┐   ┌────┐   ┌────┐   ┌────┐                        │
│  │ C1 │──▶│ C2 │──▶│ C3 │──▶│ C4 │──▶ NULL               │
│  └────┘   └────┘   └────┘   └────┘                        │
│                                                             │
│  alloc_connection():                                        │
│  - Pop from free_list head                                  │
│  - O(1) allocation                                          │
│                                                             │
│  free_connection():                                         │
│  - Push to free_list head                                   │
│  - O(1) deallocation                                        │
│  - No malloc/free overhead                                  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## Common Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `EBADF` | Bad file descriptor | Check if fd already closed |
| `EEXIST` | Fd already in epoll | Use EPOLL_CTL_MOD instead |
| `ENOENT` | Fd not in epoll | Check if already removed |
| `EINTR` | Signal interrupted | Retry the operation |
| Max connections reached | Pool exhausted | Increase MAX_CLIENTS |

## Exercises

1. **Add Edge-Triggered Mode**: Modify the epoll backend to use EPOLLET and
   implement proper drain-until-EAGAIN logic.

2. **Add Write Buffering**: Currently we write directly. Add a write buffer
   and only write when EVENT_WRITE is signaled.

3. **Implement Connection Timeout**: Track idle connections and close them
   after 60 seconds of inactivity.

4. **Add Graceful Shutdown**: On SIGTERM, stop accepting new connections
   and drain existing ones before exit.

## Key Takeaways

1. **select() doesn't scale** - O(n) per iteration, 1024 fd limit
2. **epoll/kqueue are O(1)** - Kernel maintains watch list efficiently
3. **Abstraction layers** - Hide platform differences for portability
4. **Event-driven design** - Callbacks decouple event detection from handling
5. **Connection pools** - Pre-allocated pools avoid malloc overhead

## Next Chapter

Chapter 06 introduces **Connection Pooling** - reusing backend connections
across requests for even better performance (Pingora-style 99%+ reuse).
