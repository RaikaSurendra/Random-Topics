# Chapter 01: Fundamentals

## Learning Objectives

By the end of this chapter, you will:
- Understand what reverse proxies do and why they matter
- Know Pingora's architecture and core components
- Understand the request lifecycle in Pingora
- Be ready to write your first proxy

---

## 1.1 What is a Reverse Proxy?

### The Problem

Imagine you have a web application. Users connect directly to your server:

```
User → Your Server
```

This works, but has problems:
- **Single point of failure** - server dies, everything dies
- **No scaling** - one server handles all traffic
- **Exposed backend** - attackers see your real server
- **No flexibility** - can't do A/B testing, canary deploys

### The Solution: Reverse Proxy

Put a proxy between users and your servers:

```
User → Proxy → Server 1
              → Server 2
              → Server 3
```

Now you can:
- **Load balance** across multiple servers
- **Hide** your real infrastructure
- **Cache** common responses
- **Route** intelligently based on URL, headers, etc.

---

## 1.2 Forward Proxy vs Reverse Proxy

Don't confuse these!

### Forward Proxy (Client-side)
```
[Your Computer] → [Forward Proxy] → [Internet]
```
- Sits in front of **clients**
- Example: Corporate proxy, VPN
- Client knows it's using a proxy

### Reverse Proxy (Server-side)
```
[Internet] → [Reverse Proxy] → [Your Servers]
```
- Sits in front of **servers**
- Example: Nginx, Cloudflare, **Pingora**
- Client doesn't know there's a proxy

**Pingora is a reverse proxy framework.**

---

## 1.3 Pingora Architecture

### Core Components

```
┌─────────────────────────────────────────────────────────┐
│                        Server                           │
│  ┌─────────────────────────────────────────────────┐   │
│  │                    Service                       │   │
│  │  ┌─────────────┐  ┌─────────────┐              │   │
│  │  │   Listener  │  │   Listener  │  ...         │   │
│  │  │  (TCP/TLS)  │  │  (TCP/TLS)  │              │   │
│  │  └─────────────┘  └─────────────┘              │   │
│  │         │                │                      │   │
│  │         ▼                ▼                      │   │
│  │  ┌─────────────────────────────────────────┐   │   │
│  │  │           Your ProxyHttp impl           │   │   │
│  │  │  (upstream_peer, filters, logging...)   │   │   │
│  │  └─────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────┘   │
│                                                         │
│  ┌─────────────────────────────────────────────────┐   │
│  │              Connection Pool                     │   │
│  │    (reuses connections to upstreams)            │   │
│  └─────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

### Key Types

| Type | Purpose |
|------|---------|
| `Server` | The runtime - manages services, signals, lifecycle |
| `Service` | A network service - listens and handles connections |
| `ProxyHttp` | **The trait you implement** - defines proxy behavior |
| `Session` | A single client connection/request |
| `HttpPeer` | A connection to an upstream server |

---

## 1.4 The ProxyHttp Trait

This is the heart of Pingora. You implement this trait to define your proxy's behavior.

```rust
#[async_trait]
pub trait ProxyHttp {
    /// Your custom per-request context type
    type CTX;

    /// Create context for each new request
    fn new_ctx(&self) -> Self::CTX;

    /// REQUIRED: Choose which upstream to connect to
    async fn upstream_peer(
        &self,
        session: &mut Session,
        ctx: &mut Self::CTX,
    ) -> Result<Box<HttpPeer>>;

    // Optional hooks (we'll cover these in later chapters):
    // - request_filter()
    // - upstream_request_filter()
    // - upstream_response_filter()
    // - response_filter()
    // - logging()
    // ... and more
}
```

### The Context (CTX)

Each request gets its own context. Use it to store:
- Request ID for tracing
- Start time for latency measurement
- User info from authentication
- Any per-request state

---

## 1.5 Request Lifecycle

When a request arrives, Pingora calls your methods in this order:

```
1. new_ctx()
   └─▶ Create fresh context for this request

2. request_filter()
   └─▶ First look at the incoming request
   └─▶ Can reject early (auth failed, rate limited)

3. upstream_peer()
   └─▶ Decide which backend server to use
   └─▶ REQUIRED - you must implement this

4. upstream_request_filter()
   └─▶ Modify request before sending to backend
   └─▶ Add headers, rewrite paths, etc.

5. [Pingora sends request to upstream]

6. upstream_response_filter()
   └─▶ First look at upstream's response
   └─▶ Can modify or even retry with different backend

7. response_filter()
   └─▶ Final modifications to response
   └─▶ Add security headers, etc.

8. [Pingora sends response to client]

9. logging()
   └─▶ Request complete - log metrics, stats
```

---

## 1.6 Understanding Async in Pingora

Pingora is fully async, using Tokio. This means:

### Why Async?
- Handle thousands of connections with few threads
- No blocking while waiting for network I/O
- Efficient resource usage

### Key Points:
```rust
// Methods are async
async fn upstream_peer(...) -> Result<...> {
    // You can await other async operations here
    let result = some_async_operation().await;
    // ...
}

// The trait requires async_trait macro
use async_trait::async_trait;

#[async_trait]
impl ProxyHttp for MyProxy {
    // ...
}
```

---

## 1.7 Error Handling

Pingora uses its own `Error` type and `Result`:

```rust
use pingora::prelude::*;  // Brings in Result, Error

// Return Ok for success
async fn upstream_peer(...) -> Result<Box<HttpPeer>> {
    Ok(Box::new(HttpPeer::new(...)))
}

// Return Err to fail the request
async fn request_filter(...) -> Result<bool> {
    if !authorized {
        return Err(Error::new(ErrorType::HTTPStatus(403)));
    }
    Ok(false)  // false = don't skip remaining filters
}
```

---

## 1.8 Pingora Crates

Pingora is modular. Key crates:

| Crate | Purpose |
|-------|---------|
| `pingora` | Meta-crate, re-exports everything |
| `pingora-core` | Server, connections, protocols |
| `pingora-proxy` | HTTP proxy trait and implementation |
| `pingora-cache` | Caching layer |
| `pingora-load-balancing` | Load balancing algorithms |
| `pingora-timeout` | Timeout utilities |

For most use cases, just use:
```toml
[dependencies]
pingora = { version = "0.7", features = ["proxy"] }
```

---

## Exercises

### Exercise 1.1: Concept Check
Answer these questions (answers at bottom):

1. What's the difference between forward and reverse proxy?
2. What method MUST you implement in ProxyHttp?
3. What is the CTX type used for?
4. In what order are `request_filter` and `upstream_peer` called?

### Exercise 1.2: Draw It Out
On paper, draw the path of an HTTP request through a Pingora proxy to a backend and back. Label each ProxyHttp method that gets called.

### Exercise 1.3: Research
Read the official Pingora blog post: https://blog.cloudflare.com/pingora-open-source/

Note down:
- How much memory did Pingora save compared to their old proxy?
- How much did connection reuse improve?

---

## Key Takeaways

1. **Pingora is a framework** - you write Rust code, not config files
2. **ProxyHttp is the core trait** - implement it to define behavior
3. **Request lifecycle has clear phases** - filter, peer selection, more filters, logging
4. **Context (CTX) tracks per-request state** - use it for tracing, timing, etc.
5. **Everything is async** - efficient handling of many connections

---

## Exercise Answers

### 1.1 Answers:
1. Forward proxy sits in front of clients; reverse proxy sits in front of servers
2. `upstream_peer()` - it's the only required method
3. To store per-request state (request ID, timing, user data, etc.)
4. `request_filter` is called first, then `upstream_peer`

---

## Next Chapter

Ready to write code? Continue to [Chapter 02: Your First Proxy](../02-first-proxy/README.md)
