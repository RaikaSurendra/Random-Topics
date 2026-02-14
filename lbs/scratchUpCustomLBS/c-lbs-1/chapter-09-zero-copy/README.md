# Chapter 09: Zero-Copy I/O

## Learning Objectives

1. Understand why copying data is expensive
2. Learn how `sendfile()` and `splice()` avoid user-space copies
3. Implement cross-platform zero-copy abstraction
4. Know when zero-copy helps (and when it doesn't)

## The Problem: Data Copying

```
TRADITIONAL I/O (4 copies!)
═══════════════════════════════════════════════════════════════

                      User Space
                    ┌───────────────────────────────────────┐
                    │          Your Application             │
                    │    ┌────────────────────────┐        │
                    │    │    Buffer (16KB)       │        │
                    │    │    COPY #2 → COPY #3   │        │
                    │    └────────────────────────┘        │
                    └───────────────────────────────────────┘
                           ↑                    ↓
    ─────────────────────────────────────────────────────────
                      Kernel Space
                    ┌───────────────────────────────────────┐
                    │  Read Buffer    │   Write Buffer      │
                    │  COPY #1        │   COPY #4           │
                    └───────────────────────────────────────┘
                           ↑                    ↓
    ─────────────────────────────────────────────────────────
                    Disk/Network     Network


ZERO-COPY I/O (0 user-space copies!)
═══════════════════════════════════════════════════════════════

                      User Space
                    ┌───────────────────────────────────────┐
                    │          Your Application             │
                    │    (No data touches user space!)      │
                    └───────────────────────────────────────┘

    ─────────────────────────────────────────────────────────
                      Kernel Space
                    ┌───────────────────────────────────────┐
                    │  Read Buffer ───────→ Write Buffer    │
                    │           (DMA transfer)              │
                    └───────────────────────────────────────┘
                           ↑                    ↓
    ─────────────────────────────────────────────────────────
                    Disk/Network     Network
```

## Platform Support

| Platform | sendfile() | splice() |
|----------|------------|----------|
| Linux    | Yes        | Yes      |
| macOS    | Yes        | No       |
| FreeBSD  | Yes        | No       |
| Windows  | TransmitFile() | No   |

## API

```c
// File to socket (static file serving)
ssize_t zero_copy_file_to_socket(
    int socket_fd,    // Destination socket
    int file_fd,      // Source file
    off_t *offset,    // File offset (updated)
    size_t count      // Bytes to transfer
);

// Socket to socket (proxy relay) - Linux only
ssize_t zero_copy_socket_relay(
    int dest_fd,      // Destination socket
    int src_fd,       // Source socket
    size_t count      // Max bytes to transfer
);

// Check platform support
const char* zero_copy_backend_name(void);
// Returns: "sendfile+splice", "sendfile", or "none"
```

## Linux splice() for Proxy

```c
// splice() transfers data via kernel pipe
// No user-space copy!

int pipefd[2];
pipe(pipefd);

// Step 1: Source socket → Pipe
splice(src_fd, NULL, pipefd[1], NULL, count, SPLICE_F_MOVE);

// Step 2: Pipe → Destination socket
splice(pipefd[0], NULL, dest_fd, NULL, count, SPLICE_F_MOVE);
```

## When to Use Zero-Copy

```
BENEFICIAL (Large transfers)
═══════════════════════════
- Static file serving (images, videos, downloads)
- Large API responses (> 64KB)
- Streaming media
- File uploads/downloads

CPU Savings: 50-70%
Throughput: 2-3x improvement at high bandwidth


NOT BENEFICIAL (Small transfers)
════════════════════════════════
- Small JSON responses (< 1KB)
- Dynamic content requiring transformation
- Already CPU-bound workloads

Overhead of syscall may exceed copy cost
```

## Performance Impact

```
Benchmark: Serving 100MB file, 10Gbps network

┌─────────────────┬──────────────┬──────────────┬────────────┐
│ Method          │ CPU Usage    │ Throughput   │ Latency    │
├─────────────────┼──────────────┼──────────────┼────────────┤
│ read()+write()  │ 95%          │ 3 Gbps       │ 270ms      │
│ sendfile()      │ 35%          │ 9 Gbps       │ 90ms       │
│ splice()        │ 30%          │ 9.5 Gbps     │ 85ms       │
└─────────────────┴──────────────┴──────────────┴────────────┘
```

## Integration Example

```c
// In proxy relay loop
if (response_is_static && content_length > ZERO_COPY_THRESHOLD) {
    // Use zero-copy for large static responses
    zero_copy_socket_relay(client_fd, backend_fd, content_length);
} else {
    // Use buffered I/O for small/dynamic content
    read(backend_fd, buffer, size);
    write(client_fd, buffer, size);
}
```

## Exercises

1. Benchmark zero-copy vs traditional with various file sizes
2. Implement automatic detection of when to use zero-copy
3. Add zero-copy byte counters to metrics
4. Test with `iperf` to measure maximum throughput

## Key Takeaways

1. **Data copying is expensive** - especially at high bandwidth
2. **sendfile()** - file to socket, cross-platform
3. **splice()** - socket to socket, Linux only
4. **Use for large transfers** (> 64KB), avoid for small
5. **Measure first** - zero-copy has syscall overhead
