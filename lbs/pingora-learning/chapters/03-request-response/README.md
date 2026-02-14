# Chapter 03: Request & Response Manipulation

## Learning Objectives

By the end of this chapter, you will:
- Modify incoming request headers
- Modify outgoing response headers
- Implement URL path rewriting
- Route requests based on URL patterns
- Use the request context effectively

---

## 3.1 The Filter Methods

Pingora provides several "hooks" where you can inspect and modify traffic:

```
Client Request
     │
     ▼
┌─────────────────────┐
│   request_filter()  │  ◀── Inspect/reject early
└─────────────────────┘
     │
     ▼
┌─────────────────────┐
│   upstream_peer()   │  ◀── Choose backend
└─────────────────────┘
     │
     ▼
┌─────────────────────────────┐
│ upstream_request_filter()   │  ◀── Modify before sending upstream
└─────────────────────────────┘
     │
     ▼
   [Upstream]
     │
     ▼
┌──────────────────────────────┐
│ upstream_response_filter()   │  ◀── First look at response
└──────────────────────────────┘
     │
     ▼
┌─────────────────────┐
│  response_filter()  │  ◀── Final modifications
└─────────────────────┘
     │
     ▼
Client Response
```

---

## 3.2 Adding Request Headers

Let's add a custom header to every request going upstream:

```rust
async fn upstream_request_filter(
    &self,
    _session: &mut Session,
    upstream_request: &mut RequestHeader,
    _ctx: &mut Self::CTX,
) -> Result<()> {
    // Add a header to identify traffic from our proxy
    upstream_request
        .insert_header("X-Forwarded-By", "Pingora-Learning")
        .unwrap();

    // Add the real client IP (important for logging on backend)
    upstream_request
        .insert_header("X-Real-IP", "127.0.0.1")  // We'll get real IP later
        .unwrap();

    Ok(())
}
```

### Header Methods

| Method | Purpose |
|--------|---------|
| `insert_header(name, value)` | Add header (replaces if exists) |
| `append_header(name, value)` | Add header (keeps existing) |
| `remove_header(name)` | Remove a header |
| `headers.get(name)` | Read a header value |

---

## 3.3 Adding Response Headers

Add security headers to every response:

```rust
async fn response_filter(
    &self,
    _session: &mut Session,
    upstream_response: &mut ResponseHeader,
    _ctx: &mut Self::CTX,
) -> Result<()>
where
    Self::CTX: Send + Sync,
{
    // Security headers
    upstream_response
        .insert_header("X-Content-Type-Options", "nosniff")
        .unwrap();

    upstream_response
        .insert_header("X-Frame-Options", "DENY")
        .unwrap();

    // Custom header showing this went through our proxy
    upstream_response
        .insert_header("X-Proxy", "Pingora-Learning")
        .unwrap();

    Ok(())
}
```

---

## 3.4 Reading Request Information

The `Session` object gives you access to request details:

```rust
async fn upstream_peer(
    &self,
    session: &mut Session,
    ctx: &mut Self::CTX,
) -> Result<Box<HttpPeer>> {
    // Get the request header
    let req = session.req_header();

    // Read various parts:
    let method = &req.method;           // GET, POST, etc.
    let uri = &req.uri;                 // Full URI
    let path = uri.path();              // Just the path: /api/users
    let query = uri.query();            // Query string: foo=bar

    // Read a specific header
    let host = req.headers.get("Host")
        .map(|v| v.to_str().unwrap_or("unknown"))
        .unwrap_or("unknown");

    let user_agent = req.headers.get("User-Agent")
        .map(|v| v.to_str().unwrap_or("unknown"))
        .unwrap_or("unknown");

    println!("Request: {} {} (Host: {})", method, path, host);

    // ... choose upstream based on this info
}
```

---

## 3.5 URL-Based Routing

Route different paths to different backends:

```rust
pub struct RoutingProxy {
    api_upstream: String,     // Backend for /api/*
    web_upstream: String,     // Backend for everything else
}

#[async_trait]
impl ProxyHttp for RoutingProxy {
    // ...

    async fn upstream_peer(
        &self,
        session: &mut Session,
        _ctx: &mut Self::CTX,
    ) -> Result<Box<HttpPeer>> {
        let path = session.req_header().uri.path();

        // Choose upstream based on path
        let upstream = if path.starts_with("/api/") {
            println!("Routing to API backend");
            &self.api_upstream
        } else {
            println!("Routing to Web backend");
            &self.web_upstream
        };

        let addr: std::net::SocketAddr = upstream.parse().unwrap();
        Ok(Box::new(HttpPeer::new(addr, false, String::new())))
    }
}
```

---

## 3.6 Path Rewriting

Rewrite URLs before sending upstream:

```rust
async fn upstream_request_filter(
    &self,
    _session: &mut Session,
    upstream_request: &mut RequestHeader,
    _ctx: &mut Self::CTX,
) -> Result<()> {
    // Get current path
    let path = upstream_request.uri.path();

    // Rewrite: /api/v1/users -> /users
    if path.starts_with("/api/v1/") {
        let new_path = path.strip_prefix("/api/v1").unwrap();

        // Create new URI with modified path
        let new_uri = http::Uri::builder()
            .path_and_query(new_path)
            .build()
            .unwrap();

        upstream_request.set_uri(new_uri);
        println!("Rewrote path: {} -> {}", path, new_path);
    }

    Ok(())
}
```

---

## 3.7 Using Context for Request Tracking

Track request timing and add request IDs:

```rust
use std::time::Instant;

pub struct RequestCtx {
    request_id: String,
    start_time: Instant,
}

impl ProxyHttp for MyProxy {
    type CTX = RequestCtx;

    fn new_ctx(&self) -> Self::CTX {
        RequestCtx {
            // Generate simple request ID
            request_id: format!("req-{}", rand::random::<u32>()),
            start_time: Instant::now(),
        }
    }

    async fn upstream_request_filter(
        &self,
        _session: &mut Session,
        upstream_request: &mut RequestHeader,
        ctx: &mut Self::CTX,
    ) -> Result<()> {
        // Add request ID header for tracing
        upstream_request
            .insert_header("X-Request-ID", &ctx.request_id)
            .unwrap();
        Ok(())
    }

    async fn logging(
        &self,
        session: &mut Session,
        _e: Option<&pingora::Error>,
        ctx: &mut Self::CTX,
    ) {
        let elapsed = ctx.start_time.elapsed();
        let status = session
            .response_written()
            .map(|r| r.status.as_u16())
            .unwrap_or(0);

        println!(
            "[{}] {} {} -> {} ({:?})",
            ctx.request_id,
            session.req_header().method,
            session.req_header().uri.path(),
            status,
            elapsed
        );
    }
}
```

---

## 3.8 Early Request Rejection

Reject requests before they reach the backend:

```rust
async fn request_filter(
    &self,
    session: &mut Session,
    _ctx: &mut Self::CTX,
) -> Result<bool> {
    let req = session.req_header();
    let path = req.uri.path();

    // Block access to admin paths
    if path.starts_with("/admin") {
        // Return an error response
        let mut header = ResponseHeader::build(403, None)?;
        header.insert_header("Content-Type", "text/plain")?;

        session.write_response_header(Box::new(header), false).await?;
        session.write_response_body(Some("Forbidden".into()), true).await?;

        // Return true = response already sent, skip remaining processing
        return Ok(true);
    }

    // Return false = continue normal processing
    Ok(false)
}
```

---

## 3.9 Complete Example

Here's a full proxy with multiple features:

```rust
use async_trait::async_trait;
use pingora::prelude::*;
use pingora::proxy::{http_proxy_service, ProxyHttp, Session};
use std::time::Instant;

pub struct SmartProxy {
    default_upstream: String,
}

pub struct RequestCtx {
    request_id: u32,
    start_time: Instant,
}

#[async_trait]
impl ProxyHttp for SmartProxy {
    type CTX = RequestCtx;

    fn new_ctx(&self) -> Self::CTX {
        RequestCtx {
            request_id: rand::random(),
            start_time: Instant::now(),
        }
    }

    // Early rejection
    async fn request_filter(
        &self,
        session: &mut Session,
        _ctx: &mut Self::CTX,
    ) -> Result<bool> {
        let path = session.req_header().uri.path();

        // Block bad paths
        if path.contains("..") || path.contains("//") {
            return Err(Error::new(ErrorType::HTTPStatus(400)));
        }

        Ok(false)
    }

    // Choose upstream
    async fn upstream_peer(
        &self,
        _session: &mut Session,
        _ctx: &mut Self::CTX,
    ) -> Result<Box<HttpPeer>> {
        let addr: std::net::SocketAddr = self.default_upstream.parse().unwrap();
        Ok(Box::new(HttpPeer::new(addr, false, String::new())))
    }

    // Modify request
    async fn upstream_request_filter(
        &self,
        _session: &mut Session,
        upstream_request: &mut RequestHeader,
        ctx: &mut Self::CTX,
    ) -> Result<()> {
        upstream_request
            .insert_header("X-Request-ID", ctx.request_id.to_string())
            .unwrap();
        upstream_request
            .insert_header("X-Forwarded-By", "SmartProxy")
            .unwrap();
        Ok(())
    }

    // Modify response
    async fn response_filter(
        &self,
        _session: &mut Session,
        upstream_response: &mut ResponseHeader,
        ctx: &mut Self::CTX,
    ) -> Result<()>
    where
        Self::CTX: Send + Sync,
    {
        // Add timing header
        let elapsed_ms = ctx.start_time.elapsed().as_millis();
        upstream_response
            .insert_header("X-Response-Time", format!("{}ms", elapsed_ms))
            .unwrap();

        // Security headers
        upstream_response
            .insert_header("X-Content-Type-Options", "nosniff")
            .unwrap();

        Ok(())
    }

    // Log completion
    async fn logging(
        &self,
        session: &mut Session,
        error: Option<&pingora::Error>,
        ctx: &mut Self::CTX,
    ) {
        let elapsed = ctx.start_time.elapsed();
        let status = session
            .response_written()
            .map(|r| r.status.as_u16())
            .unwrap_or(0);

        if let Some(e) = error {
            println!("[{}] ERROR: {} ({:?})", ctx.request_id, e, elapsed);
        } else {
            println!(
                "[{}] {} {} {} ({:?})",
                ctx.request_id,
                session.req_header().method,
                session.req_header().uri.path(),
                status,
                elapsed
            );
        }
    }
}
```

---

## Exercises

### Exercise 3.1: Custom Header
Add a header `X-Processed-At` with the current timestamp to all responses.

Hint: Use `chrono` crate for formatting time.

### Exercise 3.2: User-Agent Logging
Log the User-Agent header for every request in the `logging()` method.

### Exercise 3.3: Path-Based Routing
Create a proxy that:
- Routes `/images/*` to `127.0.0.1:8081`
- Routes `/api/*` to `127.0.0.1:8082`
- Routes everything else to `127.0.0.1:8080`

### Exercise 3.4: Rate Limiting Header
Add a `X-RateLimit-Remaining` header to responses. For now, just hardcode a value like "100".

### Exercise 3.5: Block Bad User-Agents
In `request_filter()`, block requests with User-Agent containing "bot" or "crawler".

---

## Key Takeaways

1. **request_filter()** - First chance to reject or short-circuit
2. **upstream_request_filter()** - Modify requests before upstream
3. **response_filter()** - Add headers to responses
4. **logging()** - Metrics and observability
5. **Context (CTX)** - Track per-request state like timing
6. Use `session.req_header()` to read incoming request
7. Modify `RequestHeader` and `ResponseHeader` directly

---

## Next Chapter

Now you can manipulate traffic! Let's distribute it across multiple backends.

Continue to [Chapter 04: Load Balancing](../04-load-balancing/README.md)
