// Pingora Learning - Simple Reverse Proxy Example
//
// This example creates a basic reverse proxy that forwards
// requests to an upstream server.
//
// Supports both:
// - Local development: cargo run (connects to 127.0.0.1:8080)
// - Docker: uses UPSTREAM_ADDR env var (e.g., "upstream:80")

use async_trait::async_trait;
use pingora::prelude::*;
use pingora::proxy::{http_proxy_service, ProxyHttp, Session};
use std::net::ToSocketAddrs;

/// Our custom proxy implementation
pub struct LearningProxy {
    /// The upstream server address (hostname:port or ip:port)
    upstream_addr: String,
}

impl LearningProxy {
    pub fn new(upstream: &str) -> Self {
        Self {
            upstream_addr: upstream.to_string(),
        }
    }

    /// Resolve hostname to socket address (supports DNS for Docker)
    fn resolve_upstream(&self) -> std::net::SocketAddr {
        self.upstream_addr
            .to_socket_addrs()
            .expect("Failed to resolve upstream address")
            .next()
            .expect("No addresses found for upstream")
    }
}

/// Context for each request - can store per-request state
pub struct RequestContext {
    // Add fields here to track request-specific data
    // e.g., request_id, start_time, etc.
}

#[async_trait]
impl ProxyHttp for LearningProxy {
    /// The context type for each request
    type CTX = RequestContext;

    /// Create a new context for each incoming request
    fn new_ctx(&self) -> Self::CTX {
        RequestContext {}
    }

    /// Determine which upstream server to connect to
    async fn upstream_peer(
        &self,
        _session: &mut Session,
        _ctx: &mut Self::CTX,
    ) -> Result<Box<HttpPeer>> {
        // Resolve the upstream address (supports DNS names)
        let addr = self.resolve_upstream();

        // Create a peer connection (false = no TLS)
        let peer = Box::new(HttpPeer::new(addr, false, String::new()));
        Ok(peer)
    }

    /// Called before sending the request upstream
    /// Use this to modify request headers
    async fn upstream_request_filter(
        &self,
        _session: &mut Session,
        upstream_request: &mut RequestHeader,
        _ctx: &mut Self::CTX,
    ) -> Result<()> {
        // Add a custom header to identify proxied requests
        upstream_request
            .insert_header("X-Proxied-By", "Pingora-Learning")
            .unwrap();
        Ok(())
    }

    /// Logging callback - called after the request completes
    async fn logging(
        &self,
        session: &mut Session,
        _e: Option<&pingora::Error>,
        _ctx: &mut Self::CTX,
    ) {
        let response_code = session
            .response_written()
            .map_or(0, |resp| resp.status.as_u16());

        println!(
            "Request completed: {} {} -> {}",
            session.req_header().method,
            session.req_header().uri,
            response_code
        );
    }
}

fn main() {
    // Get upstream address from environment or use default
    // Docker: UPSTREAM_ADDR=upstream:80
    // Local:  defaults to 127.0.0.1:8080
    let upstream_addr =
        std::env::var("UPSTREAM_ADDR").unwrap_or_else(|_| "127.0.0.1:8080".to_string());

    // Initialize the server with default configuration
    let mut server = Server::new(None).unwrap();
    server.bootstrap();

    // Create our proxy service
    let proxy = LearningProxy::new(&upstream_addr);

    // Create the HTTP proxy service
    let mut proxy_service = http_proxy_service(&server.configuration, proxy);

    // Listen on port 6188
    proxy_service.add_tcp("0.0.0.0:6188");

    println!("Starting Pingora proxy on http://0.0.0.0:6188");
    println!("Forwarding requests to http://{}", upstream_addr);

    // Add the service and run
    server.add_service(proxy_service);
    server.run_forever();
}
