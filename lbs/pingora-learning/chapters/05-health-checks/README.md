# Chapter 05: Health Checks

## Learning Objectives

By the end of this chapter, you will:
- Understand active vs passive health checks
- Implement HTTP health check probes
- Configure health check intervals and thresholds
- Handle graceful degradation

---

## 5.1 Why Health Checks?

Without health checks, your load balancer might send traffic to dead servers:

```
Request → Proxy → Dead Server → Timeout → Bad User Experience
```

With health checks:

```
Health Check → Dead Server → Mark Unhealthy
Request → Proxy → (skips dead server) → Healthy Server → Fast Response!
```

---

## 5.2 Types of Health Checks

### Passive (Reactive)
- Monitor actual request failures
- Mark unhealthy after N consecutive failures
- Pros: No extra traffic, reflects real issues
- Cons: Slow to detect, users experience failures

### Active (Proactive)
- Send periodic probe requests (e.g., GET /health)
- Mark unhealthy if probe fails
- Pros: Fast detection, no user impact
- Cons: Extra traffic, false positives possible

### Best Practice: Use Both!
- Passive for immediate reaction to failures
- Active for detection when traffic is low

---

## 5.3 Implementing Active Health Checks

```rust
use std::time::Duration;
use tokio::time::interval;

pub struct HealthChecker {
    backends: Vec<Arc<Backend>>,
    check_interval: Duration,
    timeout: Duration,
    healthy_threshold: u32,    // Successes needed to mark healthy
    unhealthy_threshold: u32,  // Failures needed to mark unhealthy
}

impl HealthChecker {
    /// Start the health check loop
    pub async fn run(&self) {
        let mut ticker = interval(self.check_interval);

        loop {
            ticker.tick().await;

            for backend in &self.backends {
                let healthy = self.check_backend(backend).await;

                if healthy {
                    backend.record_success();
                    if backend.consecutive_successes() >= self.healthy_threshold {
                        backend.mark_healthy();
                    }
                } else {
                    backend.record_failure();
                    if backend.consecutive_failures() >= self.unhealthy_threshold {
                        backend.mark_unhealthy();
                    }
                }
            }
        }
    }

    /// Check a single backend
    async fn check_backend(&self, backend: &Backend) -> bool {
        let client = reqwest::Client::builder()
            .timeout(self.timeout)
            .build()
            .unwrap();

        let url = format!("http://{}/health", backend.addr);

        match client.get(&url).send().await {
            Ok(response) => response.status().is_success(),
            Err(_) => false,
        }
    }
}
```

---

## 5.4 Health Check Endpoint Design

Your backends should expose a `/health` endpoint:

```rust
// On your backend server
async fn health_check() -> impl IntoResponse {
    // Check database connection
    // Check cache connection
    // Check disk space
    // etc.

    if all_systems_healthy {
        (StatusCode::OK, "OK")
    } else {
        (StatusCode::SERVICE_UNAVAILABLE, "Unhealthy")
    }
}
```

### Good Health Check Practices

| Do | Don't |
|----|-------|
| Check critical dependencies | Just return 200 always |
| Keep it fast (<100ms) | Do expensive operations |
| Return meaningful status | Return HTML pages |
| Include version info | Expose sensitive data |

Example response:
```json
{
  "status": "healthy",
  "version": "1.2.3",
  "checks": {
    "database": "ok",
    "cache": "ok"
  }
}
```

---

## 5.5 Configurable Thresholds

Don't mark unhealthy on first failure - use thresholds:

```rust
pub struct Backend {
    addr: String,
    healthy: AtomicBool,
    consecutive_failures: AtomicU32,
    consecutive_successes: AtomicU32,
}

impl Backend {
    pub fn record_failure(&self) {
        self.consecutive_successes.store(0, Ordering::Relaxed);
        self.consecutive_failures.fetch_add(1, Ordering::Relaxed);
    }

    pub fn record_success(&self) {
        self.consecutive_failures.store(0, Ordering::Relaxed);
        self.consecutive_successes.fetch_add(1, Ordering::Relaxed);
    }
}
```

### Recommended Settings

| Setting | Value | Reason |
|---------|-------|--------|
| Check interval | 5-10 seconds | Balance freshness vs overhead |
| Unhealthy threshold | 2-3 failures | Avoid false positives |
| Healthy threshold | 2-3 successes | Ensure stable recovery |
| Timeout | 2-5 seconds | Don't wait too long |

---

## 5.6 Graceful Degradation

What if ALL backends are unhealthy?

### Option 1: Fail Open
Return error to client immediately:
```rust
if no_healthy_backends {
    return Err(Error::new(ErrorType::HTTPStatus(503)));
}
```

### Option 2: Try Anyway
Send to unhealthy backend, hope it recovers:
```rust
let backend = self.select_backend()
    .or_else(|| self.any_backend());  // Fallback
```

### Option 3: Circuit Breaker
Stop trying for a period, then retry:
```rust
if self.circuit_open {
    if self.last_attempt.elapsed() < self.cooldown {
        return Err(Error::new(ErrorType::HTTPStatus(503)));
    }
    // Cooldown passed, try again
    self.circuit_open = false;
}
```

---

## 5.7 Complete Example

See `src/main.rs` for a complete implementation with:
- Background health check task
- Configurable intervals and thresholds
- Combined passive + active checking
- Graceful degradation

---

## Exercises

### Exercise 5.1: Add Health Endpoint
Add a `/health` endpoint to your proxy itself (not backends) that returns status of all backends.

### Exercise 5.2: Weighted Recovery
When a backend recovers, gradually increase its weight instead of immediately sending full traffic.

### Exercise 5.3: Health History
Keep last 10 health check results per backend. Report health as percentage (e.g., "80% healthy in last 10 checks").

### Exercise 5.4: TCP Health Check
Implement TCP-only health check (just connect, don't send HTTP request). Useful for non-HTTP backends.

---

## Key Takeaways

1. **Active checks** detect issues before users do
2. **Passive checks** react to real failures
3. Use **thresholds** to avoid flapping
4. Design good **health endpoints** on backends
5. Plan for **graceful degradation**

---

## Next Chapter

Your proxy is resilient. Now let's make it fast with caching!

Continue to [Chapter 06: Caching](../06-caching/README.md)
