# Quick Start Guide

Get up and running with Traefik load balancing in 5 minutes.

## Prerequisites

```bash
# Verify Docker is running
docker --version
docker-compose --version

# Install jq for JSON parsing (optional but recommended)
brew install jq

# Install hey for load testing (optional)
brew install hey
```

## Chapter 2: Load Balancing (Recommended Start)

This chapter shows Traefik balancing load across 5 application nodes.

### 1. Start the Stack

```bash
cd traefik/chapter-02-load-balancing
docker-compose up -d --build
```

### 2. Verify Everything is Running

```bash
docker-compose ps
```

Expected output:
```
NAME          STATUS    PORTS
traefik-lb    running   0.0.0.0:80->80/tcp, 0.0.0.0:8080->8080/tcp
app-node-1    running
app-node-2    running
app-node-3    running
app-node-4    running
app-node-5    running
```

### 3. Open Traefik Dashboard

Open in browser: http://localhost:8080

You'll see:
- **HTTP Routers**: `app-lb@docker`
- **HTTP Services**: `app-lb@docker` with 5 servers
- **Health status** of each backend

### 4. Test Load Balancing

```bash
# Make 10 requests and see them distributed
for i in {1..10}; do
  curl -s http://localhost | jq '.instance.instance_id'
done
```

Output shows round-robin distribution:
```
"node-1"
"node-2"
"node-3"
"node-4"
"node-5"
"node-1"
...
```

### 5. Test with Visualization Script

```bash
../scripts/test-load-balancing.sh http://localhost 20
```

### 6. Test Sticky Sessions

```bash
# First request sets cookie
curl -c cookies.txt -s http://localhost | jq '.instance.instance_id'

# All subsequent requests go to same node
for i in {1..5}; do
  curl -b cookies.txt -s http://localhost | jq '.instance.instance_id'
done
```

### 7. Test Failover

```bash
# Stop one node
docker stop app-node-3

# Requests automatically route to healthy nodes
for i in {1..6}; do curl -s http://localhost | jq '.instance.instance_id'; done

# Restart node
docker start app-node-3
```

### 8. View Logs

```bash
# Traefik access logs (JSON format)
docker-compose logs -f traefik

# Application logs
docker-compose logs -f app-node-1 app-node-2
```

### 9. Clean Up

```bash
docker-compose down
```

## Quick Commands Reference

| Command | Description |
|---------|-------------|
| `docker-compose up -d` | Start all services |
| `docker-compose down` | Stop and remove containers |
| `docker-compose logs -f` | Follow all logs |
| `docker-compose ps` | List running services |
| `docker stop <name>` | Stop specific container |
| `docker start <name>` | Start specific container |
| `curl http://localhost` | Test endpoint |
| `curl http://localhost:8080/api/overview` | Traefik API |

## Endpoints Summary

| Chapter | App URL | Dashboard |
|---------|---------|-----------|
| 01-basics | http://localhost | http://localhost:8080 |
| 02-load-balancing | http://localhost | http://localhost:8080 |
| 03-networking | http://localhost | http://localhost:8080 |
| 04-comparison | http://localhost:8000 (Traefik), :8001 (Nginx) | http://localhost:8081 |

## Troubleshooting

### Port already in use

```bash
# Find what's using the port
lsof -i :80

# Stop conflicting service or use different port
```

### Services not starting

```bash
# Check logs for errors
docker-compose logs

# Rebuild images
docker-compose up -d --build --force-recreate
```

### Traefik not finding services

```bash
# Verify labels are correct
docker inspect app-node-1 | grep -A 50 Labels

# Check Traefik logs
docker-compose logs traefik | grep -i error
```

## Next Steps

1. **Chapter 01**: Understand core concepts
2. **Chapter 02**: Master load balancing (you are here)
3. **Chapter 03**: Learn network isolation
4. **Chapter 04**: Compare with Nginx
