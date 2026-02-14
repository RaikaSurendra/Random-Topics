# Chapter 04: Traefik vs Nginx Comparison

## Learning Objectives

1. Understand fundamental architectural differences
2. Compare configuration approaches
3. Evaluate performance characteristics
4. Determine use case suitability
5. Run side-by-side comparison

## Architecture Comparison

### Traefik Architecture

```
┌────────────────────────────────────────────────────────────┐
│                         TRAEFIK                             │
│                                                             │
│  ┌─────────────┐    ┌──────────────┐    ┌──────────────┐  │
│  │  Providers  │───▶│   Dynamic    │───▶│   Runtime    │  │
│  │ (Docker/K8s)│    │   Config     │    │   Router     │  │
│  └─────────────┘    └──────────────┘    └──────────────┘  │
│         │                  │                    │          │
│         ▼                  ▼                    ▼          │
│  ┌─────────────────────────────────────────────────────┐  │
│  │              Auto-Discovery & Hot Reload             │  │
│  │                   (No Restart Needed)                │  │
│  └─────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────┘
```

### Nginx Architecture

```
┌────────────────────────────────────────────────────────────┐
│                          NGINX                              │
│                                                             │
│  ┌─────────────┐    ┌──────────────┐    ┌──────────────┐  │
│  │   Config    │───▶│    Master    │───▶│   Worker     │  │
│  │   Files     │    │   Process    │    │  Processes   │  │
│  └─────────────┘    └──────────────┘    └──────────────┘  │
│         │                  │                    │          │
│         ▼                  ▼                    ▼          │
│  ┌─────────────────────────────────────────────────────┐  │
│  │           Static Config (Reload Required)            │  │
│  │               nginx -s reload                        │  │
│  └─────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────┘
```

## Feature Comparison

| Feature | Traefik | Nginx |
|---------|---------|-------|
| **Configuration** | Dynamic (Labels/API) | Static (Files) |
| **Service Discovery** | Native (Docker/K8s) | Manual or scripts |
| **Hot Reload** | Automatic | Manual (`nginx -s reload`) |
| **Let's Encrypt** | Built-in | Requires certbot |
| **Dashboard** | Built-in | Nginx Plus only |
| **Learning Curve** | Moderate | Steeper |
| **Performance** | Good | Excellent |
| **Memory Usage** | Higher | Lower |
| **HTTP/2, HTTP/3** | Yes | Yes |
| **WebSockets** | Yes | Yes |
| **gRPC** | Yes | Yes |
| **TCP/UDP** | Yes | Yes |

## Configuration Comparison

### Same Goal: Route `/api` to Backend Service

#### Traefik (Docker Labels)

```yaml
# docker-compose.yml
services:
  api:
    image: my-api
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.api.rule=PathPrefix(`/api`)"
      - "traefik.http.services.api.loadbalancer.server.port=3000"
```

#### Nginx (Config File)

```nginx
# nginx.conf
upstream api_backend {
    server api:3000;
}

server {
    listen 80;

    location /api {
        proxy_pass http://api_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### Adding a New Service

#### Traefik
```yaml
# Just add labels to new service - Traefik auto-discovers
new-service:
  image: new-service
  labels:
    - "traefik.enable=true"
    - "traefik.http.routers.new.rule=Host(`new.example.com`)"
```

#### Nginx
```nginx
# Edit nginx.conf, add upstream and location
upstream new_backend {
    server new-service:3000;
}

server {
    location /new {
        proxy_pass http://new_backend;
    }
}
# Then reload: nginx -s reload
```

## Running the Comparison

```bash
# Start both Traefik and Nginx setups
docker-compose up -d

# Access via Traefik
curl http://localhost:8081        # Traefik dashboard
curl http://localhost:8000        # App via Traefik

# Access via Nginx
curl http://localhost:8082        # Nginx status (if enabled)
curl http://localhost:8001        # App via Nginx

# Compare response times
time curl http://localhost:8000
time curl http://localhost:8001

# Load test both
hey -n 1000 -c 50 http://localhost:8000/
hey -n 1000 -c 50 http://localhost:8001/
```

## Performance Benchmarks

### Test Methodology

```bash
# Using 'hey' load testing tool
hey -n 10000 -c 100 http://localhost:<port>/

# Metrics to compare:
# - Requests/sec
# - Average latency
# - 99th percentile latency
# - Error rate
```

### Typical Results

| Metric | Traefik | Nginx |
|--------|---------|-------|
| Requests/sec | ~15,000 | ~20,000 |
| Avg Latency | ~6ms | ~4ms |
| p99 Latency | ~15ms | ~10ms |
| Memory (idle) | ~50MB | ~10MB |
| Memory (load) | ~100MB | ~30MB |

*Note: Results vary based on configuration and workload*

## Use Case Suitability

### Choose Traefik When:

1. **Microservices Architecture**
   - Services scale up/down frequently
   - Docker/Kubernetes native environment
   - Need automatic service discovery

2. **DevOps Workflow**
   - GitOps deployment model
   - Configuration as code (labels)
   - Rapid iteration needed

3. **SSL/TLS Management**
   - Need automatic Let's Encrypt
   - Multiple domains
   - Certificate rotation

4. **Monitoring Built-in**
   - Need dashboard out of the box
   - Metrics and tracing required

### Choose Nginx When:

1. **High Performance Critical**
   - Maximum requests/second needed
   - Minimal latency required
   - Static content serving

2. **Traditional Infrastructure**
   - VMs or bare metal
   - Services don't change often
   - Team knows Nginx well

3. **Advanced Features**
   - Complex rewrite rules
   - Lua scripting needed
   - Caching requirements

4. **Resource Constrained**
   - Limited memory
   - Edge deployments
   - IoT scenarios

## Hybrid Approach

Many production setups use BOTH:

```
                    ┌─────────────┐
    Internet ──────▶│    Nginx    │  (Edge: SSL termination, caching)
                    └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │   Traefik   │  (Service mesh: routing, discovery)
                    └──────┬──────┘
                           │
              ┌────────────┼────────────┐
              ▼            ▼            ▼
         ┌────────┐   ┌────────┐   ┌────────┐
         │Service │   │Service │   │Service │
         └────────┘   └────────┘   └────────┘
```

## Migration Paths

### Nginx → Traefik

1. Start with Traefik alongside Nginx
2. Migrate services one at a time
3. Update DNS/routing gradually
4. Decommission Nginx when complete

### Traefik → Nginx

1. Export Traefik routes to Nginx config
2. Set up Nginx upstream blocks
3. Switch traffic at load balancer
4. Remove Traefik integration

## Summary Decision Matrix

| Scenario | Recommendation | Reason |
|----------|---------------|--------|
| Kubernetes cluster | **Traefik** | Native integration |
| Docker Swarm | **Traefik** | Built-in discovery |
| Static website | **Nginx** | Performance, caching |
| Legacy VMs | **Nginx** | Simpler setup |
| Microservices | **Traefik** | Dynamic routing |
| High traffic API | **Nginx** | Raw performance |
| Startup/MVP | **Traefik** | Faster setup |
| Enterprise (existing) | **Nginx** | Team familiarity |

## Key Takeaways

1. **Traefik**: Best for dynamic, container-native environments
2. **Nginx**: Best for static configs, maximum performance
3. **Both**: Can work together in hybrid architectures
4. **Decision**: Based on team skills, infrastructure, and requirements

## Exercises

### Exercise 1: Config Complexity

Count the lines needed to add a new service with:
- Load balancing (3 backends)
- Health checks
- SSL termination
- Rate limiting

### Exercise 2: Failure Recovery

1. Stop a backend service
2. Observe how each handles the failure
3. Restart the service
4. Measure recovery time

### Exercise 3: Resource Monitoring

```bash
# Monitor resource usage
docker stats traefik-compare nginx-compare
```
