# Chapter 04: Load Balancing

## Learning Objectives

By the end of this chapter, you will:
- Understand different load balancing algorithms
- Implement round-robin load balancing
- Implement weighted load balancing
- Add sticky sessions (session affinity)
- Handle backend failures gracefully

---

## 4.1 Why Load Balancing?

When you have multiple backend servers, you need to distribute traffic:

```
                    ┌──────────────┐
                    │   Server 1   │
                    │  (healthy)   │
               ┌───▶│              │
               │    └──────────────┘
┌─────────┐    │    ┌──────────────┐
│  Proxy  │────┼───▶│   Server 2   │
│   (LB)  │────┤    │  (healthy)   │
└─────────┘    │    └──────────────┘
               │    ┌──────────────┐
               └───▶│   Server 3   │
                    │   (down!)    │
                    └──────────────┘
```

**Goals:**
- Distribute load evenly
- Avoid sending traffic to failed servers
- Handle server capacity differences
- Keep user sessions consistent (sometimes)

---

## 4.2 Load Balancing Algorithms

### Round Robin
Simplest approach - rotate through servers in order.

```
Request 1 → Server 1
Request 2 → Server 2
Request 3 → Server 3
Request 4 → Server 1  (back to start)
...
```

**Pros:** Simple, fair distribution
**Cons:** Ignores server capacity, no session affinity

### Weighted Round Robin
Some servers can handle more traffic than others.

```
Server 1: weight=3  (handles 3x traffic)
Server 2: weight=1
Server 3: weight=1

Request 1 → Server 1
Request 2 → Server 1
Request 3 → Server 1
Request 4 → Server 2
Request 5 → Server 3
Request 6 → Server 1  (cycle repeats)
```

### Least Connections
Send to the server with fewest active connections.

**Pros:** Adapts to slow requests
**Cons:** More complex, needs connection tracking

### IP Hash
Hash client IP to always route to same server.

```
hash("192.168.1.1") % 3 = 0 → Server 1
hash("192.168.1.2") % 3 = 2 → Server 3
```

**Pros:** Session affinity without cookies
**Cons:** Uneven distribution if IPs aren't diverse

### Random
Just pick randomly. Surprisingly effective!

**Pros:** Simple, no state needed
**Cons:** Can be uneven with few requests

---

## 4.3 Implementing Round Robin

```rust
use std::sync::atomic::{AtomicUsize, Ordering};

pub struct RoundRobinProxy {
    /// List of upstream servers
    upstreams: Vec<String>,
    /// Current index (atomic for thread safety)
    current: AtomicUsize,
}

impl RoundRobinProxy {
    pub fn new(upstreams: Vec<String>) -> Self {
        Self {
            upstreams,
            current: AtomicUsize::new(0),
        }
    }

    /// Get next upstream in rotation
    fn next_upstream(&self) -> &str {
        // Atomically increment and wrap around
        let idx = self.current.fetch_add(1, Ordering::Relaxed);
        let idx = idx % self.upstreams.len();
        &self.upstreams[idx]
    }
}

#[async_trait]
impl ProxyHttp for RoundRobinProxy {
    type CTX = RequestCtx;

    fn new_ctx(&self) -> Self::CTX {
        RequestCtx::new()
    }

    async fn upstream_peer(
        &self,
        _session: &mut Session,
        ctx: &mut Self::CTX,
    ) -> Result<Box<HttpPeer>> {
        let upstream = self.next_upstream();
        ctx.upstream = upstream.to_string();

        println!("Selected upstream: {}", upstream);

        let addr: std::net::SocketAddr = upstream.parse().unwrap();
        Ok(Box::new(HttpPeer::new(addr, false, String::new())))
    }
}
```

---

## 4.4 Implementing Weighted Selection

```rust
use rand::Rng;

pub struct WeightedProxy {
    /// (address, weight) pairs
    upstreams: Vec<(String, u32)>,
    /// Total weight (for random selection)
    total_weight: u32,
}

impl WeightedProxy {
    pub fn new(upstreams: Vec<(String, u32)>) -> Self {
        let total_weight = upstreams.iter().map(|(_, w)| w).sum();
        Self {
            upstreams,
            total_weight,
        }
    }

    /// Select upstream based on weights
    fn select_upstream(&self) -> &str {
        let mut rng = rand::thread_rng();
        let mut target = rng.gen_range(0..self.total_weight);

        for (addr, weight) in &self.upstreams {
            if target < *weight {
                return addr;
            }
            target -= weight;
        }

        // Fallback (shouldn't reach here)
        &self.upstreams[0].0
    }
}
```

**Example weights:**
```rust
let upstreams = vec![
    ("127.0.0.1:8081".to_string(), 5),  // 50% of traffic
    ("127.0.0.1:8082".to_string(), 3),  // 30% of traffic
    ("127.0.0.1:8083".to_string(), 2),  // 20% of traffic
];
```

---

## 4.5 IP-Based Sticky Sessions

Keep users on the same backend:

```rust
use std::collections::hash_map::DefaultHasher;
use std::hash::{Hash, Hasher};

pub struct StickyProxy {
    upstreams: Vec<String>,
}

impl StickyProxy {
    fn hash_ip(&self, ip: &str) -> usize {
        let mut hasher = DefaultHasher::new();
        ip.hash(&mut hasher);
        hasher.finish() as usize
    }

    fn select_by_ip(&self, client_ip: &str) -> &str {
        let hash = self.hash_ip(client_ip);
        let idx = hash % self.upstreams.len();
        &self.upstreams[idx]
    }
}

#[async_trait]
impl ProxyHttp for StickyProxy {
    // ...

    async fn upstream_peer(
        &self,
        session: &mut Session,
        _ctx: &mut Self::CTX,
    ) -> Result<Box<HttpPeer>> {
        // Get client IP from session
        // Note: In real scenarios, check X-Forwarded-For too
        let client_addr = session.client_addr()
            .map(|a| a.to_string())
            .unwrap_or_else(|| "unknown".to_string());

        let upstream = self.select_by_ip(&client_addr);

        println!("Client {} -> {}", client_addr, upstream);

        let addr: std::net::SocketAddr = upstream.parse().unwrap();
        Ok(Box::new(HttpPeer::new(addr, false, String::new())))
    }
}
```

---

## 4.6 Handling Backend Failures

What if a backend is down? Don't send traffic there!

### Simple Approach: Track Failures

```rust
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;

pub struct Backend {
    addr: String,
    healthy: AtomicBool,
}

pub struct FailoverProxy {
    backends: Vec<Arc<Backend>>,
    current: AtomicUsize,
}

impl FailoverProxy {
    /// Get next healthy backend
    fn next_healthy(&self) -> Option<&Arc<Backend>> {
        let start = self.current.load(Ordering::Relaxed);
        let len = self.backends.len();

        for i in 0..len {
            let idx = (start + i) % len;
            let backend = &self.backends[idx];

            if backend.healthy.load(Ordering::Relaxed) {
                self.current.store(idx + 1, Ordering::Relaxed);
                return Some(backend);
            }
        }

        None  // All backends unhealthy!
    }

    /// Mark a backend as unhealthy
    fn mark_unhealthy(&self, addr: &str) {
        for backend in &self.backends {
            if backend.addr == addr {
                backend.healthy.store(false, Ordering::Relaxed);
                println!("Marked {} as unhealthy", addr);
            }
        }
    }
}
```

### Using fail_to_connect Callback

Pingora provides a callback when upstream connection fails:

```rust
async fn fail_to_connect(
    &self,
    _session: &mut Session,
    peer: &HttpPeer,
    _ctx: &mut Self::CTX,
    e: Box<Error>,
) -> Box<Error> {
    // Connection to upstream failed!
    println!("Failed to connect to {}: {}", peer.address(), e);

    // Mark it unhealthy (implement your own logic)
    self.mark_unhealthy(&peer.address().to_string());

    // Return the error (or could retry with different peer)
    e
}
```

---

## 4.7 Using Pingora's Built-in Load Balancing

Pingora has a `pingora-load-balancing` crate:

```toml
[dependencies]
pingora-load-balancing = "0.7"
```

```rust
use pingora_load_balancing::{
    LoadBalancer,
    selection::RoundRobin,
    Backend,
};

// Create backends
let backends: Vec<Backend> = vec![
    Backend::new("127.0.0.1:8081").unwrap(),
    Backend::new("127.0.0.1:8082").unwrap(),
    Backend::new("127.0.0.1:8083").unwrap(),
];

// Create load balancer with round-robin selection
let lb = LoadBalancer::<RoundRobin>::from_backends(backends);

// Select a backend
let backend = lb.select(b"").unwrap();
```

Available selection algorithms:
- `RoundRobin`
- `Random`
- `Fnv` (hash-based)
- `Ketama` (consistent hashing)

---

## 4.8 Complete Load Balancer Example

See `src/main.rs` for a complete working example with:
- Round-robin load balancing
- Health tracking
- Failure handling
- Request logging per backend

---

## Exercises

### Exercise 4.1: Implement Random Selection
Modify the round-robin proxy to use random selection instead.

### Exercise 4.2: Add Weights
Extend your proxy to support weighted backends. Test that traffic distribution matches weights.

### Exercise 4.3: Failure Counting
Instead of immediately marking backends unhealthy, track failure counts. Only mark unhealthy after 3 consecutive failures.

### Exercise 4.4: Recovery Check
Add a mechanism to periodically re-check unhealthy backends and mark them healthy again if they recover.

### Exercise 4.5: Statistics
Add per-backend statistics:
- Total requests
- Failed requests
- Average response time

Print stats every 100 requests.

---

## Key Takeaways

1. **Round Robin** is simple and often good enough
2. **Weighted selection** handles heterogeneous backends
3. **IP hashing** provides session affinity
4. **Track failures** to avoid dead backends
5. Use **atomic operations** for thread-safe counters
6. Pingora has built-in load balancing via `pingora-load-balancing`

---

## Next Chapter

Load balancing helps distribute traffic, but how do we know backends are healthy?

Continue to [Chapter 05: Health Checks](../05-health-checks/README.md)
