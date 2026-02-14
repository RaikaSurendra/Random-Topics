# Chapter 08: Metrics & Prometheus

## Learning Objectives

1. Understand the three types of metrics: counters, gauges, histograms
2. Implement Prometheus text format export
3. Design efficient metrics collection with minimal overhead
4. Create observable production systems

## Why Metrics?

```
WITHOUT METRICS
═══════════════════════════════════════════════════════════════

User: "The site is slow"
You:  "Let me check... it looks fine to me"
User: "It's been slow for 3 hours"
You:  "I have no data to investigate"


WITH METRICS
═══════════════════════════════════════════════════════════════

Alert: "P99 latency increased from 50ms to 500ms at 14:32"
Dashboard: Shows Backend-2 health check failures
Logs: "Backend-2 disk full at 14:30"
Resolution: Clear disk, latency returns to normal
```

## Metric Types

### Counters (Monotonically Increasing)

```c
// Good for: Requests, errors, bytes transferred
metrics_counter_inc(m, "requests_total", labels);
metrics_counter_add(m, "bytes_sent", 1024, labels);

// Output:
// requests_total{backend="server1"} 12847
// bytes_sent{backend="server1"} 4521893
```

### Gauges (Can Go Up or Down)

```c
// Good for: Connections, queue depth, temperature
metrics_gauge_set(m, "connections_active", 42, labels);
metrics_gauge_inc(m, "connections_active", labels);
metrics_gauge_dec(m, "connections_active", labels);

// Output:
// connections_active{backend="server1"} 42
```

### Histograms (Distributions)

```c
// Good for: Latency, request sizes
metrics_histogram_observe(m, "request_duration_seconds", 0.025, labels);

// Output:
// request_duration_seconds_bucket{le="0.01"} 423
// request_duration_seconds_bucket{le="0.05"} 1847
// request_duration_seconds_bucket{le="0.1"} 2103
// request_duration_seconds_bucket{le="+Inf"} 2156
// request_duration_seconds_sum 47.32
// request_duration_seconds_count 2156
```

## Standard Load Balancer Metrics

```c
// Register default metrics
metrics_register_lb_defaults(m);

// This creates:
// - lb_requests_total (counter)
// - lb_requests_failed_total (counter)
// - lb_connections_active (gauge)
// - lb_backends_healthy (gauge)
// - lb_request_duration_seconds (histogram)
// - lb_bytes_received_total (counter)
// - lb_bytes_sent_total (counter)
// - lb_pool_hits_total (counter)
// - lb_pool_misses_total (counter)
```

## Prometheus Format

```
# HELP lb_requests_total Total HTTP requests
# TYPE lb_requests_total counter
lb_requests_total{backend="server1",status="200"} 12847
lb_requests_total{backend="server1",status="500"} 23
lb_requests_total{backend="server2",status="200"} 11293

# HELP lb_connections_active Current active connections
# TYPE lb_connections_active gauge
lb_connections_active 42

# HELP lb_request_duration_seconds Request latency distribution
# TYPE lb_request_duration_seconds histogram
lb_request_duration_seconds_bucket{le="0.01"} 423
lb_request_duration_seconds_bucket{le="0.05"} 1847
lb_request_duration_seconds_bucket{le="0.1"} 2103
lb_request_duration_seconds_bucket{le="+Inf"} 2156
lb_request_duration_seconds_sum 47.32
lb_request_duration_seconds_count 2156
```

## Exposing /metrics Endpoint

```c
void handle_metrics_request(int client_fd) {
    metrics_expose(lb->metrics, client_fd);
}

// In request handler:
if (strstr(request, "GET /metrics")) {
    handle_metrics_request(conn->client_fd);
    return;
}
```

## Prometheus Configuration

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'loadbalancer'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 15s
```

## Grafana Dashboard

Key panels to create:
1. **Request Rate**: `rate(lb_requests_total[5m])`
2. **Error Rate**: `rate(lb_requests_failed_total[5m]) / rate(lb_requests_total[5m])`
3. **P99 Latency**: `histogram_quantile(0.99, rate(lb_request_duration_seconds_bucket[5m]))`
4. **Active Connections**: `lb_connections_active`
5. **Pool Hit Rate**: `rate(lb_pool_hits_total[5m]) / (rate(lb_pool_hits_total[5m]) + rate(lb_pool_misses_total[5m]))`

## Exercises

1. Add per-backend latency histograms
2. Implement metric cardinality limits (prevent label explosion)
3. Add Grafana dashboard JSON export
4. Implement push gateway support for short-lived processes

## Key Takeaways

1. **Counters** for things that only go up
2. **Gauges** for current state
3. **Histograms** for latency distributions (P50, P99)
4. Use **labels** for dimensions, not metric names
5. Expose **/metrics** for Prometheus scraping
