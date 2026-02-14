// Chapter 04: Load Balancing
//
// This proxy demonstrates:
// - Round-robin load balancing across multiple backends
// - Health tracking per backend
// - Failure handling
// - Per-backend statistics
//
// To test, start multiple upstream servers:
//   python3 -m http.server 8081  (terminal 1)
//   python3 -m http.server 8082  (terminal 2)
//   python3 -m http.server 8083  (terminal 3)
//
// Then run:
//   cargo run
//
// Test with multiple requests:
//   for i in {1..10}; do curl -s http://localhost:6188/ > /dev/null; done

use async_trait::async_trait;
use pingora::prelude::*;
use pingora::proxy::{http_proxy_service, ProxyHttp, Session};
use std::sync::atomic::{AtomicBool, AtomicU64, AtomicUsize, Ordering};
use std::sync::Arc;
use std::time::Instant;

// ============================================
// BACKEND DEFINITION
// ============================================

/// Represents a single backend server with health status and stats
pub struct Backend {
    /// Server address (ip:port)
    addr: String,
    /// Whether this backend is considered healthy
    healthy: AtomicBool,
    /// Number of requests sent to this backend
    request_count: AtomicU64,
    /// Number of failed requests
    failure_count: AtomicU64,
}

impl Backend {
    pub fn new(addr: &str) -> Self {
        Self {
            addr: addr.to_string(),
            healthy: AtomicBool::new(true),
            request_count: AtomicU64::new(0),
            failure_count: AtomicU64::new(0),
        }
    }

    pub fn is_healthy(&self) -> bool {
        self.healthy.load(Ordering::Relaxed)
    }

    pub fn mark_unhealthy(&self) {
        self.healthy.store(false, Ordering::Relaxed);
    }

    pub fn mark_healthy(&self) {
        self.healthy.store(true, Ordering::Relaxed);
    }

    pub fn record_request(&self) {
        self.request_count.fetch_add(1, Ordering::Relaxed);
    }

    pub fn record_failure(&self) {
        self.failure_count.fetch_add(1, Ordering::Relaxed);
    }

    pub fn stats(&self) -> (u64, u64) {
        (
            self.request_count.load(Ordering::Relaxed),
            self.failure_count.load(Ordering::Relaxed),
        )
    }
}

// ============================================
// LOAD BALANCER PROXY
// ============================================

pub struct LoadBalancerProxy {
    /// List of backend servers
    backends: Vec<Arc<Backend>>,
    /// Current index for round-robin
    current_index: AtomicUsize,
}

impl LoadBalancerProxy {
    pub fn new(addrs: Vec<&str>) -> Self {
        let backends = addrs
            .into_iter()
            .map(|addr| Arc::new(Backend::new(addr)))
            .collect();

        Self {
            backends,
            current_index: AtomicUsize::new(0),
        }
    }

    /// Round-robin selection among healthy backends
    fn select_backend(&self) -> Option<Arc<Backend>> {
        let len = self.backends.len();
        let start = self.current_index.fetch_add(1, Ordering::Relaxed);

        // Try each backend starting from current position
        for i in 0..len {
            let idx = (start + i) % len;
            let backend = &self.backends[idx];

            if backend.is_healthy() {
                return Some(Arc::clone(backend));
            }
        }

        // No healthy backends - try first one anyway (fail gracefully)
        println!("WARNING: No healthy backends available!");
        Some(Arc::clone(&self.backends[0]))
    }

    /// Find backend by address
    fn find_backend(&self, addr: &str) -> Option<&Arc<Backend>> {
        self.backends.iter().find(|b| b.addr == addr)
    }

    /// Print statistics for all backends
    fn print_stats(&self) {
        println!("\n=== Backend Statistics ===");
        for backend in &self.backends {
            let (requests, failures) = backend.stats();
            let health = if backend.is_healthy() { "healthy" } else { "UNHEALTHY" };
            println!(
                "  {} [{}]: {} requests, {} failures",
                backend.addr, health, requests, failures
            );
        }
        println!("========================\n");
    }
}

// ============================================
// PER-REQUEST CONTEXT
// ============================================

pub struct RequestCtx {
    /// Selected backend for this request
    backend: Option<Arc<Backend>>,
    /// Request timing
    start_time: Instant,
    /// Request ID
    request_id: u32,
}

impl RequestCtx {
    pub fn new() -> Self {
        Self {
            backend: None,
            start_time: Instant::now(),
            request_id: rand::random(),
        }
    }
}

// ============================================
// PROXYHTTP IMPLEMENTATION
// ============================================

#[async_trait]
impl ProxyHttp for LoadBalancerProxy {
    type CTX = RequestCtx;

    fn new_ctx(&self) -> Self::CTX {
        RequestCtx::new()
    }

    /// Select upstream using round-robin among healthy backends
    async fn upstream_peer(
        &self,
        _session: &mut Session,
        ctx: &mut Self::CTX,
    ) -> Result<Box<HttpPeer>> {
        // Select a backend
        let backend = self.select_backend()
            .ok_or_else(|| Error::new(ErrorType::ConnectNoRoute))?;

        // Record that we're sending a request
        backend.record_request();

        println!(
            "[{}] Selected backend: {} (requests: {})",
            ctx.request_id,
            backend.addr,
            backend.request_count.load(Ordering::Relaxed)
        );

        // Store backend in context for logging
        ctx.backend = Some(Arc::clone(&backend));

        // Create the peer
        let addr: std::net::SocketAddr = backend.addr.parse().unwrap();
        Ok(Box::new(HttpPeer::new(addr, false, String::new())))
    }

    /// Add load balancer headers to request
    async fn upstream_request_filter(
        &self,
        _session: &mut Session,
        upstream_request: &mut RequestHeader,
        ctx: &mut Self::CTX,
    ) -> Result<()> {
        upstream_request
            .insert_header("X-Request-ID", ctx.request_id.to_string())
            .unwrap();

        if let Some(ref backend) = ctx.backend {
            upstream_request
                .insert_header("X-Backend", &backend.addr)
                .unwrap();
        }

        Ok(())
    }

    /// Add backend info to response
    async fn response_filter(
        &self,
        _session: &mut Session,
        upstream_response: &mut ResponseHeader,
        ctx: &mut Self::CTX,
    ) -> Result<()>
    where
        Self::CTX: Send + Sync,
    {
        // Add timing
        let elapsed_ms = ctx.start_time.elapsed().as_millis();
        upstream_response
            .insert_header("X-Response-Time", format!("{}ms", elapsed_ms))
            .unwrap();

        // Add which backend served this
        if let Some(ref backend) = ctx.backend {
            upstream_response
                .insert_header("X-Served-By", &backend.addr)
                .unwrap();
        }

        Ok(())
    }

    /// Handle connection failures
    async fn fail_to_connect(
        &self,
        _session: &mut Session,
        peer: &HttpPeer,
        ctx: &mut Self::CTX,
        e: Box<Error>,
    ) -> Box<Error> {
        let addr = format!("{}", peer.address());

        println!(
            "[{}] Connection failed to {}: {}",
            ctx.request_id, addr, e
        );

        // Find and mark backend unhealthy
        if let Some(backend) = self.find_backend(&addr) {
            backend.record_failure();
            backend.mark_unhealthy();
            println!("[{}] Marked {} as unhealthy", ctx.request_id, addr);
        }

        e
    }

    /// Log request completion
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

        let backend_addr = ctx
            .backend
            .as_ref()
            .map(|b| b.addr.as_str())
            .unwrap_or("none");

        if let Some(e) = error {
            println!(
                "[{}] {} {} -> {} | backend={} | error={} | {:?}",
                ctx.request_id,
                session.req_header().method,
                session.req_header().uri.path(),
                status,
                backend_addr,
                e,
                elapsed
            );
        } else {
            println!(
                "[{}] {} {} -> {} | backend={} | {:?}",
                ctx.request_id,
                session.req_header().method,
                session.req_header().uri.path(),
                status,
                backend_addr,
                elapsed
            );
        }

        // Print stats periodically (every 5 requests to first backend)
        if let Some(ref backend) = ctx.backend {
            if backend.request_count.load(Ordering::Relaxed) % 5 == 0 {
                self.print_stats();
            }
        }
    }
}

// ============================================
// MAIN
// ============================================

fn main() {
    // Parse backend list from environment or use defaults
    let backends_str = std::env::var("BACKENDS")
        .unwrap_or_else(|_| "127.0.0.1:8081,127.0.0.1:8082,127.0.0.1:8083".to_string());

    let backends: Vec<&str> = backends_str.split(',').collect();

    let mut server = Server::new(None).unwrap();
    server.bootstrap();

    let proxy = LoadBalancerProxy::new(backends.clone());
    let mut service = http_proxy_service(&server.configuration, proxy);
    service.add_tcp("0.0.0.0:6188");

    server.add_service(service);

    println!("=============================================");
    println!("  Load Balancer Proxy - Chapter 04");
    println!("=============================================");
    println!("  Listening: http://0.0.0.0:6188");
    println!("  Backends:");
    for (i, backend) in backends.iter().enumerate() {
        println!("    {}. {}", i + 1, backend);
    }
    println!("=============================================");
    println!("  Algorithm: Round-Robin with health check");
    println!("=============================================");
    println!("  Test with multiple requests:");
    println!("    for i in {{1..10}}; do");
    println!("      curl -s http://localhost:6188/");
    println!("    done");
    println!("=============================================");

    server.run_forever();
}
