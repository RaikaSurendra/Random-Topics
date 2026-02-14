// Chapter 03: Request & Response Manipulation
//
// This proxy demonstrates:
// - Adding request headers
// - Adding response headers
// - Request timing/tracking
// - Path-based routing
// - Early request rejection
//
// Run with: cargo run
// Test with:
//   curl -v http://localhost:6188/
//   curl -v http://localhost:6188/api/users
//   curl -v http://localhost:6188/admin  (blocked!)

use async_trait::async_trait;
use pingora::prelude::*;
use pingora::proxy::{http_proxy_service, ProxyHttp, Session};
use std::time::Instant;

// ============================================
// PROXY CONFIGURATION
// ============================================

pub struct SmartProxy {
    /// Default backend for general requests
    default_upstream: String,
    /// Backend for API requests (/api/*)
    api_upstream: String,
}

impl SmartProxy {
    pub fn new(default: &str, api: &str) -> Self {
        Self {
            default_upstream: default.to_string(),
            api_upstream: api.to_string(),
        }
    }
}

// ============================================
// PER-REQUEST CONTEXT
// ============================================

pub struct RequestCtx {
    /// Unique ID for this request (for tracing)
    request_id: u32,
    /// When the request started (for timing)
    start_time: Instant,
    /// Which upstream was selected
    upstream_name: String,
}

// ============================================
// PROXY IMPLEMENTATION
// ============================================

#[async_trait]
impl ProxyHttp for SmartProxy {
    type CTX = RequestCtx;

    /// Create context for each new request
    fn new_ctx(&self) -> Self::CTX {
        RequestCtx {
            request_id: rand::random(),
            start_time: Instant::now(),
            upstream_name: String::new(),
        }
    }

    /// FILTER 1: Early request inspection
    /// Return Ok(true) to short-circuit (response already sent)
    /// Return Ok(false) to continue processing
    async fn request_filter(
        &self,
        session: &mut Session,
        ctx: &mut Self::CTX,
    ) -> Result<bool> {
        let req = session.req_header();
        let path = req.uri.path();

        println!("[{}] Incoming: {} {}", ctx.request_id, req.method, path);

        // Security: Block admin paths
        if path.starts_with("/admin") {
            println!("[{}] BLOCKED: Admin path access denied", ctx.request_id);
            return Err(Error::new(ErrorType::HTTPStatus(403)));
        }

        // Security: Block path traversal attempts
        if path.contains("..") {
            println!("[{}] BLOCKED: Path traversal attempt", ctx.request_id);
            return Err(Error::new(ErrorType::HTTPStatus(400)));
        }

        // Continue normal processing
        Ok(false)
    }

    /// FILTER 2: Choose upstream server
    async fn upstream_peer(
        &self,
        session: &mut Session,
        ctx: &mut Self::CTX,
    ) -> Result<Box<HttpPeer>> {
        let path = session.req_header().uri.path();

        // Route based on path
        let (upstream, name) = if path.starts_with("/api/") {
            (&self.api_upstream, "api")
        } else {
            (&self.default_upstream, "default")
        };

        // Store which upstream we chose (for logging)
        ctx.upstream_name = name.to_string();

        println!(
            "[{}] Routing to {} backend: {}",
            ctx.request_id, name, upstream
        );

        let addr: std::net::SocketAddr = upstream
            .parse()
            .expect("Invalid upstream address");

        Ok(Box::new(HttpPeer::new(addr, false, String::new())))
    }

    /// FILTER 3: Modify request before sending upstream
    async fn upstream_request_filter(
        &self,
        session: &mut Session,
        upstream_request: &mut RequestHeader,
        ctx: &mut Self::CTX,
    ) -> Result<()> {
        // Add request tracing header
        upstream_request
            .insert_header("X-Request-ID", ctx.request_id.to_string())
            .unwrap();

        // Add proxy identification
        upstream_request
            .insert_header("X-Forwarded-By", "Pingora-SmartProxy")
            .unwrap();

        // Forward original host (important for virtual hosting)
        if let Some(host) = session.req_header().headers.get("Host") {
            upstream_request
                .insert_header("X-Original-Host", host.to_str().unwrap_or("unknown"))
                .unwrap();
        }

        // Add timing info
        let elapsed_us = ctx.start_time.elapsed().as_micros();
        upstream_request
            .insert_header("X-Request-Start", elapsed_us.to_string())
            .unwrap();

        println!(
            "[{}] Added headers, forwarding request",
            ctx.request_id
        );

        Ok(())
    }

    /// FILTER 4: Modify response before sending to client
    async fn response_filter(
        &self,
        _session: &mut Session,
        upstream_response: &mut ResponseHeader,
        ctx: &mut Self::CTX,
    ) -> Result<()>
    where
        Self::CTX: Send + Sync,
    {
        let elapsed_ms = ctx.start_time.elapsed().as_millis();

        // Add timing information
        upstream_response
            .insert_header("X-Response-Time", format!("{}ms", elapsed_ms))
            .unwrap();

        // Add request ID for client-side correlation
        upstream_response
            .insert_header("X-Request-ID", ctx.request_id.to_string())
            .unwrap();

        // Add which backend served this request
        upstream_response
            .insert_header("X-Served-By", &ctx.upstream_name)
            .unwrap();

        // Security headers (always good practice)
        upstream_response
            .insert_header("X-Content-Type-Options", "nosniff")
            .unwrap();

        upstream_response
            .insert_header("X-Frame-Options", "DENY")
            .unwrap();

        upstream_response
            .insert_header("X-XSS-Protection", "1; mode=block")
            .unwrap();

        // Remove potentially sensitive upstream headers
        upstream_response.remove_header("Server");
        upstream_response.remove_header("X-Powered-By");

        println!(
            "[{}] Response modified, took {}ms",
            ctx.request_id, elapsed_ms
        );

        Ok(())
    }

    /// LOGGING: Called after request completes
    async fn logging(
        &self,
        session: &mut Session,
        error: Option<&pingora::Error>,
        ctx: &mut Self::CTX,
    ) {
        let elapsed = ctx.start_time.elapsed();
        let req = session.req_header();

        // Get response status
        let status = session
            .response_written()
            .map(|r| r.status.as_u16())
            .unwrap_or(0);

        // Get User-Agent for analytics
        let user_agent = req
            .headers
            .get("User-Agent")
            .and_then(|v| v.to_str().ok())
            .unwrap_or("unknown");

        if let Some(e) = error {
            println!(
                "[{}] ERROR {} {} -> {} | upstream={} | error={} | {:?}",
                ctx.request_id,
                req.method,
                req.uri.path(),
                status,
                ctx.upstream_name,
                e,
                elapsed
            );
        } else {
            println!(
                "[{}] {} {} -> {} | upstream={} | ua={} | {:?}",
                ctx.request_id,
                req.method,
                req.uri.path(),
                status,
                ctx.upstream_name,
                &user_agent[..user_agent.len().min(30)],  // Truncate UA
                elapsed
            );
        }
    }
}

// ============================================
// MAIN
// ============================================

fn main() {
    let default_upstream = std::env::var("DEFAULT_UPSTREAM")
        .unwrap_or_else(|_| "127.0.0.1:8080".to_string());

    let api_upstream = std::env::var("API_UPSTREAM")
        .unwrap_or_else(|_| "127.0.0.1:8080".to_string());

    let mut server = Server::new(None).unwrap();
    server.bootstrap();

    let proxy = SmartProxy::new(&default_upstream, &api_upstream);
    let mut service = http_proxy_service(&server.configuration, proxy);
    service.add_tcp("0.0.0.0:6188");

    server.add_service(service);

    println!("=============================================");
    println!("  Smart Proxy - Chapter 03");
    println!("=============================================");
    println!("  Listening:    http://0.0.0.0:6188");
    println!("  Default:      http://{}", default_upstream);
    println!("  API (/api/*): http://{}", api_upstream);
    println!("=============================================");
    println!("  Try these:");
    println!("    curl -v http://localhost:6188/");
    println!("    curl -v http://localhost:6188/api/users");
    println!("    curl http://localhost:6188/admin (blocked)");
    println!("=============================================");

    server.run_forever();
}
