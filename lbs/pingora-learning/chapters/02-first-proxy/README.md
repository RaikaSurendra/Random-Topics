# Chapter 02: Your First Proxy

## Learning Objectives

By the end of this chapter, you will:
- Create a working reverse proxy from scratch
- Understand each line of Pingora proxy code
- Run and test your proxy
- Debug common issues

---

## 2.1 Project Setup

Let's build a minimal reverse proxy step by step.

### Create the Project

```bash
cd chapters/02-first-proxy
cargo init
```

### Configure Dependencies

```toml
# Cargo.toml
[package]
name = "first-proxy"
version = "0.1.0"
edition = "2021"

[dependencies]
pingora = { version = "0.7", features = ["proxy"] }
async-trait = "0.1"
tokio = { version = "1", features = ["full"] }
```

---

## 2.2 The Complete Code

Here's our first proxy. Read every comment carefully!

```rust
// src/main.rs

// === IMPORTS ===
// async_trait lets us use async functions in traits
use async_trait::async_trait;

// pingora::prelude brings in common types:
// - Server: the runtime
// - HttpPeer: upstream connection
// - RequestHeader/ResponseHeader: HTTP headers
// - Result, Error: error handling
use pingora::prelude::*;

// These are specifically for HTTP proxying
use pingora::proxy::{http_proxy_service, ProxyHttp, Session};

// === OUR PROXY STRUCT ===
// This holds configuration for our proxy.
// It will be shared across all requests (hence no per-request state here).
pub struct FirstProxy {
    // Where to forward requests
    upstream: String,
}

impl FirstProxy {
    pub fn new(upstream: &str) -> Self {
        Self {
            upstream: upstream.to_string(),
        }
    }
}

// === PER-REQUEST CONTEXT ===
// Each request gets its own context instance.
// For now, it's empty - we'll add fields in later chapters.
pub struct RequestCtx {
    // We'll add: request_id, start_time, etc.
}

// === THE PROXYHTTP IMPLEMENTATION ===
// This is where the magic happens!
#[async_trait]
impl ProxyHttp for FirstProxy {
    // Tell Pingora what our context type is
    type CTX = RequestCtx;

    // Called for each new request - create a fresh context
    fn new_ctx(&self) -> Self::CTX {
        RequestCtx {}
    }

    // REQUIRED: Tell Pingora which upstream to use
    // This is called after request_filter (if any)
    async fn upstream_peer(
        &self,
        _session: &mut Session,  // Access to request info
        _ctx: &mut Self::CTX,    // Our context
    ) -> Result<Box<HttpPeer>> {
        // Parse the address
        // Note: For DNS names (like "backend:80"), use ToSocketAddrs
        let addr: std::net::SocketAddr = self
            .upstream
            .parse()
            .expect("Invalid upstream address");

        // Create the peer
        // Arguments:
        //   addr: where to connect
        //   tls: false = plain HTTP, true = HTTPS
        //   sni: SNI hostname for TLS (empty for non-TLS)
        let peer = HttpPeer::new(addr, false, String::new());

        Ok(Box::new(peer))
    }
}

// === MAIN FUNCTION ===
fn main() {
    // Step 1: Create the server
    // None = use default configuration
    // You can also pass a path to a config file
    let mut server = Server::new(None).unwrap();

    // Step 2: Bootstrap the server
    // This initializes the runtime, signal handlers, etc.
    server.bootstrap();

    // Step 3: Create our proxy instance
    let proxy = FirstProxy::new("127.0.0.1:8080");

    // Step 4: Wrap it in an HTTP proxy service
    // This creates a Service that:
    // - Accepts HTTP connections
    // - Parses HTTP requests
    // - Calls our ProxyHttp methods
    // - Forwards to upstream
    // - Returns responses
    let mut proxy_service = http_proxy_service(
        &server.configuration,  // Server config (timeouts, etc.)
        proxy,                  // Our ProxyHttp implementation
    );

    // Step 5: Tell the service where to listen
    // "0.0.0.0:6188" means:
    // - 0.0.0.0 = accept connections from any interface
    // - 6188 = the port number
    proxy_service.add_tcp("0.0.0.0:6188");

    // Step 6: Add the service to the server
    server.add_service(proxy_service);

    // Step 7: Run forever!
    // This blocks and handles requests until shutdown
    println!("Proxy listening on http://0.0.0.0:6188");
    println!("Forwarding to http://127.0.0.1:8080");
    server.run_forever();
}
```

---

## 2.3 Understanding Each Part

### The Struct

```rust
pub struct FirstProxy {
    upstream: String,
}
```

This is your proxy's configuration. It's:
- Created once at startup
- Shared across all requests (immutable access)
- Holds things like: upstream addresses, config options, shared clients

**DON'T** put per-request data here!

### The Context

```rust
pub struct RequestCtx {}
```

This is per-request state. Created fresh for each request. Use for:
- Request ID for tracing
- Timestamps for latency
- User data from auth
- Accumulated response size

### The Trait Implementation

```rust
#[async_trait]
impl ProxyHttp for FirstProxy { ... }
```

The `#[async_trait]` macro is needed because Rust traits can't have async methods directly (yet). This macro transforms your async methods to work with traits.

### upstream_peer()

```rust
async fn upstream_peer(...) -> Result<Box<HttpPeer>>
```

This is the only **required** method. You must tell Pingora where to forward each request. You can:
- Return the same upstream for all requests (like we do here)
- Choose based on URL path
- Choose based on headers
- Implement load balancing
- Return different upstreams for different users

---

## 2.4 Running Your Proxy

### Step 1: Start an Upstream Server

You need something to forward requests to. The simplest option:

```bash
# Terminal 1: Start a simple HTTP server
python3 -m http.server 8080
```

Or use the Docker setup:
```bash
# From project root
docker compose up upstream
```

### Step 2: Run Your Proxy

```bash
# Terminal 2: Run the proxy
cd chapters/02-first-proxy
cargo run
```

### Step 3: Test It!

```bash
# Terminal 3: Send requests through the proxy
curl http://localhost:6188

# You should see the response from the upstream server!
```

### Step 4: Observe

Watch Terminal 1 (upstream) - you'll see the request arrive from the proxy, not directly from curl.

---

## 2.5 What Just Happened?

```
┌────────┐         ┌──────────────┐         ┌──────────────┐
│  curl  │ ──────▶ │  Your Proxy  │ ──────▶ │   Python     │
│        │         │  port 6188   │         │  port 8080   │
│        │ ◀────── │              │ ◀────── │              │
└────────┘         └──────────────┘         └──────────────┘
    │                     │                       │
    │  GET / HTTP/1.1     │  GET / HTTP/1.1       │
    │ ──────────────────▶ │ ────────────────────▶ │
    │                     │                       │
    │  HTTP/1.1 200 OK    │  HTTP/1.1 200 OK      │
    │ ◀────────────────── │ ◀──────────────────── │
```

1. curl connects to your proxy (port 6188)
2. Proxy receives the request
3. `new_ctx()` creates a context
4. `upstream_peer()` returns the Python server address
5. Proxy connects to Python server (port 8080)
6. Proxy forwards the request
7. Python server responds
8. Proxy forwards response back to curl

---

## 2.6 Common Errors and Fixes

### Error: "Connection refused"

```
Error: Connection refused (os error 61)
```

**Cause**: The upstream server isn't running.
**Fix**: Make sure your upstream (Python server) is running on port 8080.

### Error: "Address already in use"

```
Error: Address already in use (os error 48)
```

**Cause**: Port 6188 is already in use.
**Fix**: Either stop the other process or change the port in `add_tcp()`.

### Error: "Invalid upstream address"

```
thread 'main' panicked at 'Invalid upstream address'
```

**Cause**: The address string can't be parsed.
**Fix**: Use format `"IP:PORT"` like `"127.0.0.1:8080"`.

---

## 2.7 Exercises

### Exercise 2.1: Change the Port
Modify the proxy to listen on port 3000 instead of 6188. Test it.

### Exercise 2.2: Different Upstream
1. Start another server on port 9000: `python3 -m http.server 9000`
2. Modify the proxy to forward to port 9000
3. Test and verify it works

### Exercise 2.3: Add Startup Logging
Add print statements to:
- `new()` - "Proxy created"
- `new_ctx()` - "New request received"
- `upstream_peer()` - "Selecting upstream: {address}"

Run and observe the output for multiple requests.

### Exercise 2.4: Environment Variable
Modify the code to read the upstream address from an environment variable `UPSTREAM_ADDR`, defaulting to `"127.0.0.1:8080"`.

Hint:
```rust
std::env::var("UPSTREAM_ADDR").unwrap_or_else(|_| "127.0.0.1:8080".to_string())
```

---

## 2.8 Exercise Solutions

### Solution 2.1
```rust
proxy_service.add_tcp("0.0.0.0:3000");
```

### Solution 2.3
```rust
impl FirstProxy {
    pub fn new(upstream: &str) -> Self {
        println!("Proxy created with upstream: {}", upstream);
        Self { upstream: upstream.to_string() }
    }
}

fn new_ctx(&self) -> Self::CTX {
    println!("New request received");
    RequestCtx {}
}

async fn upstream_peer(&self, ...) -> Result<Box<HttpPeer>> {
    println!("Selecting upstream: {}", self.upstream);
    // ... rest of code
}
```

### Solution 2.4
```rust
fn main() {
    let upstream = std::env::var("UPSTREAM_ADDR")
        .unwrap_or_else(|_| "127.0.0.1:8080".to_string());

    // ... server setup ...

    let proxy = FirstProxy::new(&upstream);
    // ... rest of code
}
```

---

## Key Takeaways

1. **Minimal proxy needs**: struct + context + `upstream_peer()` implementation
2. **Server lifecycle**: `new()` → `bootstrap()` → `add_service()` → `run_forever()`
3. **HttpPeer** represents a connection to upstream
4. **Each request** gets a fresh context via `new_ctx()`
5. **Test with curl** against your proxy port

---

## Next Chapter

Your proxy works but does nothing special. Let's add request and response manipulation!

Continue to [Chapter 03: Request & Response Manipulation](../03-request-response/README.md)
