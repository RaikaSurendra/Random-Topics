# Chapter 08: Advanced Topics

## Learning Objectives

By the end of this chapter, you will:
- Configure TLS termination
- Implement graceful restarts
- Tune performance settings
- Understand Pingora internals

---

## 8.1 TLS Termination

### Why Terminate TLS at Proxy?

```
Client ──HTTPS──▶ Proxy ──HTTP──▶ Backend
                    │
                    └── TLS terminates here
```

Benefits:
- Centralized certificate management
- Offload crypto from backends
- Inspect traffic for security

### Configuration

```rust
use pingora::tls::ssl::{SslAcceptor, SslMethod, SslFiletype};

fn configure_tls() -> SslAcceptor {
    let mut builder = SslAcceptor::mozilla_modern(SslMethod::tls()).unwrap();

    // Load certificate and key
    builder.set_certificate_chain_file("/path/to/cert.pem").unwrap();
    builder.set_private_key_file("/path/to/key.pem", SslFiletype::PEM).unwrap();

    // Optional: Configure protocols and ciphers
    builder.set_min_proto_version(Some(SslVersion::TLS1_2)).unwrap();

    builder.build()
}

fn main() {
    let mut server = Server::new(None).unwrap();
    server.bootstrap();

    let proxy = MyProxy::new();
    let mut service = http_proxy_service(&server.configuration, proxy);

    // Add HTTPS listener
    let tls = configure_tls();
    service.add_tls("0.0.0.0:443", tls);

    // Keep HTTP for redirects
    service.add_tcp("0.0.0.0:80");

    server.add_service(service);
    server.run_forever();
}
```

### HTTP to HTTPS Redirect

```rust
async fn request_filter(&self, session: &mut Session, _ctx: &mut Self::CTX) -> Result<bool> {
    // Check if connection is not TLS
    if !session.is_tls() {
        let host = session.req_header().headers.get("Host")
            .map(|h| h.to_str().unwrap_or(""))
            .unwrap_or("");

        let path = session.req_header().uri.path_and_query()
            .map(|pq| pq.as_str())
            .unwrap_or("/");

        // Build redirect response
        let mut header = ResponseHeader::build(301, None)?;
        header.insert_header("Location", format!("https://{}{}", host, path))?;

        session.write_response_header(Box::new(header), true).await?;
        return Ok(true);  // Response sent, skip upstream
    }

    Ok(false)
}
```

---

## 8.2 Graceful Shutdown & Hot Restarts

### The Problem

Normal shutdown:
```
1. Stop accepting new connections
2. Forcefully close existing connections
3. Exit

→ Clients see errors!
```

Graceful shutdown:
```
1. Stop accepting new connections
2. Wait for in-flight requests to complete (with timeout)
3. Close idle connections
4. Exit

→ No client errors!
```

### Pingora Signal Handling

Pingora handles signals automatically:

| Signal | Action |
|--------|--------|
| `SIGTERM` | Graceful shutdown |
| `SIGQUIT` | Graceful shutdown |
| `SIGHUP` | Graceful restart (upgrade) |
| `SIGINT` | Immediate shutdown |

### Hot Restart (Zero Downtime Upgrade)

```bash
# Running process (PID 1234)
./proxy

# Deploy new version
./proxy_new --upgrade

# Old process:
# 1. Passes listening sockets to new process
# 2. Stops accepting new connections
# 3. Waits for existing requests
# 4. Exits

# New process:
# 1. Receives sockets
# 2. Immediately accepts new connections
# 3. Serves traffic with zero downtime
```

### Configuration for Graceful Restart

```rust
let mut server = Server::new(Some(Opt {
    upgrade: std::env::args().any(|arg| arg == "--upgrade"),
    daemon: false,
    nocapture: false,
    test: false,
    conf: None,
})).unwrap();
```

### Graceful Shutdown Timeout

```yaml
# pingora.conf
server:
  grace_period_seconds: 30  # Wait up to 30s for requests
  graceful_shutdown_timeout_seconds: 60
```

---

## 8.3 Performance Tuning

### Connection Pooling

Reuse connections to backends:

```rust
// In Pingora server configuration
server:
  upstream_keepalive_pool_size: 128  # Per-backend pool size
  upstream_connect_timeout: 5s
  upstream_read_timeout: 60s
```

Without pooling:
```
Request → TCP Handshake (1.5ms) → TLS Handshake (5ms) → HTTP → Response
```

With pooling:
```
Request → Reuse Connection → HTTP → Response (saves 6.5ms!)
```

### Buffer Sizes

```rust
server:
  # Incoming request buffers
  request_body_buffer_size: 64KB

  # Response buffers
  response_body_buffer_size: 64KB

  # For large file uploads
  max_request_body_size: 100MB
```

### Worker Threads

```rust
server:
  threads: 4  # Or auto-detect: num_cpus::get()
```

Rule of thumb:
- CPU-bound work: threads = CPU cores
- IO-bound work: threads = 2 * CPU cores

### TCP Settings

```rust
// Enable TCP_NODELAY (disable Nagle's algorithm)
// Good for: Low-latency requests
// Bad for: Bulk transfers

// Pingora enables this by default for proxying
```

---

## 8.4 Rate Limiting

Protect backends from overload:

```rust
use std::collections::HashMap;
use std::sync::{Arc, Mutex};
use std::time::{Duration, Instant};

pub struct RateLimiter {
    requests: Mutex<HashMap<String, Vec<Instant>>>,
    limit: usize,
    window: Duration,
}

impl RateLimiter {
    pub fn new(limit: usize, window: Duration) -> Self {
        Self {
            requests: Mutex::new(HashMap::new()),
            limit,
            window,
        }
    }

    pub fn check(&self, key: &str) -> bool {
        let now = Instant::now();
        let mut requests = self.requests.lock().unwrap();

        let entry = requests.entry(key.to_string()).or_insert_with(Vec::new);

        // Remove old requests outside window
        entry.retain(|&t| now.duration_since(t) < self.window);

        if entry.len() < self.limit {
            entry.push(now);
            true  // Allow
        } else {
            false  // Rate limited
        }
    }
}

// In proxy
async fn request_filter(&self, session: &mut Session, _ctx: &mut Self::CTX) -> Result<bool> {
    let client_ip = session.client_addr()
        .map(|a| a.to_string())
        .unwrap_or_else(|| "unknown".to_string());

    if !self.rate_limiter.check(&client_ip) {
        return Err(Error::new(ErrorType::HTTPStatus(429)));
    }

    Ok(false)
}
```

---

## 8.5 Request/Response Body Modification

### Reading Request Body

```rust
async fn request_body_filter(
    &self,
    session: &mut Session,
    body: &mut Option<Bytes>,
    end_of_stream: bool,
    ctx: &mut Self::CTX,
) -> Result<()> {
    if let Some(data) = body {
        // Accumulate body chunks
        ctx.request_body.extend_from_slice(data);

        if end_of_stream {
            // Process complete body
            println!("Full request body: {} bytes", ctx.request_body.len());

            // Optionally modify
            if let Ok(json) = serde_json::from_slice::<Value>(&ctx.request_body) {
                // Modify JSON...
                let modified = serde_json::to_vec(&json)?;
                *body = Some(Bytes::from(modified));
            }
        }
    }
    Ok(())
}
```

### Modifying Response Body

```rust
async fn response_body_filter(
    &self,
    _session: &mut Session,
    body: &mut Option<Bytes>,
    end_of_stream: bool,
    ctx: &mut Self::CTX,
) -> Result<Option<Duration>> {
    if let Some(data) = body {
        // Example: Inject script into HTML
        if ctx.is_html && end_of_stream {
            let html = String::from_utf8_lossy(data);
            let modified = html.replace("</body>", "<script>console.log('injected')</script></body>");
            *body = Some(Bytes::from(modified.into_owned()));
        }
    }
    Ok(None)
}
```

---

## 8.6 Custom Error Pages

```rust
async fn fail_to_proxy(
    &self,
    session: &mut Session,
    e: &Error,
    _ctx: &mut Self::CTX,
) -> u16 {
    let (status, message) = match e.etype() {
        ErrorType::ConnectTimeout => (504, "Gateway Timeout"),
        ErrorType::ReadTimeout => (504, "Gateway Timeout"),
        ErrorType::ConnectRefused => (502, "Bad Gateway"),
        ErrorType::HTTPStatus(code) => (*code, "Upstream Error"),
        _ => (500, "Internal Server Error"),
    };

    // Write custom error response
    let body = format!(
        r#"<!DOCTYPE html>
        <html>
        <head><title>{} {}</title></head>
        <body>
            <h1>{} {}</h1>
            <p>Request ID: {}</p>
        </body>
        </html>"#,
        status, message, status, message, ctx.request_id
    );

    let mut header = ResponseHeader::build(status, None).unwrap();
    header.insert_header("Content-Type", "text/html").unwrap();

    session.write_response_header(Box::new(header), false).await.ok();
    session.write_response_body(Some(Bytes::from(body)), true).await.ok();

    status
}
```

---

## 8.7 WebSocket Proxying

```rust
async fn upstream_peer(
    &self,
    session: &mut Session,
    ctx: &mut Self::CTX,
) -> Result<Box<HttpPeer>> {
    let req = session.req_header();

    // Check for WebSocket upgrade
    let is_websocket = req.headers.get("Upgrade")
        .map(|v| v.to_str().unwrap_or("").to_lowercase() == "websocket")
        .unwrap_or(false);

    if is_websocket {
        ctx.is_websocket = true;
        // Route to WebSocket backend
        return Ok(Box::new(HttpPeer::new(
            "127.0.0.1:9000".parse().unwrap(),
            false,
            String::new(),
        )));
    }

    // Normal HTTP routing
    // ...
}
```

---

## 8.8 Configuration Management

### YAML Configuration

```yaml
# pingora.conf
server:
  threads: 4
  grace_period_seconds: 30

proxy:
  listen:
    - addr: "0.0.0.0:80"
    - addr: "0.0.0.0:443"
      tls:
        cert: /etc/ssl/cert.pem
        key: /etc/ssl/key.pem

  upstreams:
    - name: api
      backends:
        - addr: "10.0.0.1:8080"
          weight: 5
        - addr: "10.0.0.2:8080"
          weight: 3

    - name: web
      backends:
        - addr: "10.0.0.10:80"

  routes:
    - path_prefix: /api/
      upstream: api
    - path_prefix: /
      upstream: web
```

### Loading Config

```rust
use serde::Deserialize;

#[derive(Deserialize)]
struct Config {
    server: ServerConfig,
    proxy: ProxyConfig,
}

fn load_config(path: &str) -> Config {
    let content = std::fs::read_to_string(path).unwrap();
    serde_yaml::from_str(&content).unwrap()
}
```

---

## Exercises

### Exercise 8.1: TLS Setup
Configure your proxy to accept HTTPS on port 443. Generate self-signed certificates for testing.

### Exercise 8.2: Rate Limiting
Implement per-IP rate limiting: 100 requests per minute.

### Exercise 8.3: Custom Error Pages
Create styled error pages for 502, 503, and 504 errors.

### Exercise 8.4: Config Hot Reload
Implement configuration reloading on SIGHUP without restarting.

---

## Key Takeaways

1. **TLS termination** centralizes certificate management
2. **Graceful shutdown** prevents client errors during deploys
3. **Connection pooling** dramatically reduces latency
4. **Rate limiting** protects backends from overload
5. **Configuration files** make deployment flexible

---

## Congratulations!

You've completed the Pingora Mastery Course! You now know how to:

- Build custom reverse proxies
- Manipulate requests and responses
- Load balance across backends
- Monitor backend health
- Cache responses
- Collect metrics and traces
- Handle TLS and advanced scenarios

### What's Next?

1. **Build something real** - Apply these skills to a project
2. **Read Pingora source** - Learn from the implementation
3. **Contribute** - Fix bugs or add features to Pingora
4. **Share knowledge** - Write about what you learned

### Resources

- [Pingora GitHub](https://github.com/cloudflare/pingora)
- [Pingora Docs](https://docs.rs/pingora/latest/pingora/)
- [Cloudflare Blog](https://blog.cloudflare.com/tag/pingora/)

Good luck building amazing proxies! 🚀
