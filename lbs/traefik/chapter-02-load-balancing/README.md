# Chapter 02: Load Balancing with 5 Nodes

## Learning Objectives

1. Configure load balancing across multiple instances
2. Understand load balancing algorithms
3. Implement health checks
4. Configure sticky sessions
5. Monitor load distribution via logging

## Architecture

```
                         ┌─────────────┐
                         │   Traefik   │
        Port 80 ────────▶│ Entry Point │
                         └──────┬──────┘
                                │
                         ┌──────▼──────┐
                         │   Router    │
                         │ Host: app.* │
                         └──────┬──────┘
                                │
                    ┌───────────┼───────────┐
                    │    Load Balancer      │
                    │   (Round Robin)       │
                    └───────────┬───────────┘
                                │
        ┌───────┬───────┬───────┼───────┬───────┐
        ▼       ▼       ▼       ▼       ▼       │
    ┌──────┐┌──────┐┌──────┐┌──────┐┌──────┐    │
    │App 1 ││App 2 ││App 3 ││App 4 ││App 5 │    │
    │ Blue ││Green ││Yellow││Orange││Purple│    │
    └──────┘└──────┘└──────┘└──────┘└──────┘    │
        │       │       │       │       │       │
        └───────┴───────┴───────┴───────┴───────┘
                    Health Checks
```

## Running This Chapter

```bash
# Start all 5 app instances + Traefik
docker-compose up -d

# Verify all containers are running
docker-compose ps

# Watch load balancing in action
for i in {1..10}; do curl -s http://localhost | jq '.instance.hostname'; done

# View detailed logs
docker-compose logs -f

# Stop everything
docker-compose down
```

## Load Balancing Algorithms

### 1. Round Robin (Default)

Requests are distributed sequentially across all healthy servers.

```
Request 1 → App1
Request 2 → App2
Request 3 → App3
Request 4 → App4
Request 5 → App5
Request 6 → App1 (cycle repeats)
```

### 2. Weighted Round Robin

Assign weights to servers based on capacity.

```yaml
labels:
  - "traefik.http.services.app.loadbalancer.server.weight=3"
```

### 3. Sticky Sessions

Keep a user on the same backend server.

```yaml
labels:
  - "traefik.http.services.app.loadbalancer.sticky.cookie=true"
  - "traefik.http.services.app.loadbalancer.sticky.cookie.name=server_id"
```

## Health Checks

Traefik automatically removes unhealthy backends.

```yaml
labels:
  - "traefik.http.services.app.loadbalancer.healthcheck.path=/health"
  - "traefik.http.services.app.loadbalancer.healthcheck.interval=10s"
  - "traefik.http.services.app.loadbalancer.healthcheck.timeout=3s"
```

### Testing Health Checks

```bash
# Stop one instance
docker stop app-node-3

# Verify Traefik routes around it
for i in {1..10}; do curl -s http://localhost | jq '.instance.instance_id'; done

# Restart the instance
docker start app-node-3
```

## Exercises

### Exercise 1: Observe Round Robin

```bash
# Make 10 requests and see distribution
for i in {1..10}; do
  echo "Request $i:"
  curl -s http://localhost | jq -r '.instance | "  Hostname: \(.hostname), ID: \(.instance_id)"'
  sleep 0.5
done
```

Expected output shows cycling through all 5 nodes.

### Exercise 2: Test Sticky Sessions

```bash
# First request - will set cookie
curl -c cookies.txt -s http://localhost | jq '.instance.instance_id'

# Subsequent requests - same server
for i in {1..5}; do
  curl -b cookies.txt -s http://localhost | jq '.instance.instance_id'
done
```

All requests should go to the same instance.

### Exercise 3: Load Test

```bash
# Install hey (HTTP load generator) if needed
# brew install hey

# Generate load
hey -n 100 -c 10 http://localhost/

# Watch distribution in logs
docker-compose logs -f app-node-1 app-node-2 app-node-3 app-node-4 app-node-5
```

### Exercise 4: Simulate Failure

```bash
# Terminal 1: Watch logs
docker-compose logs -f traefik

# Terminal 2: Stop a node
docker stop app-node-2

# Terminal 2: Make requests (should skip node-2)
for i in {1..6}; do curl -s http://localhost | jq '.instance.instance_id'; done

# Restart
docker start app-node-2
```

## Log Analysis

### Understanding Access Logs

```
192.168.65.1 - - [Date] "GET / HTTP/1.1" 200 523 "-" "curl" 5 "app-lb@docker" "http://172.19.0.3:5000" 12ms
                                                              │        │                │               │
                                                              │        │                │               └─ Response time
                                                              │        │                └─ Backend selected
                                                              │        └─ Service name
                                                              └─ Request count
```

### Tracking Distribution

```bash
# Count requests per backend (from Traefik logs)
docker-compose logs traefik | grep "GET /" | awk -F'"' '{print $8}' | sort | uniq -c
```

## Configuration Deep Dive

### Scaling Services

```bash
# Scale to 10 instances
docker-compose up -d --scale app-node-1=2 app-node-2=2

# Note: Our setup uses explicit services for better learning visibility
```

### Custom Load Balancer Settings

```yaml
labels:
  # Pass host header
  - "traefik.http.services.app.loadbalancer.passhostheader=true"

  # Response forwarding timeout
  - "traefik.http.services.app.loadbalancer.responseforwarding.flushinterval=100ms"
```

## Monitoring via Dashboard

1. Open http://localhost:8080
2. Navigate to HTTP → Services
3. Click on "app-lb@docker"
4. See all 5 servers and their status
5. Watch the "Requests" counter

## Key Takeaways

| Concept | What You Learned |
|---------|------------------|
| Round Robin | Default, even distribution |
| Health Checks | Automatic failover |
| Sticky Sessions | Session persistence |
| Logging | Debug load distribution |
| Dashboard | Visual monitoring |

## Next Chapter

Chapter 03 explores Docker networking - internal vs external networks and how Traefik bridges them.
