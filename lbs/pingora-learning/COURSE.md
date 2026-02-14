# Pingora Mastery Course

## Welcome, Future Proxy Engineer!

This course will take you from zero to building production-grade HTTP proxies with Cloudflare's Pingora framework. Each chapter builds on the previous one, with theory, code examples, and hands-on exercises.

---

## Course Overview

| Chapter | Topic | What You'll Build |
|---------|-------|-------------------|
| 01 | Fundamentals | Understanding proxies and Pingora architecture |
| 02 | First Proxy | A working reverse proxy from scratch |
| 03 | Request/Response | Header manipulation, routing, rewriting |
| 04 | Load Balancing | Round-robin, weighted, least connections |
| 05 | Health Checks | Active/passive health monitoring |
| 06 | Caching | Response caching with Pingora |
| 07 | Observability | Metrics, logging, tracing |
| 08 | Advanced | TLS, graceful reload, performance tuning |

---

## How to Use This Course

### For Each Chapter:

1. **Read** the `README.md` - understand the concepts
2. **Study** the code in `src/main.rs` - read the comments carefully
3. **Run** the example - see it work
4. **Complete** the exercises - apply what you learned
5. **Check** your solutions against provided answers

### Running Chapter Examples:

```bash
# Navigate to a chapter
cd chapters/02-first-proxy

# Run the example
cargo run

# Or with Docker (from project root)
docker compose -f chapters/02-first-proxy/docker-compose.yml up --build
```

---

## Prerequisites Checklist

Before starting, ensure you understand:

- [ ] **Rust Basics**: Variables, functions, structs, enums
- [ ] **Ownership**: Borrowing, references, lifetimes
- [ ] **Async Rust**: `async`/`await`, `Future` trait basics
- [ ] **Traits**: Implementing traits, trait bounds
- [ ] **HTTP**: Request/response cycle, headers, methods, status codes
- [ ] **Networking**: TCP, ports, DNS, TLS basics

### Quick Rust Refresher Resources:
- [Rust Book](https://doc.rust-lang.org/book/) - Chapters 1-10, 16-17
- [Async Book](https://rust-lang.github.io/async-book/)
- [Tokio Tutorial](https://tokio.rs/tokio/tutorial)

---

## The Big Picture: What is a Proxy?

```
┌──────────┐         ┌─────────────┐         ┌──────────────┐
│  Client  │ ──────▶ │   PROXY     │ ──────▶ │   Upstream   │
│ (browser)│ ◀────── │  (Pingora)  │ ◀────── │   (backend)  │
└──────────┘         └─────────────┘         └──────────────┘
     │                      │                       │
     │    HTTP Request      │    HTTP Request       │
     │ ──────────────────▶  │ ──────────────────▶   │
     │                      │                       │
     │   HTTP Response      │   HTTP Response       │
     │ ◀──────────────────  │ ◀──────────────────   │
```

A **reverse proxy** sits between clients and your backend servers. It can:

- **Route** requests to different backends
- **Load balance** across multiple servers
- **Cache** responses to reduce backend load
- **Terminate TLS** to handle HTTPS
- **Modify** requests and responses
- **Protect** backends from direct exposure

---

## Why Pingora?

| Feature | Benefit |
|---------|---------|
| **Rust** | Memory safety, no garbage collection pauses |
| **Async** | Handle millions of concurrent connections |
| **Programmable** | Full control via Rust code, not just config |
| **Battle-tested** | Powers Cloudflare's 1+ trillion requests/day |
| **Modular** | Use only what you need |

---

## Let's Begin!

Start with [Chapter 01: Fundamentals](./chapters/01-fundamentals/README.md)

---

## Quick Reference

### Key Pingora Types

```rust
Server              // The main runtime
Service             // A running network service
ProxyHttp           // Trait you implement for HTTP proxying
Session             // Represents a client connection
HttpPeer            // Represents an upstream connection
RequestHeader       // HTTP request headers
ResponseHeader      // HTTP response headers
```

### ProxyHttp Lifecycle

```
Client Request Arrives
         │
         ▼
    new_ctx()                    // Create per-request context
         │
         ▼
    request_filter()             // Inspect/modify incoming request
         │
         ▼
    upstream_peer()              // Choose which backend to use
         │
         ▼
    upstream_request_filter()    // Modify request before sending upstream
         │
         ▼
    [Request sent to upstream]
         │
         ▼
    upstream_response_filter()   // Process upstream response headers
         │
         ▼
    response_filter()            // Final response modifications
         │
         ▼
    [Response sent to client]
         │
         ▼
    logging()                    // Log the completed request
```

Good luck on your journey! 🚀
