# C-Based Load Balancer - College Curriculum & Learning Roadmap

> **Computer Networks & Systems Programming Course**
> Duration: 16 Weeks (One Semester)
> Credits: 4

---

## Course Information

**Course Code:** CS-445 Network Systems Programming
**Prerequisites:** CS-201 Data Structures, CS-220 Operating Systems Basics
**Lab Hours:** 4 hours/week
**Lecture Hours:** 3 hours/week

---

## Learning Paths

### Path A: Full Semester Course (16 weeks)
For computer science students with programming background.

### Path B: Accelerated (8 weeks)
For experienced developers learning systems programming.

### Path C: Self-Paced Online
Estimated 80-120 hours total.

---

## Weekly Schedule

### MODULE 1: FOUNDATIONS (Weeks 1-4)

---

#### Week 1: Course Introduction & Networking Basics

**Lecture Topics:**
- Introduction to load balancers and their role in modern infrastructure
- OSI model review (emphasis on Layers 4 and 7)
- TCP/IP protocol stack
- Client-server architecture

**Lab Work:**
- Environment setup (gcc, make, debugging tools)
- Install and explore Wireshark
- Capture and analyze TCP handshake

**Reading:**
- DOCUMENTATION.md: Chapters 1-3
- Beej's Guide: Introduction

**Deliverable:**
- Lab report: TCP handshake analysis

---

#### Week 2: C Programming Review & System Calls

**Lecture Topics:**
- Pointers and memory management review
- File descriptors in Unix
- System calls vs library functions
- Error handling with errno

**Lab Work:**
- Implement file copy using read()/write()
- Trace system calls with strace/dtrace
- Memory leak detection with valgrind

**Reading:**
- The Linux Programming Interface: Chapters 1-4

**Deliverable:**
- Working file copy utility

---

#### Week 3: Socket Programming Basics

**Lecture Topics:**
- Berkeley sockets API
- Address structures (sockaddr_in)
- Byte ordering (htons, htonl)
- socket(), bind(), listen(), accept()

**Lab Work:**
- Build echo server (chapter-01)
- Build echo client (chapter-01)
- Test with netcat

**Reading:**
- DOCUMENTATION.md: Chapter 5
- Beej's Guide: Socket basics

**Deliverable:**
- Working echo server/client pair

**Code Checkpoint:**
```c
// Students must understand:
int fd = socket(AF_INET, SOCK_STREAM, 0);
struct sockaddr_in addr;
addr.sin_family = AF_INET;
addr.sin_port = htons(8080);
addr.sin_addr.s_addr = INADDR_ANY;
```

---

#### Week 4: Client-Side Sockets & DNS

**Lecture Topics:**
- connect() system call
- getaddrinfo() for DNS resolution
- Error handling in network code
- Connection timeouts

**Lab Work:**
- Build HTTP client that fetches web pages
- Implement DNS lookup utility
- Handle connection failures gracefully

**Reading:**
- RFC 793 (TCP) overview
- getaddrinfo man page

**Deliverable:**
- HTTP GET client

**Assessment:** Quiz 1 - Socket API basics (10%)

---

### MODULE 2: PROXY IMPLEMENTATION (Weeks 5-7)

---

#### Week 5: I/O Multiplexing with select()

**Lecture Topics:**
- Blocking vs non-blocking I/O
- The select() system call
- fd_set manipulation macros
- Timeout handling

**Lab Work:**
- Modify echo server to handle multiple clients
- Implement chat server with select()
- Compare blocking vs multiplexed approaches

**Reading:**
- DOCUMENTATION.md: Chapter 4.5
- select() man page

**Deliverable:**
- Multi-client echo server

**Code Checkpoint:**
```c
fd_set read_fds;
FD_ZERO(&read_fds);
FD_SET(fd1, &read_fds);
FD_SET(fd2, &read_fds);
select(max_fd + 1, &read_fds, NULL, NULL, &timeout);
```

---

#### Week 6: Reverse Proxy Implementation

**Lecture Topics:**
- Forward vs reverse proxy concepts
- Bidirectional data relay
- HTTP protocol basics
- Error responses (502, 503, 504)

**Lab Work:**
- Build reverse proxy (chapter-02)
- Add request logging
- Implement error handling

**Reading:**
- DOCUMENTATION.md: Chapter 6
- HTTP/1.1 RFC 2616 overview

**Deliverable:**
- Working reverse proxy with logging

---

#### Week 7: Forward Proxy & HTTP Tunneling

**Lecture Topics:**
- HTTP CONNECT method
- HTTPS tunneling
- Man-in-the-middle considerations
- Proxy authentication (theory)

**Lab Work:**
- Build forward proxy (chapter-02)
- Test with browser proxy settings
- Implement CONNECT tunneling

**Reading:**
- RFC 7231 (HTTP semantics)
- CONNECT method specification

**Deliverable:**
- Working forward proxy with HTTPS support

**Assessment:** Midterm Project Part 1 - Complete proxy implementation (15%)

---

### MODULE 3: LOAD BALANCING CORE (Weeks 8-11)

---

#### Week 8: Load Balancing Fundamentals

**Lecture Topics:**
- Why load balancing matters
- Layer 4 vs Layer 7 load balancing
- Backend pools and health states
- Round-robin algorithm

**Lab Work:**
- Build basic load balancer (chapter-03)
- Test with multiple backends
- Observe traffic distribution

**Reading:**
- DOCUMENTATION.md: Chapter 7
- HAProxy introduction docs

**Deliverable:**
- Basic round-robin load balancer

**Code Checkpoint:**
```c
Backend* select_round_robin(LoadBalancer *lb) {
    int start = lb->current_index;
    do {
        lb->current_index = (lb->current_index + 1) % lb->num_backends;
        if (lb->backends[lb->current_index].is_healthy)
            return &lb->backends[lb->current_index];
    } while (lb->current_index != start);
    return NULL;
}
```

---

#### Week 9: Advanced Load Balancing Algorithms

**Lecture Topics:**
- Weighted round-robin
- Least connections
- IP hash (session persistence)
- Algorithm comparison and use cases

**Lab Work:**
- Implement weighted round-robin
- Implement least connections
- Implement IP hash
- Compare behavior under load

**Reading:**
- DOCUMENTATION.md: Chapter 7.3
- Nginx upstream documentation

**Deliverable:**
- Load balancer with 4 algorithm options

---

#### Week 10: Health Checking & Resilience

**Lecture Topics:**
- Active vs passive health checks
- TCP vs HTTP health checks
- Failure detection and recovery
- Backend warm-up strategies

**Lab Work:**
- Add TCP health checking to LB
- Implement failure detection
- Add backend recovery logic
- Test failure scenarios

**Reading:**
- HAProxy health check documentation
- Circuit breaker pattern

**Deliverable:**
- Health-checking load balancer

---

#### Week 11: Statistics & Monitoring

**Lecture Topics:**
- Metrics collection strategies
- Request counting and timing
- Bytes in/out tracking
- Signal handling for statistics

**Lab Work:**
- Add statistics tracking
- Implement SIGUSR1 stats printing
- Create JSON statistics endpoint
- Graph metrics with gnuplot

**Reading:**
- Prometheus metrics format
- StatsD protocol

**Deliverable:**
- Load balancer with statistics API

**Assessment:** Midterm Project Part 2 - Full load balancer (20%)

---

### MODULE 4: ADVANCED TOPICS (Weeks 12-15)

---

#### Week 12: Concurrent Connection Handling

**Lecture Topics:**
- Connection pools and tracking
- Non-blocking connection setup
- Event-driven architecture
- Memory management for connections

**Lab Work:**
- Build advanced LB (chapter-04)
- Handle 100+ concurrent connections
- Profile memory usage
- Stress testing with wrk

**Reading:**
- DOCUMENTATION.md: Chapter 8
- C10K problem paper

**Deliverable:**
- Concurrent load balancer (100+ connections)

---

#### Week 13: HTTP Features & Header Injection

**Lecture Topics:**
- HTTP header parsing
- X-Forwarded-For and X-Real-IP
- Keep-alive connections
- Content-Length vs chunked encoding

**Lab Work:**
- Add header injection
- Parse HTTP requests properly
- Handle keep-alive
- Test with real applications

**Reading:**
- HTTP headers reference
- Reverse proxy headers guide

**Deliverable:**
- Load balancer with proper HTTP handling

---

#### Week 14: Production Considerations

**Lecture Topics:**
- Configuration file formats
- Graceful shutdown and draining
- Logging best practices
- Security considerations

**Lab Work:**
- Add YAML configuration
- Implement graceful shutdown
- Add structured logging
- Security hardening

**Reading:**
- DOCUMENTATION.md: Chapter 13
- 12-factor app methodology

**Deliverable:**
- Production-ready configuration

---

#### Week 15: Performance Optimization

**Lecture Topics:**
- epoll vs select performance
- Buffer sizing strategies
- Zero-copy techniques
- Profiling and benchmarking

**Lab Work:**
- Convert to epoll (Linux) or kqueue (macOS)
- Benchmark with different buffer sizes
- Profile with perf/dtrace
- Document performance results

**Reading:**
- DOCUMENTATION.md: Chapter 9
- epoll/kqueue documentation

**Deliverable:**
- Performance comparison report

**Assessment:** Quiz 2 - Advanced topics (10%)

---

### MODULE 5: FINAL PROJECT (Week 16)

---

#### Week 16: Final Project & Presentations

**Final Project Options:**

**Option A: Production Load Balancer**
- All 4 algorithms
- HTTP and TCP health checks
- YAML configuration
- Prometheus metrics endpoint
- Graceful shutdown
- Documentation

**Option B: HTTP/2 Aware Proxy**
- HTTP/2 frame parsing
- Stream multiplexing
- Header compression (HPACK)
- Fallback to HTTP/1.1

**Option C: Service Mesh Component**
- Service discovery integration
- Circuit breaker implementation
- Distributed tracing headers
- Rate limiting

**Deliverables:**
- Source code with Makefile
- README with usage instructions
- Design document
- Live demonstration

**Assessment:** Final Project (35%)

---

## Assessment Summary

| Component | Weight | Description |
|-----------|--------|-------------|
| Quiz 1 | 10% | Socket API basics (Week 4) |
| Quiz 2 | 10% | Advanced topics (Week 15) |
| Midterm Part 1 | 15% | Proxy implementation (Week 7) |
| Midterm Part 2 | 20% | Load balancer (Week 11) |
| Final Project | 35% | Complete system (Week 16) |
| Lab Participation | 10% | Weekly lab attendance and completion |

---

## Grading Scale

| Grade | Percentage | Description |
|-------|------------|-------------|
| A | 90-100% | Exceptional understanding and implementation |
| B | 80-89% | Strong grasp of concepts, minor issues |
| C | 70-79% | Adequate understanding, some gaps |
| D | 60-69% | Below expectations, significant gaps |
| F | <60% | Insufficient mastery |

---

## Learning Objectives Mapping

| Week | Chapter | Learning Objectives |
|------|---------|---------------------|
| 1-2 | 3-4 | Understand networking fundamentals and OS concepts |
| 3-4 | 5 | Master socket API and client-server patterns |
| 5-7 | 6 | Implement proxy architectures |
| 8-11 | 7 | Understand and implement LB algorithms |
| 12-15 | 8-9 | Build production-quality systems |
| 16 | - | Synthesize knowledge in complete project |

---

## Required Materials

### Textbooks
1. **Primary:** This documentation (DOCUMENTATION.md)
2. **Reference:** "UNIX Network Programming, Vol 1" - Stevens

### Software
- GCC 9+ or Clang 10+
- GNU Make
- Wireshark
- curl, netcat
- wrk (load testing)
- valgrind (memory analysis)

### Hardware
- Linux/macOS system (VM acceptable)
- 4GB+ RAM recommended
- Network access

---

## Skills Progression Matrix

```
                    Weeks
Skill               1  2  3  4  5  6  7  8  9  10 11 12 13 14 15 16
─────────────────────────────────────────────────────────────────────
C Programming       ■■ ■■ ■■ ■■ ■■ ■■ ■■ ■■ ■■ ■■ ■■ ■■ ■■ ■■ ■■ ■■
Networking Basics   ■■ ■■ ■■ ■■
Socket API             ■■ ■■ ■■ ■■ ■■ ■■ ■■ ■■ ■■ ■■ ■■ ■■ ■■ ■■ ■■
I/O Multiplexing             ■■ ■■ ■■ ■■ ■■ ■■ ■■ ■■ ■■ ■■ ■■ ■■ ■■
HTTP Protocol                   ■■ ■■ ■■ ■■ ■■ ■■ ■■ ■■ ■■ ■■ ■■ ■■
Proxy Patterns                  ■■ ■■ ■■
LB Algorithms                            ■■ ■■ ■■ ■■ ■■ ■■ ■■ ■■ ■■
Health Checking                                ■■ ■■ ■■ ■■ ■■ ■■ ■■
Event-Driven Arch                                    ■■ ■■ ■■ ■■ ■■
Performance Tuning                                         ■■ ■■ ■■
Production Systems                                            ■■ ■■

■■ = Active learning/application
```

---

## Self-Study Checkpoints

Use these checkpoints to verify understanding:

### After Week 4 (Socket Basics)
- [ ] Can explain TCP three-way handshake
- [ ] Can create socket, bind, listen, accept
- [ ] Can connect to server and send/receive data
- [ ] Understands byte ordering

### After Week 7 (Proxies)
- [ ] Can explain forward vs reverse proxy
- [ ] Can implement bidirectional data relay
- [ ] Understands select() for multiplexing
- [ ] Can handle HTTP requests

### After Week 11 (Load Balancing)
- [ ] Can implement round-robin
- [ ] Can implement weighted round-robin
- [ ] Can implement least connections
- [ ] Can implement IP hash
- [ ] Can implement health checking

### After Week 15 (Advanced)
- [ ] Can handle concurrent connections
- [ ] Can optimize for performance
- [ ] Understands production considerations
- [ ] Can profile and benchmark

---

## Common Mistakes to Avoid

| Mistake | Solution |
|---------|----------|
| Forgetting htons() | Always convert port numbers |
| Not checking return values | Every syscall can fail |
| Memory leaks | Free what you allocate |
| Ignoring SIGPIPE | Handle or ignore explicitly |
| Blocking on single client | Use select()/poll()/epoll() |
| Hardcoding buffer sizes | Define constants |
| Not closing file descriptors | Track and close all FDs |

---

## Future Course Extensions

### CS-446: Distributed Systems (Follow-up course)
- Service mesh architecture
- Consensus protocols
- Distributed tracing
- Container orchestration

### CS-447: Network Security (Elective)
- TLS/SSL implementation
- Certificate management
- Security auditing
- Penetration testing

### CS-448: High-Performance Computing (Elective)
- DPDK and kernel bypass
- io_uring deep dive
- RDMA networking
- Lock-free data structures

---

## Industry Relevance

Skills learned in this course directly apply to:

| Role | Application |
|------|-------------|
| Backend Engineer | Building scalable services |
| Site Reliability Engineer | Understanding infrastructure |
| DevOps Engineer | Configuring load balancers |
| Network Engineer | Protocol understanding |
| Security Engineer | Network security analysis |

### Industry Tools Using These Concepts
- **HAProxy** - High-performance TCP/HTTP LB
- **Nginx** - Web server and reverse proxy
- **Envoy** - Modern L7 proxy
- **Traefik** - Cloud-native LB
- **Kubernetes** - Container orchestration networking

---

## Feedback and Iteration

This curriculum is designed to evolve. After completing the course, consider:

1. What topics needed more depth?
2. Which labs were most valuable?
3. What real-world scenarios were missing?
4. How could the pacing be improved?

---

*Last Updated: 2024*
*Version: 1.0*
