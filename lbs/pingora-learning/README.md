# Pingora Learning Guide

> **Start Here:** See [COURSE.md](./COURSE.md) for the complete chapter-by-chapter curriculum!

## What is Pingora?

Pingora is Cloudflare's open-source Rust framework for building fast, reliable, and programmable network services (primarily HTTP proxies). It powers Cloudflare's global infrastructure, handling over 1 trillion requests daily.

## Course Structure

| Chapter | Topic | Key Concepts |
|---------|-------|--------------|
| [01](./chapters/01-fundamentals/) | Fundamentals | Proxy concepts, Pingora architecture |
| [02](./chapters/02-first-proxy/) | First Proxy | Build a working proxy from scratch |
| [03](./chapters/03-request-response/) | Request/Response | Header manipulation, routing |
| [04](./chapters/04-load-balancing/) | Load Balancing | Round-robin, weighted, failover |
| [05](./chapters/05-health-checks/) | Health Checks | Active/passive health monitoring |
| [06](./chapters/06-caching/) | Caching | Response caching, cache keys |
| [07](./chapters/07-observability/) | Observability | Metrics, logging, tracing |
| [08](./chapters/08-advanced/) | Advanced | TLS, graceful restart, tuning |

## Prerequisites

Before diving into Pingora, you should be comfortable with:

1. **Rust fundamentals** - ownership, borrowing, lifetimes, async/await
2. **HTTP protocol basics** - requests, responses, headers, methods
3. **Networking concepts** - TCP, TLS, load balancing, reverse proxies
4. **Async programming** - tokio runtime basics

## Learning Path

### Phase 1: Setup & Basics (Week 1)

- [x] Install Rust (if not already): `curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh`
- [ ] Clone Pingora repo: `git clone https://github.com/cloudflare/pingora.git`
- [ ] Read the official quick start guide
- [ ] Run the basic examples in the repo
- [ ] Understand the project structure

### Phase 2: Core Concepts (Week 2)

- [ ] Learn about `Server` and `Service` abstractions
- [ ] Understand the `ProxyHttp` trait
- [ ] Study request/response filters
- [ ] Explore upstream selection and load balancing
- [ ] Learn about connection pooling

### Phase 3: Build Projects (Week 3-4)

- [ ] Build a simple reverse proxy
- [ ] Add request/response modification
- [ ] Implement custom load balancing
- [ ] Add health checks
- [ ] Implement rate limiting

### Phase 4: Advanced Topics (Week 5+)

- [ ] TLS termination and configuration
- [ ] Caching strategies
- [ ] Metrics and observability
- [ ] Graceful shutdown and hot restarts
- [ ] Performance tuning

## Key Resources

### Official Resources
- GitHub: https://github.com/cloudflare/pingora
- User Guide: https://github.com/cloudflare/pingora/blob/main/docs/user_guide/index.md
- API Docs: https://docs.rs/pingora/latest/pingora/

### Blog Posts
- [How we built Pingora](https://blog.cloudflare.com/how-we-built-pingora-the-proxy-that-connects-cloudflare-to-the-internet/)
- [Open sourcing Pingora](https://blog.cloudflare.com/pingora-open-source/)

## Project Structure Overview

```
pingora/
├── pingora/           # Main crate (re-exports everything)
├── pingora-core/      # Core server and connection management
├── pingora-proxy/     # HTTP proxy implementation
├── pingora-cache/     # Caching layer
├── pingora-load-balancing/  # Load balancing algorithms
├── pingora-timeout/   # Timeout utilities
└── examples/          # Example implementations
```

## Quick Start Example

```rust
use pingora::prelude::*;
use pingora::proxy::{http_proxy_service, ProxyHttp, Session};
use async_trait::async_trait;

pub struct MyProxy;

#[async_trait]
impl ProxyHttp for MyProxy {
    type CTX = ();

    fn new_ctx(&self) -> Self::CTX {}

    async fn upstream_peer(
        &self,
        _session: &mut Session,
        _ctx: &mut Self::CTX,
    ) -> Result<Box<HttpPeer>> {
        let addr: std::net::SocketAddr = "127.0.0.1:8080".parse().unwrap();
        let peer = Box::new(HttpPeer::new(addr, false, String::new()));
        Ok(peer)
    }
}

fn main() {
    let mut server = Server::new(None).unwrap();
    server.bootstrap();

    let mut proxy = http_proxy_service(&server.configuration, MyProxy);
    proxy.add_tcp("0.0.0.0:6188");

    server.add_service(proxy);
    server.run_forever();
}
```

**Cargo.toml:**
```toml
[dependencies]
pingora = { version = "0.7", features = ["proxy"] }
async-trait = "0.1"
tokio = { version = "1", features = ["full"] }
```

## Comparison with Alternatives

| Feature | Pingora | NGINX | Envoy |
|---------|---------|-------|-------|
| Language | Rust | C | C++ |
| Programmability | High (Rust code) | Limited (config + Lua) | Medium (filters) |
| Memory Safety | Yes | No | No |
| Learning Curve | Steep | Moderate | Moderate |
| Ready to Use | No (library) | Yes | Yes |

## Notes

- Pingora requires Rust 1.72+ (6-month rolling MSRV policy)
- It's a library, not a standalone binary - you build your proxy
- Experimental Windows support added in v0.4
- Current version: 0.7.0 (as of early 2026)

## Running the Demo

### Option 1: Local Development

```bash
# Terminal 1: Start a test upstream server
python3 -m http.server 8080

# Terminal 2: Run the proxy
cargo run

# Terminal 3: Test it
curl http://localhost:6188
```

### Option 2: Docker (Recommended for Learning)

```bash
# Start everything with Docker Compose
docker compose up --build

# Test the proxy
curl http://localhost:6188

# See the headers added by Pingora
curl -v http://localhost:6188

# Test the API endpoint
curl http://localhost:6188/api/

# Stop everything
docker compose down
```

## Docker Setup

The project includes a complete Docker environment for learning:

```
pingora-learning/
├── Dockerfile              # Multi-stage build for the proxy
├── docker-compose.yml      # Orchestrates all services
└── docker/
    ├── upstream/           # Primary backend (nginx)
    │   ├── nginx.conf
    │   └── html/index.html
    └── upstream-api/       # Secondary backend (for load balancing)
        ├── nginx.conf
        └── html/index.html
```

### Services

| Service | Port | Description |
|---------|------|-------------|
| `pingora-proxy` | 6188 | Your Pingora proxy |
| `upstream` | internal | Primary nginx backend |
| `upstream-api` | internal | Secondary nginx backend |
| `httpbin` | internal | API testing service |

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `UPSTREAM_ADDR` | `127.0.0.1:8080` | Upstream server address |
| `RUST_LOG` | `info` | Log level |

### Docker Learning Exercises

1. **Basic Proxy**: Run `docker compose up` and test with `curl`
2. **Inspect Headers**: Use `curl -v` to see `X-Proxied-By` header
3. **Modify Code**: Edit `src/main.rs`, rebuild with `docker compose up --build`
4. **Switch Upstreams**: Change `UPSTREAM_ADDR` to `upstream-api:80` in docker-compose.yml
5. **Use httpbin**: Change `UPSTREAM_ADDR` to `httpbin:80` for API testing
