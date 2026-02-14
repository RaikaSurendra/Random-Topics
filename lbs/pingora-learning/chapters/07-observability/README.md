# Chapter 07: Observability

## Learning Objectives

By the end of this chapter, you will:
- Implement structured logging
- Export Prometheus metrics
- Add distributed tracing
- Build a health dashboard

---

## 7.1 The Three Pillars of Observability

```
┌─────────────────────────────────────────────────────────┐
│                    OBSERVABILITY                         │
├──────────────────┬──────────────────┬──────────────────┤
│      LOGS        │     METRICS      │     TRACES       │
│                  │                  │                  │
│ What happened?   │ How much?        │ Where did it go? │
│                  │                  │                  │
│ • Request/error  │ • Request rate   │ • Request path   │
│   details        │ • Latency p99    │   through system │
│ • Debug info     │ • Error rate     │ • Cross-service  │
│                  │ • CPU/Memory     │   correlation    │
└──────────────────┴──────────────────┴──────────────────┘
```

---

## 7.2 Structured Logging

### Bad: Unstructured logs
```
Request to /api/users completed in 45ms
Error processing request
User logged in successfully
```

### Good: Structured logs
```json
{"level":"info","ts":"2024-01-15T10:30:00Z","method":"GET","path":"/api/users","status":200,"latency_ms":45,"request_id":"abc123"}
{"level":"error","ts":"2024-01-15T10:30:01Z","method":"POST","path":"/api/orders","error":"connection refused","upstream":"10.0.0.5:8080"}
```

### Implementation with tracing

```rust
use tracing::{info, error, warn, instrument};
use tracing_subscriber;

// Initialize at startup
fn init_logging() {
    tracing_subscriber::fmt()
        .json()                        // JSON format
        .with_max_level(Level::INFO)   // Log level
        .init();
}

// In your proxy
#[instrument(skip(session, ctx), fields(request_id = %ctx.request_id))]
async fn upstream_peer(&self, session: &mut Session, ctx: &mut Self::CTX) -> Result<Box<HttpPeer>> {
    let path = session.req_header().uri.path();

    info!(
        method = %session.req_header().method,
        path = %path,
        "Routing request"
    );

    // ... routing logic
}

async fn logging(&self, session: &mut Session, error: Option<&Error>, ctx: &mut Self::CTX) {
    let elapsed = ctx.start_time.elapsed();

    if let Some(e) = error {
        error!(
            request_id = %ctx.request_id,
            path = %session.req_header().uri.path(),
            error = %e,
            latency_ms = %elapsed.as_millis(),
            "Request failed"
        );
    } else {
        info!(
            request_id = %ctx.request_id,
            path = %session.req_header().uri.path(),
            status = %session.response_written().map(|r| r.status.as_u16()).unwrap_or(0),
            latency_ms = %elapsed.as_millis(),
            "Request completed"
        );
    }
}
```

---

## 7.3 Prometheus Metrics

Pingora has built-in Prometheus support:

```rust
use prometheus::{
    Counter, Histogram, HistogramOpts, IntCounterVec, IntGauge, Opts, Registry,
    histogram_opts, opts,
};

lazy_static! {
    // Request counter
    static ref REQUESTS_TOTAL: IntCounterVec = IntCounterVec::new(
        opts!("proxy_requests_total", "Total requests"),
        &["method", "status", "backend"]
    ).unwrap();

    // Latency histogram
    static ref REQUEST_LATENCY: Histogram = Histogram::with_opts(
        histogram_opts!(
            "proxy_request_latency_seconds",
            "Request latency in seconds",
            vec![0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0]
        )
    ).unwrap();

    // Active connections
    static ref ACTIVE_CONNECTIONS: IntGauge = IntGauge::new(
        "proxy_active_connections",
        "Number of active connections"
    ).unwrap();
}

// Record metrics
async fn logging(&self, session: &mut Session, error: Option<&Error>, ctx: &mut Self::CTX) {
    let status = session.response_written()
        .map(|r| r.status.as_u16().to_string())
        .unwrap_or_else(|| "0".to_string());

    REQUESTS_TOTAL
        .with_label_values(&[
            session.req_header().method.as_str(),
            &status,
            &ctx.upstream_name,
        ])
        .inc();

    REQUEST_LATENCY.observe(ctx.start_time.elapsed().as_secs_f64());
}
```

### Essential Metrics

| Metric | Type | Labels | Purpose |
|--------|------|--------|---------|
| `requests_total` | Counter | method, status, backend | Traffic volume |
| `request_latency` | Histogram | backend | Performance |
| `errors_total` | Counter | type, backend | Error rate |
| `active_connections` | Gauge | - | Concurrency |
| `backend_health` | Gauge | backend | Health status |

### Expose Metrics Endpoint

```rust
// Add a metrics service
async fn metrics_handler() -> impl IntoResponse {
    let encoder = prometheus::TextEncoder::new();
    let metrics = prometheus::gather();
    encoder.encode_to_string(&metrics).unwrap()
}

// Or use Pingora's built-in prometheus service
let mut prometheus_service = prometheus_service(&server.configuration);
prometheus_service.add_tcp("0.0.0.0:9090");
server.add_service(prometheus_service);
```

---

## 7.4 Distributed Tracing

Track requests across services:

```
Client → Proxy → Auth Service → User Service → Database
   └── trace-id: abc123 ──────────────────────────────┘
```

### OpenTelemetry Integration

```rust
use opentelemetry::{global, trace::Tracer};
use tracing_opentelemetry::OpenTelemetryLayer;

fn init_tracing() {
    let tracer = opentelemetry_jaeger::new_pipeline()
        .with_service_name("pingora-proxy")
        .install_simple()
        .unwrap();

    let telemetry = OpenTelemetryLayer::new(tracer);

    tracing_subscriber::registry()
        .with(telemetry)
        .init();
}

// Propagate trace context to upstream
async fn upstream_request_filter(
    &self,
    session: &mut Session,
    upstream_request: &mut RequestHeader,
    ctx: &mut Self::CTX,
) -> Result<()> {
    // Get current span context
    let cx = Span::current().context();

    // Inject into headers (W3C Trace Context format)
    global::get_text_map_propagator(|propagator| {
        propagator.inject_context(&cx, &mut HeaderInjector(upstream_request));
    });

    Ok(())
}
```

---

## 7.5 Health Dashboard

Combine metrics into a useful dashboard:

### Key Panels

```
┌─────────────────────────────────────────────────────────┐
│  Request Rate              │  Error Rate                │
│  [====== 1.2k/s ========]  │  [= 0.5% ===============]  │
├────────────────────────────┼───────────────────────────┤
│  P50 Latency: 12ms         │  P99 Latency: 145ms       │
│  P95 Latency: 45ms         │  Max Latency: 890ms       │
├────────────────────────────┴───────────────────────────┤
│  Backend Health                                         │
│  [●] backend-1: healthy (234 req/s)                    │
│  [●] backend-2: healthy (228 req/s)                    │
│  [○] backend-3: unhealthy (0 req/s)                    │
├─────────────────────────────────────────────────────────┤
│  Top Paths by Request Count                             │
│  1. /api/users     (45%)                               │
│  2. /api/orders    (30%)                               │
│  3. /static/app.js (15%)                               │
└─────────────────────────────────────────────────────────┘
```

### Grafana Queries

```promql
# Request rate
rate(proxy_requests_total[5m])

# Error rate
rate(proxy_requests_total{status=~"5.."}[5m]) / rate(proxy_requests_total[5m])

# P99 latency
histogram_quantile(0.99, rate(proxy_request_latency_seconds_bucket[5m]))

# Requests per backend
sum by (backend) (rate(proxy_requests_total[5m]))
```

---

## 7.6 Alerting

Set up alerts for critical conditions:

```yaml
# Prometheus alerting rules
groups:
  - name: proxy
    rules:
      - alert: HighErrorRate
        expr: rate(proxy_requests_total{status=~"5.."}[5m]) / rate(proxy_requests_total[5m]) > 0.05
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High error rate (> 5%)"

      - alert: HighLatency
        expr: histogram_quantile(0.99, rate(proxy_request_latency_seconds_bucket[5m])) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "P99 latency > 1s"

      - alert: BackendDown
        expr: proxy_backend_health == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Backend {{ $labels.backend }} is down"
```

---

## Exercises

### Exercise 7.1: Request Logging
Add JSON logging for every request with: method, path, status, latency, upstream.

### Exercise 7.2: Custom Metrics
Add a metric for cache hit rate: `cache_hits_total` / `cache_requests_total`.

### Exercise 7.3: Request ID Propagation
Generate a request ID if not present, propagate to backend in `X-Request-ID` header.

### Exercise 7.4: Error Categorization
Create separate error counters for: connection errors, timeout errors, 4xx errors, 5xx errors.

---

## Key Takeaways

1. **Structured logs** make debugging easier
2. **Prometheus metrics** enable dashboards and alerts
3. **Distributed tracing** tracks requests across services
4. Monitor the **four golden signals**: latency, traffic, errors, saturation
5. Set up **alerts** for critical thresholds

---

## Next Chapter

You've built a solid proxy! Let's cover advanced topics.

Continue to [Chapter 08: Advanced Topics](../08-advanced/README.md)
