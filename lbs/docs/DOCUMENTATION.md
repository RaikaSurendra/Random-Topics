# C-Based Load Balancer (LBS) - Complete Technical Documentation & Learning Guide

> **A Comprehensive Technical Book & College Curriculum for Network Programming and Load Balancing**

---

## Table of Contents

### Part I: Foundation & Prerequisites
1. [Course Overview](#chapter-1-course-overview)
2. [Prerequisites & Environment Setup](#chapter-2-prerequisites--environment-setup)
3. [Computer Networking Fundamentals](#chapter-3-computer-networking-fundamentals)
4. [Operating Systems Concepts](#chapter-4-operating-systems-concepts)

### Part II: Core Implementation (Current Codebase)
5. [Socket Programming Fundamentals](#chapter-5-socket-programming-fundamentals)
6. [Proxy Architecture & Implementation](#chapter-6-proxy-architecture--implementation)
7. [Load Balancing Concepts & Algorithms](#chapter-7-load-balancing-concepts--algorithms)
8. [Advanced Load Balancer Features](#chapter-8-advanced-load-balancer-features)

### Part III: Advanced Topics (Implemented)
9. [High-Performance I/O](#chapter-9-high-performance-io) - epoll/kqueue event loop
10. [Connection Pooling](#chapter-10-connection-pooling) - Pingora-style 99%+ reuse
11. [Rate Limiting](#chapter-11-rate-limiting) - Token bucket & sliding window
12. [Metrics & Prometheus](#chapter-12-metrics--prometheus) - Counters, gauges, histograms
13. [Zero-Copy I/O](#chapter-13-zero-copy-io) - sendfile/splice optimization
14. [Hot Reload & Configuration](#chapter-14-hot-reload--configuration) - JSON config, SO_REUSEPORT

### Part IV: Future Curriculum
15. [Security & TLS/SSL](#chapter-15-security--tlsssl)
16. [HTTP Protocol Deep Dive](#chapter-16-http-protocol-deep-dive)
17. [Distributed Systems Concepts](#chapter-17-distributed-systems-concepts)

### Part V: Appendices
- [A. API Reference](#appendix-a-api-reference)
- [B. Glossary](#appendix-b-glossary)
- [C. Further Reading](#appendix-c-further-reading)
- [D. Lab Exercises](#appendix-d-lab-exercises)

---

# Part I: Foundation & Prerequisites

---

## Chapter 1: Course Overview

### 1.1 What is This Project?

This is an educational implementation of a **Layer 4/Layer 7 Load Balancer** written in pure C using POSIX socket APIs. The codebase progressively teaches:

1. **Socket Programming** - The foundation of all network communication
2. **Proxy Patterns** - Forward and reverse proxy architectures
3. **Load Balancing** - Distribution algorithms and health management
4. **Event-Driven Architecture** - Non-blocking I/O for scalability

### 1.2 Learning Objectives

By completing this curriculum, you will:

| Objective | Covered In |
|-----------|------------|
| Understand TCP/IP socket lifecycle | Chapter 5 |
| Implement client-server communication | Chapter 5 |
| Build forward and reverse proxies | Chapter 6 |
| Implement load balancing algorithms | Chapter 7 |
| Handle concurrent connections | Chapter 8 |
| Design event-driven systems | Chapter 8-9 |
| Understand production considerations | Chapter 13 |

### 1.3 Project Structure

```
c-lbs-1/
├── chapter-01-fundamentals/       # Socket basics
│   ├── echo_server.c             # TCP echo server
│   ├── echo_client.c             # TCP client
│   └── README.md
├── chapter-02-simple-proxy/       # Proxy patterns
│   ├── forward_proxy.c           # HTTP CONNECT proxy
│   ├── reverse_proxy.c           # Backend proxy
│   └── README.md
├── chapter-03-load-balancer/      # Basic LB
│   ├── load_balancer.c           # Round-robin LB
│   └── README.md
├── chapter-04-advanced-features/  # Production LB
│   ├── advanced_lb.c             # Multi-algorithm LB
│   └── README.md
├── chapter-05-high-perf-io/       # Event-driven I/O
│   ├── event_loop.h              # Cross-platform abstraction
│   ├── event_loop_epoll.c        # Linux epoll implementation
│   ├── event_loop_kqueue.c       # macOS/BSD kqueue implementation
│   ├── event_loop_select.c       # Portable fallback
│   ├── high_perf_lb.c            # Event-driven load balancer
│   └── README.md
├── chapter-06-connection-pooling/ # Backend connection reuse
│   ├── conn_pool.h               # Pool interface
│   ├── conn_pool.c               # LRU-based pool implementation
│   ├── pooled_lb.c               # LB with connection pooling
│   └── README.md
├── chapter-07-rate-limiting/      # Traffic control
│   ├── rate_limiter.h            # Rate limiter interface
│   ├── rate_limiter.c            # Token bucket & sliding window
│   └── README.md
├── chapter-08-metrics/            # Observability
│   ├── metrics.h                 # Metrics interface
│   ├── metrics.c                 # Prometheus-compatible metrics
│   └── README.md
├── chapter-09-zero-copy/          # High-throughput I/O
│   ├── zero_copy.h               # Zero-copy interface
│   ├── zero_copy.c               # sendfile/splice implementation
│   └── README.md
├── chapter-10-hot-reload/         # Runtime configuration
│   ├── config.h                  # Configuration interface
│   ├── config.c                  # JSON parser, reload logic
│   ├── lb.json                   # Sample configuration
│   └── README.md
└── backends/                      # Test servers
    └── simple_http_backend.c
```

### 1.4 Technology Stack

| Component | Technology | Rationale |
|-----------|------------|-----------|
| Language | C (C99) | Direct system call access, performance |
| Build System | GNU Make | Industry standard, cross-platform |
| Socket API | POSIX | Portable across Unix-like systems |
| I/O Multiplexing | select() | Portable, educational |
| Dependencies | None | Pure POSIX for maximum learning |

### 1.5 How to Use This Guide

```
Beginner Path:           Intermediate Path:        Advanced Path:
Ch 3 → Ch 4 → Ch 5      Ch 5 → Ch 6 → Ch 7       Ch 8 → Ch 9 → Ch 10
     ↓                        ↓                        ↓
  Ch 6 → Ch 7              Ch 8 → Ch 9             Ch 11 → Ch 12
     ↓                        ↓                        ↓
  Ch 8                     Ch 10 → Ch 11           Ch 13 → Ch 14
```

---

## Chapter 2: Prerequisites & Environment Setup

### 2.1 Required Knowledge

#### Essential (Must Have)
- **C Programming**: Pointers, structs, memory management, file I/O
- **Unix/Linux Basics**: Command line, file descriptors, processes
- **Basic Networking**: IP addresses, ports, TCP vs UDP

#### Helpful (Good to Have)
- Data structures (arrays, linked lists, hash tables)
- Basic understanding of HTTP protocol
- Familiarity with debugging tools (gdb, valgrind)

### 2.2 Development Environment Setup

#### macOS
```bash
# Install Xcode Command Line Tools
xcode-select --install

# Verify gcc
gcc --version

# Clone the project
git clone <repository-url>
cd lbs/scratchUpCustomLBS/c-lbs-1
```

#### Linux (Ubuntu/Debian)
```bash
# Install build essentials
sudo apt update
sudo apt install build-essential

# Verify
gcc --version
make --version
```

#### Linux (RHEL/CentOS)
```bash
sudo yum groupinstall "Development Tools"
```

### 2.3 Building the Project

```bash
# Build all chapters
make all

# Build specific chapter
cd chapter-01-fundamentals && make

# Clean all
make clean
```

### 2.4 Testing Infrastructure

```bash
# Start test backends
./backends/simple_http_backend 9001 backend-1 &
./backends/simple_http_backend 9002 backend-2 &
./backends/simple_http_backend 9003 backend-3 &

# Test with curl
curl http://localhost:8080/

# Test with netcat
echo "Hello" | nc localhost 8080
```

### 2.5 Essential Tools

| Tool | Purpose | Installation |
|------|---------|--------------|
| `gcc` | C compiler | System package manager |
| `make` | Build automation | System package manager |
| `curl` | HTTP client | System package manager |
| `nc` (netcat) | TCP/UDP utility | System package manager |
| `lsof` | List open files | System package manager |
| `tcpdump` | Packet capture | System package manager |
| `strace/dtrace` | System call tracing | System package manager |
| `gdb` | Debugger | System package manager |
| `valgrind` | Memory analysis | System package manager |

---

## Chapter 3: Computer Networking Fundamentals

### 3.1 The OSI Model

Understanding the OSI model is crucial for load balancer development:

```
┌─────────────────────────────────────────────────────────────────┐
│ Layer 7: Application  │ HTTP, HTTPS, FTP, DNS                   │
│         (L7 LB)       │ Content-aware routing, URL inspection   │
├─────────────────────────────────────────────────────────────────┤
│ Layer 6: Presentation │ SSL/TLS encryption, data formatting     │
├─────────────────────────────────────────────────────────────────┤
│ Layer 5: Session      │ Connection management, sessions         │
├─────────────────────────────────────────────────────────────────┤
│ Layer 4: Transport    │ TCP, UDP - Port-based routing           │
│         (L4 LB)       │ Connection state, flow control          │
├─────────────────────────────────────────────────────────────────┤
│ Layer 3: Network      │ IP addressing, routing                  │
├─────────────────────────────────────────────────────────────────┤
│ Layer 2: Data Link    │ MAC addresses, switches                 │
├─────────────────────────────────────────────────────────────────┤
│ Layer 1: Physical     │ Cables, signals, bits                   │
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 TCP/IP Model

The practical model used in our implementation:

```
Application Layer  ←─ HTTP requests/responses (Chapter 6-8)
      ↓
Transport Layer    ←─ TCP sockets, ports (Chapter 5)
      ↓
Internet Layer     ←─ IP addresses, routing
      ↓
Network Access     ←─ Ethernet, WiFi
```

### 3.3 TCP Three-Way Handshake

```
Client                              Server
   │                                   │
   │ ──────── SYN (seq=x) ──────────→ │
   │                                   │
   │ ←─── SYN-ACK (seq=y, ack=x+1) ── │
   │                                   │
   │ ──────── ACK (ack=y+1) ─────────→ │
   │                                   │
   │      [Connection Established]     │
```

**In Code (echo_server.c):**
```c
// Server prepares to receive SYN
listen(server_fd, BACKLOG);

// Server accepts SYN, sends SYN-ACK, receives ACK
int client_fd = accept(server_fd, ...);  // Completes handshake
```

### 3.4 TCP Connection Termination

```
Client                              Server
   │                                   │
   │ ──────── FIN ──────────────────→ │
   │                                   │
   │ ←─────── ACK ─────────────────── │
   │                                   │
   │ ←─────── FIN ─────────────────── │
   │                                   │
   │ ──────── ACK ──────────────────→ │
   │                                   │
   │      [Connection Terminated]      │
```

**In Code:**
```c
close(client_fd);  // Initiates FIN sequence
```

### 3.5 Port Numbers

| Range | Name | Examples |
|-------|------|----------|
| 0-1023 | Well-Known | 80 (HTTP), 443 (HTTPS), 22 (SSH) |
| 1024-49151 | Registered | 3306 (MySQL), 5432 (PostgreSQL) |
| 49152-65535 | Dynamic/Private | Client-side ephemeral ports |

### 3.6 IP Addresses

#### IPv4 Address Structure
```
192.168.1.100 = 11000000.10101000.00000001.01100100
     ↓              ↓
 Dotted Decimal   Binary (32 bits)
```

#### Special Addresses
| Address | Meaning |
|---------|---------|
| `127.0.0.1` | Loopback (localhost) |
| `0.0.0.0` | All interfaces (INADDR_ANY) |
| `255.255.255.255` | Broadcast |
| `10.x.x.x`, `172.16-31.x.x`, `192.168.x.x` | Private ranges |

### 3.7 Network Address Translation (NAT)

```
Private Network              NAT Router              Internet
┌──────────────┐            ┌─────────┐           ┌──────────┐
│ 192.168.1.10 │──────────→│         │──────────→│          │
│ 192.168.1.11 │            │ Public  │           │  Server  │
│ 192.168.1.12 │←──────────│ IP:     │←──────────│ 93.1.2.3 │
└──────────────┘            │ 1.2.3.4 │           └──────────┘
                            └─────────┘
```

### 3.8 DNS Resolution

```
Application: "connect to example.com"
      │
      ↓
DNS Resolver → DNS Server → Returns IP: 93.184.216.34
      │
      ↓
Application: connect(93.184.216.34)
```

**In Code (echo_client.c):**
```c
struct addrinfo hints, *res;
getaddrinfo("example.com", "80", &hints, &res);
// res now contains resolved IP addresses
```

### 3.9 Lab Exercise: Network Analysis

```bash
# 1. Capture TCP handshake
sudo tcpdump -i any port 8080 -X

# 2. In another terminal, run echo client
./echo_client localhost 8080

# 3. Observe SYN, SYN-ACK, ACK sequence
```

---

## Chapter 4: Operating Systems Concepts

### 4.1 File Descriptors

In Unix, everything is a file - including network connections:

```
┌────────────────────────────────────────────────┐
│ Process File Descriptor Table                  │
├────────┬───────────────────────────────────────┤
│ FD 0   │ stdin (keyboard input)                │
│ FD 1   │ stdout (terminal output)              │
│ FD 2   │ stderr (error output)                 │
│ FD 3   │ Socket (listening server)             │
│ FD 4   │ Socket (client connection #1)         │
│ FD 5   │ Socket (client connection #2)         │
│ ...    │ ...                                   │
└────────┴───────────────────────────────────────┘
```

**In Code:**
```c
int server_fd = socket(...);  // Returns FD (e.g., 3)
int client_fd = accept(...);  // Returns FD (e.g., 4)
```

### 4.2 Process Memory Layout

```
High Address
┌────────────────────────┐
│ Kernel Space           │  (Inaccessible to process)
├────────────────────────┤
│ Stack                  │  (Function calls, local vars)
│ ↓                      │
│                        │
│                        │
│ ↑                      │
│ Heap                   │  (malloc, dynamic allocation)
├────────────────────────┤
│ BSS                    │  (Uninitialized globals)
├────────────────────────┤
│ Data                   │  (Initialized globals)
├────────────────────────┤
│ Text                   │  (Program code)
└────────────────────────┘
Low Address
```

### 4.3 System Calls

Socket operations are system calls that transition from user space to kernel space:

```
User Space                    Kernel Space
┌─────────────┐              ┌─────────────────────┐
│ Your Code   │              │ TCP/IP Stack        │
│             │   syscall    │                     │
│ socket()  ──┼─────────────→│ Create socket       │
│             │              │                     │
│ bind()    ──┼─────────────→│ Assign address      │
│             │              │                     │
│ listen()  ──┼─────────────→│ Mark as passive     │
│             │              │                     │
│ accept()  ──┼─────────────→│ Accept connection   │
│             │   blocking   │                     │
└─────────────┘              └─────────────────────┘
```

### 4.4 Blocking vs Non-Blocking I/O

#### Blocking I/O (Chapters 1-3)
```c
// Process sleeps until data arrives
bytes = read(fd, buffer, size);  // Blocks here
// Continues after data received
```

```
Time →
Process: [Working][Blocked.........][Working]
                  ↑
          Waiting for I/O
```

#### Non-Blocking I/O (Chapter 4)
```c
// Set socket to non-blocking
fcntl(fd, F_SETFL, O_NONBLOCK);

// Returns immediately, may have no data
bytes = read(fd, buffer, size);
if (bytes == -1 && errno == EAGAIN) {
    // No data available, try later
}
```

```
Time →
Process: [Working][Poll][Working][Poll][Working]
                    ↑              ↑
           Check for data    Check for data
```

### 4.5 I/O Multiplexing with select()

```c
fd_set read_fds;
FD_ZERO(&read_fds);
FD_SET(client1_fd, &read_fds);
FD_SET(client2_fd, &read_fds);

// Wait for any socket to be readable
select(max_fd + 1, &read_fds, NULL, NULL, &timeout);

if (FD_ISSET(client1_fd, &read_fds)) {
    // client1 has data
}
if (FD_ISSET(client2_fd, &read_fds)) {
    // client2 has data
}
```

**Visual:**
```
select() monitors multiple FDs simultaneously:

       ┌──────────────────────────────────┐
       │            select()              │
       │                                  │
       │  ┌─────┐ ┌─────┐ ┌─────┐        │
       │  │FD 3 │ │FD 4 │ │FD 5 │ ...    │
       │  └──┬──┘ └──┬──┘ └──┬──┘        │
       └─────┼───────┼───────┼───────────┘
             │       │       │
             ↓       ↓       ↓
          Readable? Readable? Readable?
```

### 4.6 Signal Handling

```c
// Signal handler function
void handle_signal(int sig) {
    if (sig == SIGINT || sig == SIGTERM) {
        running = 0;  // Graceful shutdown
    }
}

// Register handler
signal(SIGINT, handle_signal);
signal(SIGTERM, handle_signal);
signal(SIGPIPE, SIG_IGN);  // Ignore broken pipe
```

**Common Signals:**
| Signal | Default | Use in LB |
|--------|---------|-----------|
| SIGINT | Terminate | Graceful shutdown |
| SIGTERM | Terminate | Graceful shutdown |
| SIGPIPE | Terminate | Ignore (client disconnect) |
| SIGUSR1 | Terminate | Print statistics |
| SIGHUP | Terminate | Reload configuration |

### 4.7 Socket Options

```c
// Allow port reuse after server restart
int optval = 1;
setsockopt(fd, SOL_SOCKET, SO_REUSEADDR, &optval, sizeof(optval));

// Set receive timeout
struct timeval timeout = {5, 0};  // 5 seconds
setsockopt(fd, SOL_SOCKET, SO_RCVTIMEO, &timeout, sizeof(timeout));

// Set send buffer size
int bufsize = 65536;
setsockopt(fd, SOL_SOCKET, SO_SNDBUF, &bufsize, sizeof(bufsize));
```

### 4.8 Lab Exercise: System Call Tracing

```bash
# Trace system calls of echo server
strace -f ./echo_server 8080

# You'll see:
# socket(AF_INET, SOCK_STREAM, ...) = 3
# setsockopt(3, SOL_SOCKET, SO_REUSEADDR, ...) = 0
# bind(3, {sa_family=AF_INET, sin_port=htons(8080), ...}) = 0
# listen(3, 10) = 0
# accept(3, ...)  # blocks here waiting for client
```

---

# Part II: Core Implementation (Current Codebase)

---

## Chapter 5: Socket Programming Fundamentals

### 5.1 The Socket API

The Berkeley Sockets API provides the foundation for network programming:

```
┌─────────────────────────────────────────────────────────────┐
│                    SOCKET LIFECYCLE                         │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│    SERVER                           CLIENT                  │
│    ──────                           ──────                  │
│                                                             │
│    socket()                         socket()                │
│       │                                │                    │
│       ↓                                │                    │
│    bind()                              │                    │
│       │                                │                    │
│       ↓                                │                    │
│    listen()                            │                    │
│       │                                │                    │
│       ↓                                ↓                    │
│    accept() ←─── 3-way handshake ─── connect()             │
│       │                                │                    │
│       ↓                                ↓                    │
│    read() ←──────── data ───────── write()                 │
│       │                                │                    │
│       ↓                                ↓                    │
│    write() ─────── data ──────────→ read()                 │
│       │                                │                    │
│       ↓                                ↓                    │
│    close()                          close()                 │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 5.2 Creating a Socket

```c
#include <sys/socket.h>
#include <netinet/in.h>

int socket(int domain, int type, int protocol);

// Example: Create TCP/IPv4 socket
int fd = socket(AF_INET, SOCK_STREAM, 0);
```

**Parameters:**
| Parameter | Common Values | Description |
|-----------|---------------|-------------|
| domain | AF_INET, AF_INET6 | IPv4 or IPv6 |
| type | SOCK_STREAM, SOCK_DGRAM | TCP or UDP |
| protocol | 0 | Auto-select (TCP for STREAM) |

### 5.3 Address Structures

```c
// IPv4 address structure
struct sockaddr_in {
    sa_family_t    sin_family;   // AF_INET
    in_port_t      sin_port;     // Port (network byte order)
    struct in_addr sin_addr;     // IP address
};

struct in_addr {
    uint32_t s_addr;             // IP (network byte order)
};

// Usage
struct sockaddr_in addr;
memset(&addr, 0, sizeof(addr));
addr.sin_family = AF_INET;
addr.sin_port = htons(8080);           // Host to Network Short
addr.sin_addr.s_addr = INADDR_ANY;     // 0.0.0.0
```

### 5.4 Byte Order Conversion

Network protocols use **big-endian** (network byte order):

```c
// Host to Network
uint16_t htons(uint16_t hostshort);   // short (16-bit)
uint32_t htonl(uint32_t hostlong);    // long (32-bit)

// Network to Host
uint16_t ntohs(uint16_t netshort);
uint32_t ntohl(uint32_t netlong);

// Example
addr.sin_port = htons(8080);  // 8080 → 0x1F90 → 0x901F (on little-endian)
```

### 5.5 Binding and Listening

```c
// Bind socket to address
int bind(int sockfd, const struct sockaddr *addr, socklen_t addrlen);

// Mark socket as passive (server)
int listen(int sockfd, int backlog);

// Example
bind(server_fd, (struct sockaddr*)&addr, sizeof(addr));
listen(server_fd, 10);  // Queue up to 10 pending connections
```

**The Backlog Parameter:**
```
                    ┌───────────────────────────────────┐
Incoming            │        Connection Queue           │
Connections ───────→│ [Pending] [Pending] [Pending]    │───→ accept()
                    │         (backlog = 3)             │
                    └───────────────────────────────────┘
                                    │
                    If queue full, new connections rejected
```

### 5.6 Accepting Connections

```c
int accept(int sockfd, struct sockaddr *addr, socklen_t *addrlen);

// Example
struct sockaddr_in client_addr;
socklen_t client_len = sizeof(client_addr);
int client_fd = accept(server_fd, (struct sockaddr*)&client_addr, &client_len);

// Get client IP
char client_ip[INET_ADDRSTRLEN];
inet_ntop(AF_INET, &client_addr.sin_addr, client_ip, sizeof(client_ip));
printf("Client connected: %s\n", client_ip);
```

### 5.7 Connecting (Client Side)

```c
int connect(int sockfd, const struct sockaddr *addr, socklen_t addrlen);

// Modern approach using getaddrinfo
struct addrinfo hints, *res;
memset(&hints, 0, sizeof(hints));
hints.ai_family = AF_INET;
hints.ai_socktype = SOCK_STREAM;

getaddrinfo("example.com", "80", &hints, &res);

int fd = socket(res->ai_family, res->ai_socktype, res->ai_protocol);
connect(fd, res->ai_addr, res->ai_addrlen);

freeaddrinfo(res);
```

### 5.8 Reading and Writing

```c
// Read from socket
ssize_t read(int fd, void *buf, size_t count);
ssize_t recv(int sockfd, void *buf, size_t len, int flags);

// Write to socket
ssize_t write(int fd, const void *buf, size_t count);
ssize_t send(int sockfd, const void *buf, size_t len, int flags);

// Example
char buffer[1024];
ssize_t bytes = read(client_fd, buffer, sizeof(buffer) - 1);
if (bytes > 0) {
    buffer[bytes] = '\0';
    write(client_fd, buffer, bytes);  // Echo back
}
```

**Return Values:**
| Value | Meaning |
|-------|---------|
| > 0 | Number of bytes read/written |
| 0 | Connection closed (EOF) |
| -1 | Error (check errno) |

### 5.9 Complete Echo Server Example

```c
// echo_server.c - Annotated
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>

#define BUFFER_SIZE 1024
#define BACKLOG 10

int main(int argc, char *argv[]) {
    if (argc != 2) {
        fprintf(stderr, "Usage: %s <port>\n", argv[0]);
        exit(1);
    }

    int port = atoi(argv[1]);

    // 1. Create socket
    int server_fd = socket(AF_INET, SOCK_STREAM, 0);
    if (server_fd < 0) {
        perror("socket");
        exit(1);
    }

    // 2. Allow port reuse
    int optval = 1;
    setsockopt(server_fd, SOL_SOCKET, SO_REUSEADDR, &optval, sizeof(optval));

    // 3. Bind to address
    struct sockaddr_in addr = {
        .sin_family = AF_INET,
        .sin_port = htons(port),
        .sin_addr.s_addr = INADDR_ANY
    };

    if (bind(server_fd, (struct sockaddr*)&addr, sizeof(addr)) < 0) {
        perror("bind");
        exit(1);
    }

    // 4. Listen for connections
    if (listen(server_fd, BACKLOG) < 0) {
        perror("listen");
        exit(1);
    }

    printf("Echo server listening on port %d\n", port);

    // 5. Accept loop
    while (1) {
        struct sockaddr_in client_addr;
        socklen_t client_len = sizeof(client_addr);

        int client_fd = accept(server_fd,
                               (struct sockaddr*)&client_addr,
                               &client_len);
        if (client_fd < 0) {
            perror("accept");
            continue;
        }

        char client_ip[INET_ADDRSTRLEN];
        inet_ntop(AF_INET, &client_addr.sin_addr, client_ip, sizeof(client_ip));
        printf("Client connected: %s\n", client_ip);

        // 6. Handle client
        char buffer[BUFFER_SIZE];
        ssize_t bytes;

        while ((bytes = read(client_fd, buffer, sizeof(buffer) - 1)) > 0) {
            buffer[bytes] = '\0';
            printf("Received: %s", buffer);
            write(client_fd, buffer, bytes);  // Echo
        }

        printf("Client disconnected: %s\n", client_ip);
        close(client_fd);
    }

    close(server_fd);
    return 0;
}
```

### 5.10 Key Concepts Summary

| Concept | Function | Purpose |
|---------|----------|---------|
| Socket Creation | socket() | Create communication endpoint |
| Address Binding | bind() | Assign address to socket |
| Passive Mode | listen() | Prepare to accept connections |
| Accept Connection | accept() | Get new client socket |
| Client Connect | connect() | Establish server connection |
| Data Transfer | read()/write() | Send/receive data |
| Cleanup | close() | Release resources |

### 5.11 Lab Exercises

**Exercise 5.1:** Modify echo server to convert text to uppercase before echoing.

**Exercise 5.2:** Implement a "time server" that returns current time to clients.

**Exercise 5.3:** Create a simple chat server that broadcasts messages to all connected clients.

**Exercise 5.4:** Add connection logging that writes to a file.

---

## Chapter 6: Proxy Architecture & Implementation

### 6.1 What is a Proxy?

A proxy is an intermediary that forwards requests between clients and servers:

```
Without Proxy:
Client ←─────────────────────────────────────────→ Server

With Proxy:
Client ←────────→ Proxy ←────────→ Server
       Request            Request
       ←────────  ←────────
       Response   Response
```

### 6.2 Types of Proxies

#### Forward Proxy (Client-Side)
```
┌────────────────────────────────────────────────────────────┐
│                                                            │
│   Internal Network              │        Internet          │
│                                 │                          │
│   ┌────────┐    ┌────────┐     │     ┌────────────────┐   │
│   │Client 1│───→│        │     │     │                │   │
│   └────────┘    │Forward │─────┼────→│ Web Servers    │   │
│   ┌────────┐───→│ Proxy  │     │     │                │   │
│   │Client 2│    │        │     │     └────────────────┘   │
│   └────────┘    └────────┘     │                          │
│                                 │                          │
│                            Firewall                        │
└────────────────────────────────────────────────────────────┘
```

**Use Cases:**
- Access control and filtering
- Caching for performance
- Anonymity and privacy
- Bypassing geographic restrictions

#### Reverse Proxy (Server-Side)
```
┌────────────────────────────────────────────────────────────┐
│                                                            │
│        Internet                 │    Internal Network      │
│                                 │                          │
│   ┌────────────┐               │    ┌───────────────┐     │
│   │            │    ┌────────┐ │    │ Backend 1     │     │
│   │  Clients   │───→│Reverse │─┼───→│ Backend 2     │     │
│   │            │    │ Proxy  │ │    │ Backend 3     │     │
│   └────────────┘    └────────┘ │    └───────────────┘     │
│                                 │                          │
│                            Firewall                        │
└────────────────────────────────────────────────────────────┘
```

**Use Cases:**
- Load balancing
- SSL termination
- Compression
- Caching
- Security (hide backend servers)

### 6.3 HTTP Protocol Basics

#### HTTP Request Structure
```
GET /path/to/resource HTTP/1.1\r\n
Host: www.example.com\r\n
User-Agent: Mozilla/5.0\r\n
Accept: text/html\r\n
Connection: keep-alive\r\n
\r\n
[Optional Body]
```

#### HTTP Response Structure
```
HTTP/1.1 200 OK\r\n
Content-Type: text/html\r\n
Content-Length: 1234\r\n
Connection: keep-alive\r\n
\r\n
[Response Body]
```

### 6.4 Forward Proxy Implementation

#### HTTP CONNECT Method (HTTPS Tunneling)
```
Client → Proxy: CONNECT www.example.com:443 HTTP/1.1
Proxy → Server: [TCP Connect to example.com:443]
Proxy → Client: HTTP/1.1 200 Connection Established
Client ↔ Proxy ↔ Server: [Encrypted TLS data passed through]
```

**Implementation (forward_proxy.c):**
```c
int parse_connect_request(const char *request, char *host, char *port) {
    // Parse: "CONNECT host:port HTTP/1.1"
    if (strncmp(request, "CONNECT ", 8) != 0) {
        return -1;
    }

    const char *start = request + 8;
    const char *colon = strchr(start, ':');
    const char *space = strchr(start, ' ');

    if (!colon || !space) return -1;

    strncpy(host, start, colon - start);
    host[colon - start] = '\0';

    strncpy(port, colon + 1, space - colon - 1);
    port[space - colon - 1] = '\0';

    return 0;
}

void handle_connect_tunnel(int client_fd, int target_fd) {
    // Send success response
    const char *response = "HTTP/1.1 200 Connection Established\r\n\r\n";
    write(client_fd, response, strlen(response));

    // Bidirectional relay using select()
    fd_set read_fds;
    char buffer[8192];
    int max_fd = (client_fd > target_fd) ? client_fd : target_fd;

    while (1) {
        FD_ZERO(&read_fds);
        FD_SET(client_fd, &read_fds);
        FD_SET(target_fd, &read_fds);

        if (select(max_fd + 1, &read_fds, NULL, NULL, NULL) < 0) {
            break;
        }

        // Client → Target
        if (FD_ISSET(client_fd, &read_fds)) {
            ssize_t n = read(client_fd, buffer, sizeof(buffer));
            if (n <= 0) break;
            write(target_fd, buffer, n);
        }

        // Target → Client
        if (FD_ISSET(target_fd, &read_fds)) {
            ssize_t n = read(target_fd, buffer, sizeof(buffer));
            if (n <= 0) break;
            write(client_fd, buffer, n);
        }
    }
}
```

### 6.5 Reverse Proxy Implementation

```c
// reverse_proxy.c - Core logic
void handle_client(int client_fd, const char *backend_host, int backend_port) {
    // Connect to backend
    int backend_fd = connect_to_backend(backend_host, backend_port);
    if (backend_fd < 0) {
        send_error(client_fd, 502, "Bad Gateway");
        return;
    }

    // Relay data bidirectionally
    relay_data(client_fd, backend_fd);

    close(backend_fd);
}

void relay_data(int client_fd, int backend_fd) {
    fd_set read_fds;
    char buffer[BUFFER_SIZE];
    int max_fd = (client_fd > backend_fd) ? client_fd : backend_fd;

    while (1) {
        FD_ZERO(&read_fds);
        FD_SET(client_fd, &read_fds);
        FD_SET(backend_fd, &read_fds);

        struct timeval timeout = {30, 0};  // 30 second timeout

        int ready = select(max_fd + 1, &read_fds, NULL, NULL, &timeout);
        if (ready <= 0) break;

        if (FD_ISSET(client_fd, &read_fds)) {
            ssize_t n = read(client_fd, buffer, sizeof(buffer));
            if (n <= 0) break;
            write(backend_fd, buffer, n);
        }

        if (FD_ISSET(backend_fd, &read_fds)) {
            ssize_t n = read(backend_fd, buffer, sizeof(buffer));
            if (n <= 0) break;
            write(client_fd, buffer, n);
        }
    }
}
```

### 6.6 Error Handling

```c
void send_error(int fd, int code, const char *message) {
    char response[512];
    int len = snprintf(response, sizeof(response),
        "HTTP/1.1 %d %s\r\n"
        "Content-Type: text/plain\r\n"
        "Content-Length: %zu\r\n"
        "Connection: close\r\n"
        "\r\n"
        "%s\n",
        code, message, strlen(message) + 1, message);

    write(fd, response, len);
}
```

**Common Error Codes:**
| Code | Message | Meaning |
|------|---------|---------|
| 400 | Bad Request | Malformed request |
| 502 | Bad Gateway | Backend unreachable |
| 503 | Service Unavailable | No healthy backends |
| 504 | Gateway Timeout | Backend timeout |

### 6.7 Data Flow Diagram

```
┌────────────────────────────────────────────────────────────────┐
│                    REVERSE PROXY DATA FLOW                     │
├────────────────────────────────────────────────────────────────┤
│                                                                │
│  Client               Proxy                    Backend         │
│    │                    │                         │            │
│    │──── HTTP Req ─────→│                         │            │
│    │                    │──── HTTP Req ──────────→│            │
│    │                    │                         │            │
│    │                    │←─── HTTP Resp ──────────│            │
│    │←─── HTTP Resp ─────│                         │            │
│    │                    │                         │            │
│                                                                │
│  Timeline:                                                     │
│  1. Client connects to proxy                                   │
│  2. Proxy accepts connection                                   │
│  3. Proxy connects to backend                                  │
│  4. Proxy reads request from client                           │
│  5. Proxy forwards request to backend                         │
│  6. Backend processes request                                 │
│  7. Proxy reads response from backend                         │
│  8. Proxy forwards response to client                         │
│  9. Connections closed                                        │
│                                                                │
└────────────────────────────────────────────────────────────────┘
```

### 6.8 Lab Exercises

**Exercise 6.1:** Add request logging to reverse proxy (method, URL, response code, latency).

**Exercise 6.2:** Implement a simple HTTP cache in the forward proxy.

**Exercise 6.3:** Add custom header injection (X-Forwarded-For).

**Exercise 6.4:** Implement connection timeout handling.

---

## Chapter 7: Load Balancing Concepts & Algorithms

### 7.1 What is Load Balancing?

Load balancing distributes incoming traffic across multiple backend servers:

```
                                    ┌─────────────┐
                                ┌──→│ Backend 1   │
                                │   └─────────────┘
┌──────────┐     ┌────────────┐ │   ┌─────────────┐
│ Clients  │────→│    Load    │─┼──→│ Backend 2   │
└──────────┘     │  Balancer  │ │   └─────────────┘
                 └────────────┘ │   ┌─────────────┐
                                └──→│ Backend 3   │
                                    └─────────────┘
```

### 7.2 Benefits of Load Balancing

| Benefit | Description |
|---------|-------------|
| **High Availability** | If one server fails, others continue serving |
| **Scalability** | Add more servers to handle increased load |
| **Performance** | Distribute load evenly for better response times |
| **Maintenance** | Take servers offline without downtime |
| **Geographic Distribution** | Route to nearest server |

### 7.3 Load Balancing Algorithms

#### 7.3.1 Round Robin

The simplest algorithm - requests are distributed sequentially:

```
Request 1 → Backend 1
Request 2 → Backend 2
Request 3 → Backend 3
Request 4 → Backend 1  (cycle repeats)
...
```

**Implementation:**
```c
typedef struct {
    Backend backends[MAX_BACKENDS];
    int num_backends;
    int current_index;  // Tracks position
} LoadBalancer;

Backend* select_round_robin(LoadBalancer *lb) {
    int attempts = 0;

    while (attempts < lb->num_backends) {
        lb->current_index = (lb->current_index + 1) % lb->num_backends;

        if (lb->backends[lb->current_index].is_healthy) {
            return &lb->backends[lb->current_index];
        }
        attempts++;
    }

    return NULL;  // No healthy backends
}
```

**Pros:** Simple, no state tracking needed
**Cons:** Doesn't account for server capacity or current load

#### 7.3.2 Weighted Round Robin

Assigns weights based on server capacity:

```
Backend 1: weight=3  →  Gets 3/6 (50%) of requests
Backend 2: weight=2  →  Gets 2/6 (33%) of requests
Backend 3: weight=1  →  Gets 1/6 (17%) of requests

Sequence: 1,1,1,2,2,3,1,1,1,2,2,3,...
```

**Implementation (Smooth Weighted Round Robin):**
```c
Backend* select_weighted_round_robin(LoadBalancer *lb) {
    Backend *best = NULL;
    int total_weight = 0;

    // Calculate total weight and find best candidate
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

**Why Smooth?** Instead of `1,1,1,2,2,3`, produces `1,2,1,3,1,2` - better distribution.

#### 7.3.3 Least Connections

Routes to the server with fewest active connections:

```
Backend 1: 5 connections
Backend 2: 3 connections  ← Selected (fewest)
Backend 3: 8 connections
```

**Implementation:**
```c
Backend* select_least_connections(LoadBalancer *lb) {
    Backend *best = NULL;
    int min_connections = INT_MAX;

    for (int i = 0; i < lb->num_backends; i++) {
        Backend *b = &lb->backends[i];
        if (!b->is_healthy) continue;

        if (b->active_connections < min_connections) {
            min_connections = b->active_connections;
            best = b;
        }
    }

    return best;
}
```

**Weight-Adjusted Version:**
```c
// Score = connections / weight (lower is better)
int score = (b->active_connections * 100) / b->weight;
```

#### 7.3.4 IP Hash (Session Persistence)

Same client IP always goes to same backend:

```
Client 192.168.1.10 → hash → Backend 2
Client 192.168.1.11 → hash → Backend 1
Client 192.168.1.10 → hash → Backend 2  (same as before)
```

**Implementation:**
```c
Backend* select_ip_hash(LoadBalancer *lb, const char *client_ip) {
    // Simple hash function
    unsigned int hash = 0;
    for (const char *p = client_ip; *p; p++) {
        hash = hash * 31 + *p;
    }

    // Find healthy backend starting from hash position
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

**Use Cases:**
- Shopping carts (session affinity)
- WebSocket connections
- Stateful applications

### 7.4 Algorithm Comparison

| Algorithm | Complexity | Statefulness | Best For |
|-----------|------------|--------------|----------|
| Round Robin | O(1) | Minimal | Homogeneous backends, stateless apps |
| Weighted RR | O(n) | Per-backend weight state | Heterogeneous backend capacities |
| Least Connections | O(n) | Connection counters | Long-lived connections, varying workloads |
| IP Hash | O(1) | None (computed) | Session persistence required |

### 7.5 Health Checking

Health checks detect failed backends before routing traffic to them:

```c
#define HEALTH_CHECK_INTERVAL 10  // seconds

int check_backend_health(Backend *backend) {
    // Create test socket
    int sock = socket(AF_INET, SOCK_STREAM, 0);
    if (sock < 0) return 0;

    // Set timeout
    struct timeval timeout = {2, 0};  // 2 seconds
    setsockopt(sock, SOL_SOCKET, SO_SNDTIMEO, &timeout, sizeof(timeout));
    setsockopt(sock, SOL_SOCKET, SO_RCVTIMEO, &timeout, sizeof(timeout));

    // Try to connect
    struct sockaddr_in addr;
    addr.sin_family = AF_INET;
    addr.sin_port = htons(backend->port);
    inet_pton(AF_INET, backend->host, &addr.sin_addr);

    int result = connect(sock, (struct sockaddr*)&addr, sizeof(addr));
    close(sock);

    return (result == 0);  // Success = healthy
}

void health_check_all(LoadBalancer *lb) {
    time_t now = time(NULL);

    for (int i = 0; i < lb->num_backends; i++) {
        Backend *b = &lb->backends[i];

        if (now - b->last_health_check >= HEALTH_CHECK_INTERVAL) {
            int was_healthy = b->is_healthy;
            b->is_healthy = check_backend_health(b);
            b->last_health_check = now;

            if (was_healthy && !b->is_healthy) {
                printf("Backend %s:%d marked UNHEALTHY\n", b->host, b->port);
            } else if (!was_healthy && b->is_healthy) {
                printf("Backend %s:%d marked HEALTHY\n", b->host, b->port);
            }
        }
    }
}
```

### 7.6 Health Check Types

| Type | Method | Pros | Cons |
|------|--------|------|------|
| **TCP Connect** | Just open connection | Simple, fast | Doesn't verify app health |
| **HTTP GET** | Request specific endpoint | Verifies app responds | More overhead |
| **Custom Script** | Run external script | Flexible | Complex, slower |
| **Passive** | Monitor real traffic | No extra load | Slower detection |

### 7.7 Statistics Tracking

```c
typedef struct {
    unsigned long total_requests;
    unsigned long failed_requests;
    unsigned long bytes_in;
    unsigned long bytes_out;
    double avg_response_time;
} BackendStats;

void print_statistics(LoadBalancer *lb) {
    time_t uptime = time(NULL) - lb->start_time;

    printf("\n=== Load Balancer Statistics ===\n");
    printf("Uptime: %ld seconds\n", uptime);
    printf("Total requests: %lu\n", lb->total_requests);
    printf("Requests/sec: %.2f\n",
           (double)lb->total_requests / uptime);

    printf("\n--- Backend Statistics ---\n");
    for (int i = 0; i < lb->num_backends; i++) {
        Backend *b = &lb->backends[i];
        printf("%s:%d - Requests: %lu, Failed: %lu, Health: %s\n",
               b->host, b->port,
               b->total_requests, b->failed_requests,
               b->is_healthy ? "UP" : "DOWN");
    }
}
```

### 7.8 Complete Load Balancer Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                    LOAD BALANCER MAIN LOOP                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  1. Initialize                                                  │
│     ├── Parse backend list from arguments                       │
│     ├── Create listening socket                                 │
│     └── Register signal handlers                                │
│                                                                 │
│  2. Main Loop                                                   │
│     ┌─────────────────────────────────────────┐                │
│     │  while (running) {                      │                │
│     │      // Periodic health checks          │                │
│     │      if (time_for_health_check())       │                │
│     │          health_check_all();            │                │
│     │                                         │                │
│     │      // Wait for client connection      │                │
│     │      client_fd = accept();              │                │
│     │                                         │                │
│     │      // Select backend                  │                │
│     │      backend = select_backend();        │                │
│     │                                         │                │
│     │      // Connect to backend              │                │
│     │      backend_fd = connect_to_backend(); │                │
│     │                                         │                │
│     │      // Relay data                      │                │
│     │      relay_data(client_fd, backend_fd); │                │
│     │                                         │                │
│     │      // Update statistics               │                │
│     │      backend->total_requests++;         │                │
│     │                                         │                │
│     │      // Cleanup                         │                │
│     │      close(client_fd);                  │                │
│     │      close(backend_fd);                 │                │
│     │  }                                      │                │
│     └─────────────────────────────────────────┘                │
│                                                                 │
│  3. Shutdown                                                    │
│     ├── Print final statistics                                  │
│     └── Close listening socket                                  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 7.9 Lab Exercises

**Exercise 7.1:** Implement "Random" load balancing algorithm.

**Exercise 7.2:** Add "Response Time" based selection (route to fastest backend).

**Exercise 7.3:** Implement HTTP health checks (verify specific endpoint returns 200).

**Exercise 7.4:** Add backend warm-up (gradually increase traffic to recovered backend).

---

## Chapter 8: Advanced Load Balancer Features

### 8.1 Event-Driven Architecture

The advanced load balancer handles multiple concurrent connections using non-blocking I/O:

```
┌─────────────────────────────────────────────────────────────────┐
│                    EVENT-DRIVEN MODEL                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Traditional (Blocking):                                        │
│  ┌────────┐ ┌────────┐ ┌────────┐                              │
│  │Thread 1│ │Thread 2│ │Thread 3│  One thread per client       │
│  │Client 1│ │Client 2│ │Client 3│  Heavy resource usage        │
│  └────────┘ └────────┘ └────────┘                              │
│                                                                 │
│  Event-Driven (Non-Blocking):                                   │
│  ┌──────────────────────────────────────────┐                  │
│  │              Single Thread               │                  │
│  │  ┌─────────────────────────────────────┐│                  │
│  │  │           Event Loop                ││                  │
│  │  │  select()/poll()/epoll()           ││                  │
│  │  │                                     ││                  │
│  │  │  Client 1 ──┐                       ││                  │
│  │  │  Client 2 ──┼──→ Process events     ││                  │
│  │  │  Client 3 ──┘                       ││                  │
│  │  └─────────────────────────────────────┘│                  │
│  └──────────────────────────────────────────┘                  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 8.2 Connection Management

```c
#define MAX_CLIENTS 256

typedef struct {
    int client_fd;
    int backend_fd;
    Backend *backend;
    char client_ip[INET_ADDRSTRLEN];
    char buffer[BUFFER_SIZE];
    size_t buffer_len;
    time_t start_time;
    int state;  // CONNECTING, RELAYING, CLOSING
} Connection;

typedef struct {
    Connection connections[MAX_CLIENTS];
    int num_connections;
} ConnectionPool;

Connection* allocate_connection(ConnectionPool *pool) {
    for (int i = 0; i < MAX_CLIENTS; i++) {
        if (pool->connections[i].client_fd == -1) {
            pool->num_connections++;
            return &pool->connections[i];
        }
    }
    return NULL;  // Pool exhausted
}

void free_connection(ConnectionPool *pool, Connection *conn) {
    if (conn->client_fd >= 0) close(conn->client_fd);
    if (conn->backend_fd >= 0) close(conn->backend_fd);
    conn->client_fd = -1;
    conn->backend_fd = -1;
    conn->backend = NULL;
    pool->num_connections--;
}
```

### 8.3 Non-Blocking Event Loop

```c
void event_loop(LoadBalancer *lb, ConnectionPool *pool) {
    fd_set read_fds;
    int max_fd;

    while (g_running) {
        // Periodic health checks
        health_check_all(lb);

        // Build fd_set
        FD_ZERO(&read_fds);
        FD_SET(lb->server_fd, &read_fds);
        max_fd = lb->server_fd;

        // Add all active connections
        for (int i = 0; i < MAX_CLIENTS; i++) {
            Connection *c = &pool->connections[i];
            if (c->client_fd >= 0) {
                FD_SET(c->client_fd, &read_fds);
                if (c->client_fd > max_fd) max_fd = c->client_fd;
            }
            if (c->backend_fd >= 0) {
                FD_SET(c->backend_fd, &read_fds);
                if (c->backend_fd > max_fd) max_fd = c->backend_fd;
            }
        }

        // Wait for events (with timeout for health checks)
        struct timeval timeout = {1, 0};  // 1 second
        int ready = select(max_fd + 1, &read_fds, NULL, NULL, &timeout);

        if (ready < 0) {
            if (errno == EINTR) continue;  // Signal interrupted
            perror("select");
            break;
        }

        // New client connection
        if (FD_ISSET(lb->server_fd, &read_fds)) {
            accept_new_client(lb, pool);
        }

        // Process existing connections
        for (int i = 0; i < MAX_CLIENTS; i++) {
            Connection *c = &pool->connections[i];

            // Client → Backend
            if (c->client_fd >= 0 && FD_ISSET(c->client_fd, &read_fds)) {
                handle_client_data(c);
            }

            // Backend → Client
            if (c->backend_fd >= 0 && FD_ISSET(c->backend_fd, &read_fds)) {
                handle_backend_data(c);
            }
        }
    }
}
```

### 8.4 Header Injection

Add custom headers for backend awareness:

```c
void inject_headers(char *buffer, size_t *len, size_t max_len,
                    const char *client_ip, const char *proto) {
    // Find end of first line
    char *line_end = strstr(buffer, "\r\n");
    if (!line_end) return;

    // Headers to inject
    char headers[256];
    int headers_len = snprintf(headers, sizeof(headers),
        "\r\nX-Forwarded-For: %s"
        "\r\nX-Forwarded-Proto: %s"
        "\r\nX-Real-IP: %s",
        client_ip, proto, client_ip);

    // Check space available
    if (*len + headers_len >= max_len) return;

    // Insert headers after first line
    char *insert_point = line_end + 2;
    size_t remaining = *len - (insert_point - buffer);

    memmove(insert_point + headers_len, insert_point, remaining);
    memcpy(insert_point, headers + 2, headers_len);  // Skip initial \r\n
    *len += headers_len;
}
```

**Standard Headers:**
| Header | Purpose |
|--------|---------|
| X-Forwarded-For | Original client IP |
| X-Forwarded-Proto | Original protocol (http/https) |
| X-Real-IP | Client IP (single value) |
| X-Forwarded-Host | Original Host header |
| X-Request-ID | Unique request identifier |

### 8.5 Connection Draining

Graceful shutdown without dropping active connections:

```c
volatile sig_atomic_t g_draining = 0;

void handle_sigterm(int sig) {
    g_draining = 1;  // Stop accepting new connections
}

void graceful_shutdown(LoadBalancer *lb, ConnectionPool *pool) {
    printf("Starting graceful shutdown...\n");

    // Stop accepting new connections
    close(lb->server_fd);

    // Wait for active connections to complete (with timeout)
    time_t deadline = time(NULL) + 30;  // 30 second drain

    while (pool->num_connections > 0 && time(NULL) < deadline) {
        // Continue processing existing connections
        process_connections(pool);
        usleep(100000);  // 100ms
    }

    // Force close remaining connections
    for (int i = 0; i < MAX_CLIENTS; i++) {
        if (pool->connections[i].client_fd >= 0) {
            free_connection(pool, &pool->connections[i]);
        }
    }

    printf("Shutdown complete\n");
}
```

### 8.6 Multiple Algorithm Support

```c
typedef enum {
    ALG_ROUND_ROBIN,
    ALG_WEIGHTED_ROUND_ROBIN,
    ALG_LEAST_CONNECTIONS,
    ALG_IP_HASH
} Algorithm;

Backend* select_backend(LoadBalancer *lb, const char *client_ip) {
    switch (lb->algorithm) {
        case ALG_ROUND_ROBIN:
            return select_round_robin(lb);
        case ALG_WEIGHTED_ROUND_ROBIN:
            return select_weighted_round_robin(lb);
        case ALG_LEAST_CONNECTIONS:
            return select_least_connections(lb);
        case ALG_IP_HASH:
            return select_ip_hash(lb, client_ip);
        default:
            return select_round_robin(lb);
    }
}
```

### 8.7 Command Line Interface

```c
void print_usage(const char *prog) {
    fprintf(stderr, "Usage: %s <port> [backend:port[:weight]]... [-a algorithm]\n", prog);
    fprintf(stderr, "\nAlgorithms:\n");
    fprintf(stderr, "  rr      Round-robin (default)\n");
    fprintf(stderr, "  wrr     Weighted round-robin\n");
    fprintf(stderr, "  lc      Least connections\n");
    fprintf(stderr, "  iphash  IP hash (sticky sessions)\n");
    fprintf(stderr, "\nExamples:\n");
    fprintf(stderr, "  %s 8080 127.0.0.1:9001 127.0.0.1:9002\n", prog);
    fprintf(stderr, "  %s 8080 127.0.0.1:9001:3 127.0.0.1:9002:1 -a wrr\n", prog);
}

int parse_backend(const char *str, Backend *b) {
    // Format: host:port[:weight]
    char *copy = strdup(str);
    char *host = strtok(copy, ":");
    char *port = strtok(NULL, ":");
    char *weight = strtok(NULL, ":");

    if (!host || !port) {
        free(copy);
        return -1;
    }

    strncpy(b->host, host, sizeof(b->host) - 1);
    b->port = atoi(port);
    b->weight = weight ? atoi(weight) : 1;
    b->is_healthy = 1;
    b->active_connections = 0;

    free(copy);
    return 0;
}
```

### 8.8 Performance Considerations

| Factor | Impact | Optimization |
|--------|--------|--------------|
| Buffer Size | Memory vs syscalls | 8KB-64KB typical |
| Max Connections | Memory usage | Set based on RAM |
| Health Check Interval | Detection speed vs load | 5-30 seconds |
| select() vs epoll | Scalability | epoll for >1000 connections |
| Timeout Values | Resource cleanup | 30-60 seconds |

### 8.9 Complete Advanced LB Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    ADVANCED LOAD BALANCER                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │                    Main Event Loop                        │ │
│  │                                                           │ │
│  │  ┌─────────────┐   ┌─────────────┐   ┌─────────────┐     │ │
│  │  │   select()  │   │  Health     │   │  Statistics │     │ │
│  │  │  Monitoring │   │  Checker    │   │  Collector  │     │ │
│  │  └──────┬──────┘   └──────┬──────┘   └──────┬──────┘     │ │
│  │         │                 │                 │             │ │
│  │         ↓                 ↓                 ↓             │ │
│  │  ┌──────────────────────────────────────────────────┐    │ │
│  │  │              Event Dispatcher                    │    │ │
│  │  └──────────────────────────────────────────────────┘    │ │
│  │         │                 │                 │             │ │
│  │         ↓                 ↓                 ↓             │ │
│  │  ┌───────────┐    ┌───────────┐    ┌───────────┐        │ │
│  │  │  Accept   │    │  Relay    │    │ Algorithm │        │ │
│  │  │  Handler  │    │  Handler  │    │ Selector  │        │ │
│  │  └───────────┘    └───────────┘    └───────────┘        │ │
│  └───────────────────────────────────────────────────────────┘ │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │                    Backend Pool                           │ │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐     │ │
│  │  │Backend 1│  │Backend 2│  │Backend 3│  │Backend N│     │ │
│  │  │ w=3, UP │  │ w=2, UP │  │ w=1,DOWN│  │ w=1, UP │     │ │
│  │  └─────────┘  └─────────┘  └─────────┘  └─────────┘     │ │
│  └───────────────────────────────────────────────────────────┘ │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │                  Connection Pool                          │ │
│  │  [Conn 1][Conn 2][Conn 3]...[Conn 256]                   │ │
│  │  Active: 45    Max: 256    Peak: 128                     │ │
│  └───────────────────────────────────────────────────────────┘ │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 8.10 Lab Exercises

**Exercise 8.1:** Add connection timeout (close idle connections after 60 seconds).

**Exercise 8.2:** Implement request rate limiting per client IP.

**Exercise 8.3:** Add JSON API endpoint for statistics (`/lb-stats`).

**Exercise 8.4:** Implement backend weight hot-reloading via SIGHUP.

---

# Part III: Advanced Topics (Implemented)

> **Note:** Chapters 9-14 have been implemented with full source code in the corresponding chapter directories. See the README.md in each chapter folder for detailed implementation guides.

---

## Chapter 9: High-Performance I/O

> **Implementation:** `chapter-05-high-perf-io/` (event_loop.h, event_loop_epoll.c, event_loop_kqueue.c, event_loop_select.c, high_perf_lb.c)

### 9.1 I/O Multiplexing Comparison

| Method | Scalability | Complexity | Platform |
|--------|-------------|------------|----------|
| select() | O(n) | Low | POSIX |
| poll() | O(n) | Low | POSIX |
| epoll() | O(1) | Medium | Linux |
| kqueue() | O(1) | Medium | BSD/macOS |
| io_uring | O(1) | High | Linux 5.1+ |

### 9.2 epoll (Linux)

```c
#include <sys/epoll.h>

#define MAX_EVENTS 1024

int epoll_fd = epoll_create1(0);

// Add file descriptor
struct epoll_event ev;
ev.events = EPOLLIN | EPOLLET;  // Edge-triggered
ev.data.fd = client_fd;
epoll_ctl(epoll_fd, EPOLL_CTL_ADD, client_fd, &ev);

// Event loop
struct epoll_event events[MAX_EVENTS];
while (running) {
    int nfds = epoll_wait(epoll_fd, events, MAX_EVENTS, 1000);

    for (int i = 0; i < nfds; i++) {
        if (events[i].data.fd == server_fd) {
            // New connection
        } else {
            // Data on existing connection
        }
    }
}
```

### 9.3 kqueue (BSD/macOS)

```c
#include <sys/event.h>

int kq = kqueue();

// Add event
struct kevent change;
EV_SET(&change, client_fd, EVFILT_READ, EV_ADD | EV_ENABLE, 0, 0, NULL);
kevent(kq, &change, 1, NULL, 0, NULL);

// Event loop
struct kevent events[MAX_EVENTS];
while (running) {
    int nev = kevent(kq, NULL, 0, events, MAX_EVENTS, &timeout);

    for (int i = 0; i < nev; i++) {
        handle_event(&events[i]);
    }
}
```

### 9.4 io_uring (Linux 5.1+)

```c
#include <liburing.h>

struct io_uring ring;
io_uring_queue_init(256, &ring, 0);

// Submit read request
struct io_uring_sqe *sqe = io_uring_get_sqe(&ring);
io_uring_prep_read(sqe, fd, buffer, len, 0);
io_uring_sqe_set_data(sqe, user_data);
io_uring_submit(&ring);

// Get completion
struct io_uring_cqe *cqe;
io_uring_wait_cqe(&ring, &cqe);
// Process cqe->res
io_uring_cqe_seen(&ring, cqe);
```

### 9.5 Level-Triggered vs Edge-Triggered

```
Level-Triggered (Default):
- Event fires as long as condition is true
- Easier to program, may cause spurious wakeups
- select(), poll() are level-triggered

Edge-Triggered:
- Event fires only on state change
- More efficient, requires careful handling
- Must read until EAGAIN
- epoll EPOLLET, kqueue EV_CLEAR
```

### 9.6 Cross-Platform Implementation

The implementation provides a unified API across platforms:

```c
// event_loop.h - Cross-platform abstraction
typedef struct event_loop event_loop_t;
typedef void (*event_callback)(event_loop_t*, int fd, uint32_t events, void* data);

event_loop_t* event_loop_create(int max_events);
int event_loop_add(event_loop_t* loop, int fd, uint32_t events,
                   event_callback cb, void* data);
int event_loop_run(event_loop_t* loop, int timeout_ms);
void event_loop_destroy(event_loop_t* loop);
```

Build system automatically selects backend:
- Linux: epoll (O(1) scalability)
- macOS/BSD: kqueue (O(1) scalability)
- Others: select (portable fallback)

---

## Chapter 10: Connection Pooling

> **Implementation:** `chapter-06-connection-pooling/` (conn_pool.h, conn_pool.c, pooled_lb.c)

### 10.1 Why Connection Pooling?

```
WITHOUT POOLING (Nginx-style)
═══════════════════════════════════════════════════════════════

Request 1 → Connect → Request → Response → Close
Request 2 → Connect → Request → Response → Close
Request 3 → Connect → Request → Response → Close

TCP Handshakes: 3
Connection reuse: 0%


WITH POOLING (Pingora-style)
═══════════════════════════════════════════════════════════════

Request 1 → Connect ─┐
Request 2 ──────────→├─→ Reused Connection → Responses
Request 3 ──────────→│
                     │
            Pool maintains connections

TCP Handshakes: 1
Connection reuse: 99.92%
```

### 10.2 Pool API

```c
// Create pool for a backend
conn_pool_t* conn_pool_create(const char* host, int port,
                               int max_size, int ttl_seconds);

// Get connection (creates new if needed)
int conn_pool_get(conn_pool_t* pool);

// Return connection to pool
void conn_pool_return(conn_pool_t* pool, int fd, int healthy);

// Get statistics
void conn_pool_stats(conn_pool_t* pool, int* total, int* available,
                     int* in_use, double* hit_rate);
```

### 10.3 LRU Eviction

Connections are tracked with last-used timestamps. When pool is full, least-recently-used connections are evicted first.

---

## Chapter 11: Rate Limiting

> **Implementation:** `chapter-07-rate-limiting/` (rate_limiter.h, rate_limiter.c)

### 11.1 Token Bucket Algorithm

```
Bucket Capacity: 10 tokens (burst size)
Refill Rate: 1 token/second

Initial:      After 3 reqs:   After 10s idle:
┌──────────┐  ┌──────────┐   ┌──────────┐
│██████████│  │███████   │   │██████████│
└──────────┘  └──────────┘   └──────────┘
   10 tokens    7 tokens       10 tokens
```

### 11.2 Sliding Window Algorithm

More accurate than fixed windows, prevents burst at window boundaries:

```
Window = prev_count * (1 - elapsed/window) + curr_count
```

### 11.3 API

```c
rate_limiter_t* rate_limiter_create(RateLimitAlgorithm algorithm,
                                     double rate, double burst);

// Returns 1 if allowed, 0 if rate limited
int rate_limiter_allow(rate_limiter_t* limiter, const char* client_ip);
```

---

## Chapter 12: Metrics & Prometheus

> **Implementation:** `chapter-08-metrics/` (metrics.h, metrics.c)

### 12.1 Metric Types

| Type | Use Case | Example |
|------|----------|---------|
| Counter | Monotonically increasing | Total requests |
| Gauge | Can go up or down | Active connections |
| Histogram | Distribution | Request latency |

### 12.2 Prometheus Format

```
# HELP lb_requests_total Total HTTP requests
# TYPE lb_requests_total counter
lb_requests_total{backend="server1",status="200"} 12847

# HELP lb_request_duration_seconds Request latency
# TYPE lb_request_duration_seconds histogram
lb_request_duration_seconds_bucket{le="0.01"} 423
lb_request_duration_seconds_bucket{le="0.05"} 1847
lb_request_duration_seconds_bucket{le="+Inf"} 2156
```

### 12.3 API

```c
metrics_t* metrics_create(void);
void metrics_counter_inc(metrics_t* m, const char* name, const char* labels);
void metrics_gauge_set(metrics_t* m, const char* name, double value, const char* labels);
void metrics_histogram_observe(metrics_t* m, const char* name, double value, const char* labels);
void metrics_expose(metrics_t* m, int fd);  // Write to /metrics endpoint
```

---

## Chapter 13: Zero-Copy I/O

> **Implementation:** `chapter-09-zero-copy/` (zero_copy.h, zero_copy.c)

### 13.1 Traditional vs Zero-Copy

```
TRADITIONAL (4 copies)           ZERO-COPY (0 user-space copies)
Disk → Kernel Buffer             Disk ─────────────────────┐
Kernel → User Buffer                                       │
User → Kernel Buffer             Kernel Buffer ──→ Network │
Kernel → Network                        (DMA transfer)     │
```

### 13.2 Platform Support

| Platform | sendfile() | splice() |
|----------|------------|----------|
| Linux | Yes | Yes |
| macOS | Yes | No |
| FreeBSD | Yes | No |

### 13.3 API

```c
// File to socket (static files)
ssize_t zero_copy_file_to_socket(int socket_fd, int file_fd,
                                  off_t* offset, size_t count);

// Socket to socket (proxy relay, Linux only)
ssize_t zero_copy_socket_relay(int dest_fd, int src_fd, size_t count);
```

---

## Chapter 14: Hot Reload & Configuration

> **Implementation:** `chapter-10-hot-reload/` (config.h, config.c, lb.json)

### 14.1 Zero-Downtime Reload with SO_REUSEPORT

```
T=0: Admin sends SIGHUP
T=1: Fork new process with new config
T=2: Both processes accept (SO_REUSEPORT)
     Old: draining existing connections
     New: handling all new connections
T=3: Old process exits when drained
```

### 14.2 Configuration File Format

```json
{
  "listen_port": 8080,
  "algorithm": "weighted_round_robin",
  "backends": [
    {"host": "127.0.0.1", "port": 9001, "weight": 3}
  ],
  "pool": {"max_size": 64, "ttl": 60},
  "rate_limit": {"per_ip": 100.0, "burst": 20},
  "drain_timeout": 30
}
```

### 14.3 API

```c
config_t* config_load(const char* filename);
int config_validate(config_t* cfg);
int config_changed(config_t* cfg);  // For file watching
config_t* config_reload(const char* filename);
```

---

# Part IV: Future Curriculum

---

## Chapter 15: Security & TLS/SSL

### 15.1 TLS Termination

```
┌────────┐  HTTPS    ┌────────────┐  HTTP   ┌─────────┐
│ Client │──────────→│    Load    │────────→│ Backend │
│        │← TLS ────→│  Balancer  │ Plain   │ Server  │
└────────┘           └────────────┘         └─────────┘
                     TLS Termination
```

### 15.2 OpenSSL Integration (Future)

```c
#include <openssl/ssl.h>
#include <openssl/err.h>

// Initialize OpenSSL
SSL_library_init();
SSL_load_error_strings();
OpenSSL_add_all_algorithms();

// Create context
SSL_CTX *ctx = SSL_CTX_new(TLS_server_method());
SSL_CTX_use_certificate_file(ctx, "cert.pem", SSL_FILETYPE_PEM);
SSL_CTX_use_PrivateKey_file(ctx, "key.pem", SSL_FILETYPE_PEM);

// Wrap socket
SSL *ssl = SSL_new(ctx);
SSL_set_fd(ssl, client_fd);
SSL_accept(ssl);

// Read/Write
SSL_read(ssl, buffer, size);
SSL_write(ssl, buffer, size);

// Cleanup
SSL_shutdown(ssl);
SSL_free(ssl);
```

### 15.3 Topics to Cover

- Certificate management
- Perfect Forward Secrecy (PFS)
- OCSP stapling
- Client certificate authentication
- TLS 1.3 features
- SNI (Server Name Indication)
- HTTP/2 with ALPN
- Security headers (HSTS, CSP)

---

## Chapter 16: HTTP Protocol Deep Dive

### 16.1 HTTP/1.1 Features

```
Connection: keep-alive           # Persistent connections
Transfer-Encoding: chunked       # Streaming responses
Content-Encoding: gzip           # Compression
Cache-Control: max-age=3600      # Caching directives
```

### 16.2 HTTP/2 Features

```
┌──────────────────────────────────────────┐
│              HTTP/2 Connection           │
├──────────────────────────────────────────┤
│  Stream 1: GET /index.html               │
│  Stream 3: GET /style.css                │
│  Stream 5: GET /script.js                │
│  Stream 7: GET /image.png                │
│                                          │
│  (All streams multiplexed on one TCP)    │
└──────────────────────────────────────────┘
```

### 16.3 Topics to Cover

- HTTP parsing libraries (http-parser, llhttp)
- Request pipelining
- WebSocket upgrade
- Server-Sent Events (SSE)
- HTTP/2 HPACK compression
- HTTP/3 and QUIC
- gRPC load balancing

---

## Chapter 17: Distributed Systems Concepts

### 17.1 Service Discovery

```
┌─────────────────────────────────────────────────────────────┐
│                    SERVICE DISCOVERY                        │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Backend 1 ──register──→ ┌─────────────┐                   │
│  Backend 2 ──register──→ │  Discovery  │ ←──query── Load   │
│  Backend 3 ──register──→ │   Service   │            Balancer│
│                          └─────────────┘                   │
│                                                             │
│  Tools: Consul, etcd, Zookeeper, DNS SRV                   │
└─────────────────────────────────────────────────────────────┘
```

### 17.2 Consistent Hashing

```
                    Hash Ring (0 to 2^32)

                         ┌───────────────┐
                        ╱                 ╲
                       ╱    Node A (h=100)─┼── Keys 0-100
                      ╱                     ╲
                     ╱                       ╲
                    │                         │
           Keys     │                         │
          900-1000 ─┼─ Node D (h=1000)        │
                    │                         │
                     ╲                       ╱
                      ╲                     ╱
                       ╲   Node C (h=700)──┼── Keys 500-700
                        ╲                 ╱
                         └───────────────┘
                               │
                          Node B (h=500) ── Keys 100-500
```

### 17.3 Topics to Cover

- Circuit breaker pattern
- Retry with exponential backoff
- Bulkhead isolation
- Rate limiting algorithms (token bucket, leaky bucket)
- Leader election
- Consensus protocols (Raft, Paxos)
- CAP theorem implications

---

## Chapter 18: Production-Grade Features (Additional)

### 18.1 Additional Configuration Management

```yaml
# config.yaml
listen:
  port: 8080
  address: 0.0.0.0

backends:
  - host: 192.168.1.10
    port: 9000
    weight: 3
    max_connections: 100
  - host: 192.168.1.11
    port: 9000
    weight: 2
    max_connections: 100

health_check:
  interval: 10s
  timeout: 2s
  healthy_threshold: 2
  unhealthy_threshold: 3
  path: /health

algorithm: weighted_round_robin

timeouts:
  connect: 5s
  read: 30s
  write: 30s
```

### 18.2 Advanced Hot Reload Patterns

> **Note:** Basic hot reload is implemented in Chapter 14. This section covers advanced patterns.

```c
// Atomic configuration swap with grace period
void reload_config(LoadBalancer *lb) {
    Config *new_config = parse_config("config.yaml");

    if (!validate_config(new_config)) {
        free_config(new_config);
        return;
    }

    pthread_mutex_lock(&lb->lock);
    Config *old_config = lb->config;
    lb->config = new_config;
    pthread_mutex_unlock(&lb->lock);

    sleep(5);  // Grace period
    free_config(old_config);
}
```

### 18.3 Topics to Cover

- Configuration file formats (YAML, JSON, TOML)
- Command-line argument parsing
- Environment variable support
- Secrets management
- Logging frameworks (syslog, structured logging)
- Log rotation and aggregation
- Graceful restart
- Blue-green deployments
- Canary releases

---

## Chapter 19: Testing & Advanced Observability

> **Note:** Basic metrics and Prometheus integration is implemented in Chapter 12. This section covers advanced observability patterns.

### 19.1 Advanced Metrics Patterns

Building on the metrics implementation in Chapter 12:

```c
// Extended metrics with histograms
typedef struct {
    uint64_t requests_total;
    uint64_t requests_success;
    uint64_t requests_failed;
    uint64_t bytes_received;
    uint64_t bytes_sent;
    int32_t active_connections;
    int32_t healthy_backends;
    uint64_t latency_buckets[10];  // P50, P90, P99 calculation
} ExtendedMetrics;
```

### 19.2 Distributed Tracing

```
┌─────────────────────────────────────────────────────────────────┐
│ Trace ID: abc123                                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ Client ─────┬─────────────────────────────────────────────→     │
│             │                                                   │
│ Load        │  Span: lb-proxy                                   │
│ Balancer ───┼──┬──────────────────────────────────────→         │
│             │  │                                                │
│ Backend ────┼──┼──┬───────────────────────────────→             │
│             │  │  │                                             │
│ Database ───┼──┼──┼──┬────────────────────→                     │
│             │  │  │  │                                          │
│ Time ───────┴──┴──┴──┴─────────────────────────────────────→    │
│            0  10 20 50                              100ms       │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 19.3 Topics to Cover

- Unit testing strategies
- Integration testing
- Load testing (wrk, ab, locust)
- Chaos engineering
- Prometheus metrics format
- Grafana dashboards
- Jaeger/Zipkin tracing
- ELK stack integration
- Alerting strategies

---

# Part V: Appendices

---

## Appendix A: API Reference

### Socket Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| socket | `int socket(int domain, int type, int protocol)` | Create socket |
| bind | `int bind(int fd, struct sockaddr*, socklen_t)` | Assign address |
| listen | `int listen(int fd, int backlog)` | Mark passive |
| accept | `int accept(int fd, struct sockaddr*, socklen_t*)` | Accept connection |
| connect | `int connect(int fd, struct sockaddr*, socklen_t)` | Connect to server |
| read | `ssize_t read(int fd, void *buf, size_t count)` | Read data |
| write | `ssize_t write(int fd, void *buf, size_t count)` | Write data |
| close | `int close(int fd)` | Close socket |
| setsockopt | `int setsockopt(int fd, int level, int optname, ...)` | Set options |
| getsockopt | `int getsockopt(int fd, int level, int optname, ...)` | Get options |

### Address Functions

| Function | Description |
|----------|-------------|
| htons/htonl | Host to network byte order |
| ntohs/ntohl | Network to host byte order |
| inet_pton | String to binary IP address |
| inet_ntop | Binary to string IP address |
| getaddrinfo | Hostname resolution |
| freeaddrinfo | Free addrinfo results |

---

## Appendix B: Glossary

| Term | Definition |
|------|------------|
| **Backend** | Server that handles actual requests |
| **Connection Pool** | Pre-allocated connections for reuse |
| **Draining** | Gracefully stopping new connections |
| **Ephemeral Port** | Temporary port assigned by OS |
| **File Descriptor** | Integer handle for I/O resource |
| **Health Check** | Periodic backend availability test |
| **Keep-Alive** | Persistent TCP connection |
| **L4/L7** | OSI layer 4 (transport) / layer 7 (application) |
| **Latency** | Time from request to response |
| **Multiplexing** | Handling multiple I/O sources simultaneously |
| **Proxy** | Intermediary between client and server |
| **Round-Robin** | Sequential distribution algorithm |
| **Socket** | Endpoint for network communication |
| **Sticky Session** | Route same client to same backend |
| **TLS Termination** | Decrypt HTTPS at load balancer |
| **Upstream** | Backend servers (from proxy perspective) |

---

## Appendix C: Further Reading

### Books
- "UNIX Network Programming" by W. Richard Stevens
- "TCP/IP Illustrated" by W. Richard Stevens
- "The Linux Programming Interface" by Michael Kerrisk
- "High Performance Browser Networking" by Ilya Grigorik
- "Designing Data-Intensive Applications" by Martin Kleppmann

### Online Resources
- Beej's Guide to Network Programming: https://beej.us/guide/bgnet/
- Linux man pages: https://man7.org/linux/man-pages/
- Nginx Documentation: https://nginx.org/en/docs/
- HAProxy Documentation: https://www.haproxy.com/documentation/

### Related Projects
- HAProxy: http://www.haproxy.org/
- Nginx: https://nginx.org/
- Envoy: https://www.envoyproxy.io/
- Traefik: https://traefik.io/

---

## Appendix D: Lab Exercises

### Beginner Labs

**Lab 1: Echo Server Variations**
- Implement uppercase echo
- Add timestamp to each response
- Log client IP and message to file

**Lab 2: Simple Calculator Server**
- Accept "ADD 5 3" format
- Return calculation result
- Handle invalid input gracefully

**Lab 3: File Server**
- Serve files from a directory
- Handle 404 for missing files
- Add content-type headers

### Intermediate Labs

**Lab 4: Chat Server**
- Multiple clients
- Broadcast messages
- Private messaging

**Lab 5: Rate Limiter**
- Token bucket algorithm
- Per-IP tracking
- Return 429 when exceeded

**Lab 6: Caching Proxy**
- Cache HTTP responses
- TTL-based expiration
- Cache invalidation

### Advanced Labs

**Lab 7: HTTP/1.1 Parser**
- Parse headers correctly
- Handle chunked encoding
- Keep-alive connections

**Lab 8: Connection Pooling**
- Maintain backend connections
- Pool sizing and recycling
- Health-based eviction

**Lab 9: Full Production LB**
- TLS termination
- Multiple algorithms
- Prometheus metrics
- Configuration reload

---

## Course Assessment Rubric

### Knowledge Checks

| Chapter | Assessment Type | Passing Score |
|---------|-----------------|---------------|
| 5 | Implement TCP client/server | Working code |
| 6 | Build reverse proxy | Correct relay |
| 7 | Implement 2 LB algorithms | Both working |
| 8 | Build concurrent LB | 100+ simultaneous |

### Final Project Options

1. **Production Load Balancer**
   - Multiple algorithms
   - Health checking
   - Statistics API
   - Configuration file

2. **HTTP/2 Proxy**
   - HTTP/2 parsing
   - Stream multiplexing
   - Header compression

3. **Service Mesh Component**
   - Service discovery
   - Circuit breaker
   - Distributed tracing

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0 | 2024-01 | Initial curriculum |
| 1.1 | 2024-02 | Added advanced chapters |
| 2.0 | 2026-02 | Implemented Chapters 5-10: High-Performance I/O, Connection Pooling, Rate Limiting, Metrics, Zero-Copy, Hot Reload |
| 2.1 | 2026-XX | TBD - HTTP/2, TLS sections |

---

*This documentation is designed as both a reference manual and educational curriculum. Work through the chapters sequentially for best learning outcomes, or use the index to find specific topics.*
