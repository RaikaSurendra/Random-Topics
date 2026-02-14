// Chapter 02: Your First Proxy
//
// This is the simplest possible Pingora proxy.
// It forwards ALL requests to a single upstream server.
//
// Run with: cargo run
// Test with: curl http://localhost:6188

use async_trait::async_trait;
use pingora::prelude::*;
use pingora::proxy::{http_proxy_service, ProxyHttp, Session};

/// Our proxy configuration
/// This struct holds settings that apply to ALL requests
pub struct FirstProxy {
    /// The upstream server address (IP:port)
    upstream: String,
}

impl FirstProxy {
    pub fn new(upstream: &str) -> Self {
        println!("[INIT] Creating proxy with upstream: {}", upstream);
        Self {
            upstream: upstream.to_string(),
        }
    }
}

/// Per-request context
/// A fresh instance is created for each incoming request
pub struct RequestCtx {
    // Empty for now - we'll add fields in Chapter 03
}

#[async_trait]
impl ProxyHttp for FirstProxy {
    type CTX = RequestCtx;

    /// Called for each new request
    fn new_ctx(&self) -> Self::CTX {
        println!("[REQUEST] New request received");
        RequestCtx {}
    }

    /// Choose which upstream server to use
    /// This is the ONLY required method in ProxyHttp
    async fn upstream_peer(
        &self,
        _session: &mut Session,
        _ctx: &mut Self::CTX,
    ) -> Result<Box<HttpPeer>> {
        println!("[UPSTREAM] Forwarding to: {}", self.upstream);

        // Parse the address string into a SocketAddr
        let addr: std::net::SocketAddr = self
            .upstream
            .parse()
            .expect("Invalid upstream address - use format IP:PORT");

        // Create an HTTP peer (connection to upstream)
        // Parameters:
        //   - addr: The socket address
        //   - tls: false for HTTP, true for HTTPS
        //   - sni: TLS server name (empty for non-TLS)
        let peer = HttpPeer::new(addr, false, String::new());

        Ok(Box::new(peer))
    }

    /// Called after the request completes (success or failure)
    async fn logging(
        &self,
        session: &mut Session,
        _e: Option<&pingora::Error>,
        _ctx: &mut Self::CTX,
    ) {
        // Get response status code (if any)
        let status = session
            .response_written()
            .map(|r| r.status.as_u16())
            .unwrap_or(0);

        println!(
            "[COMPLETE] {} {} -> {}",
            session.req_header().method,
            session.req_header().uri,
            status
        );
    }
}

fn main() {
    // Read upstream from environment or use default
    let upstream = std::env::var("UPSTREAM_ADDR")
        .unwrap_or_else(|_| "127.0.0.1:8080".to_string());

    // Create and configure the server
    let mut server = Server::new(None).unwrap();
    server.bootstrap();

    // Create our proxy
    let proxy = FirstProxy::new(&upstream);

    // Create the HTTP proxy service
    let mut service = http_proxy_service(&server.configuration, proxy);

    // Listen on port 6188
    service.add_tcp("0.0.0.0:6188");

    // Add service and run
    server.add_service(service);

    println!("=========================================");
    println!("  First Proxy - Chapter 02");
    println!("=========================================");
    println!("  Listening on: http://0.0.0.0:6188");
    println!("  Upstream:     http://{}", upstream);
    println!("=========================================");
    println!("  Test with: curl http://localhost:6188");
    println!("=========================================");

    server.run_forever();
}
