# Chapter 01: Traefik Basics

## Learning Objectives

1. Understand Traefik's architecture
2. Learn about Entry Points, Routers, and Services
3. Configure Traefik with Docker provider
4. Access the Traefik Dashboard
5. Understand basic logging and debugging

## Key Concepts

### What is Traefik?

Traefik is a modern HTTP reverse proxy and load balancer designed for microservices. Unlike traditional proxies, Traefik:

- **Auto-discovers services** from Docker, Kubernetes, etc.
- **Dynamic configuration** - no restarts needed
- **Native Let's Encrypt** support
- **Built-in dashboard** for monitoring

### Core Components

```
┌─────────────────────────────────────────────────────────────┐
│                        TRAEFIK                               │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐  │
│  │ Entry Point  │───▶│    Router    │───▶│   Service    │  │
│  │  (Port 80)   │    │ (Path/Host)  │    │(Load Balance)│  │
│  └──────────────┘    └──────────────┘    └──────────────┘  │
│                             │                               │
│                      ┌──────▼──────┐                       │
│                      │ Middleware  │                       │
│                      │ (Optional)  │                       │
│                      └─────────────┘                       │
└─────────────────────────────────────────────────────────────┘
```

1. **Entry Points**: Network entry points (ports) into Traefik
2. **Routers**: Connect requests to services based on rules
3. **Services**: Define how to reach actual applications
4. **Middlewares**: Modify requests/responses (auth, headers, etc.)

### Provider System

Traefik uses "providers" to discover services:

| Provider | Use Case |
|----------|----------|
| Docker | Container orchestration |
| Kubernetes | K8s ingress |
| File | Static configuration |
| Consul/etcd | Service discovery |

## Running This Chapter

```bash
# Start the basic setup
docker-compose up -d

# View logs
docker-compose logs -f traefik

# Access endpoints
curl http://localhost        # App response
curl http://localhost:8080   # Dashboard (browser recommended)

# Stop
docker-compose down
```

## Configuration Explained

### Static Configuration (traefik.yml)

```yaml
# Entry points define where Traefik listens
entryPoints:
  web:
    address: ":80"    # HTTP
  websecure:
    address: ":443"   # HTTPS

# API/Dashboard
api:
  dashboard: true
  insecure: true      # Only for development!

# Providers tell Traefik where to find services
providers:
  docker:
    exposedByDefault: false  # Require explicit enable
```

### Dynamic Configuration (Docker Labels)

```yaml
services:
  app:
    labels:
      - "traefik.enable=true"                              # Enable for Traefik
      - "traefik.http.routers.app.rule=Host(`localhost`)"  # Routing rule
      - "traefik.http.services.app.loadbalancer.server.port=5000"  # Container port
```

## Exercises

### Exercise 1: Explore the Dashboard

1. Open http://localhost:8080 in your browser
2. Navigate to "HTTP" → "Routers" - see your configured router
3. Check "HTTP" → "Services" - see your service
4. View "HTTP" → "Middlewares" - currently empty

### Exercise 2: Test Routing Rules

```bash
# This works (matches Host rule)
curl -H "Host: localhost" http://localhost

# This fails (no matching rule)
curl -H "Host: example.com" http://localhost
```

### Exercise 3: Check Logs

```bash
# Watch Traefik access logs
docker-compose logs -f traefik

# In another terminal, make requests
curl http://localhost
```

## Log Output Explained

```
traefik  | 192.168.65.1 - - [02/Feb/2024:10:30:45 +0000] "GET / HTTP/1.1" 200 523 "-" "curl/8.1.2" 1 "app@docker" "http://172.18.0.2:5000" 15ms
           │              │                                    │   │                    │                    │              │
           │              │                                    │   │                    │                    │              └─ Response time
           │              │                                    │   │                    │                    └─ Backend server
           │              │                                    │   │                    └─ Router/Service used
           │              │                                    │   └─ Response size
           │              │                                    └─ HTTP status
           │              └─ Timestamp
           └─ Client IP
```

## Troubleshooting

| Issue | Solution |
|-------|----------|
| 404 Not Found | Check router rules, ensure labels match |
| 502 Bad Gateway | Backend service not running |
| Dashboard not loading | Check port 8080 is exposed |
| Service not discovered | Verify `traefik.enable=true` label |

## Next Chapter

In Chapter 02, we'll scale to 5 app instances and explore load balancing strategies.
