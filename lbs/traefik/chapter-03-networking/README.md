# Chapter 03: Internal vs External Networks

## Learning Objectives

1. Understand Docker network types and their purposes
2. Differentiate between internal and external networks
3. Learn how Traefik bridges network boundaries
4. Implement network isolation for security
5. Configure cross-network communication

## Network Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              HOST MACHINE                                    │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    EXTERNAL NETWORK (traefik-external)               │   │
│  │                         Accessible from host                         │   │
│  │  ┌─────────────────────────────────────────────────────────────┐    │   │
│  │  │                       TRAEFIK                                │    │   │
│  │  │                  Ports: 80, 443, 8080                        │    │   │
│  │  │              (Bridge between networks)                       │    │   │
│  │  └─────────────────────────┬───────────────────────────────────┘    │   │
│  └────────────────────────────┼────────────────────────────────────────┘   │
│                               │                                             │
│  ┌────────────────────────────┼────────────────────────────────────────┐   │
│  │            INTERNAL NETWORK (traefik-internal)                       │   │
│  │                  NOT accessible from host                            │   │
│  │                                                                      │   │
│  │    ┌─────────┐   ┌─────────┐   ┌─────────┐   ┌─────────────────┐   │   │
│  │    │  App 1  │   │  App 2  │   │  App 3  │   │    Database     │   │   │
│  │    │ (web)   │   │ (api)   │   │ (admin) │   │  (PostgreSQL)   │   │   │
│  │    └─────────┘   └─────────┘   └─────────┘   └─────────────────┘   │   │
│  │                                                                      │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │              BACKEND NETWORK (backend-only)                          │   │
│  │           Isolated - No Traefik access                               │   │
│  │                                                                      │   │
│  │    ┌─────────────┐      ┌─────────────┐      ┌─────────────┐        │   │
│  │    │    Redis    │      │   Worker    │      │   Worker    │        │   │
│  │    │   (cache)   │      │     #1      │      │     #2      │        │   │
│  │    └─────────────┘      └─────────────┘      └─────────────┘        │   │
│  │                                                                      │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Network Types Explained

### 1. External Network
- **Purpose**: Public-facing services
- **Access**: Reachable from host machine and internet
- **Use Case**: Load balancers, web servers
- **Security**: Needs firewall, rate limiting

```yaml
networks:
  traefik-external:
    driver: bridge
    # No internal flag = external (default)
```

### 2. Internal Network
- **Purpose**: Service-to-service communication
- **Access**: Only containers on same network
- **Use Case**: App servers, APIs, microservices
- **Security**: No direct internet access

```yaml
networks:
  traefik-internal:
    driver: bridge
    internal: true  # KEY: Makes network internal-only
```

### 3. Backend-Only Network
- **Purpose**: Sensitive backend services
- **Access**: No Traefik, no external access
- **Use Case**: Databases, caches, workers
- **Security**: Maximum isolation

## Running This Chapter

```bash
# Start the multi-network setup
docker-compose up -d

# Verify network isolation
docker network ls | grep traefik

# Test external access
curl http://localhost        # Works: web app
curl http://localhost/api    # Works: API (via Traefik)

# Test internal isolation
docker exec traefik-networks ping -c 1 app-web     # Works
docker exec traefik-networks ping -c 1 db          # Works
docker exec traefik-networks ping -c 1 redis       # Fails (different network)

# Stop
docker-compose down
```

## Exercises

### Exercise 1: Network Inspection

```bash
# List all networks
docker network ls

# Inspect internal network
docker network inspect traefik-internal

# See which containers are on which networks
docker network inspect traefik-external --format '{{range .Containers}}{{.Name}} {{end}}'
docker network inspect traefik-internal --format '{{range .Containers}}{{.Name}} {{end}}'
```

### Exercise 2: Test Network Isolation

```bash
# From Traefik container, try to reach different services
docker exec traefik-networks sh -c "wget -qO- http://app-web:5000/health"     # Success
docker exec traefik-networks sh -c "wget -qO- http://app-api:5000/health"     # Success
docker exec traefik-networks sh -c "wget -qO- http://redis:6379" 2>&1         # Fails

# From app container, try to reach backend
docker exec app-web sh -c "wget -qO- http://db:5432" 2>&1                     # Success (same network)
```

### Exercise 3: Cross-Network Communication

```bash
# Web app can talk to API (both on internal network)
docker exec app-web sh -c "wget -qO- http://app-api:5000/health"

# API can talk to database (both on internal network)
docker exec app-api sh -c "nc -zv db 5432"
```

## Security Patterns

### Pattern 1: DMZ Architecture

```
Internet → External Net → Traefik → Internal Net → App → Backend Net → DB
                  │                        │                    │
            Public facing           App servers          Databases only
```

### Pattern 2: Service Mesh

```yaml
# Each service only accesses what it needs
app-web:
  networks:
    - traefik-internal  # Traefik access
    - cache-network     # Redis access

app-api:
  networks:
    - traefik-internal  # Traefik access
    - db-network        # Database access
```

### Pattern 3: Sidecar Proxy

```yaml
# Traefik as sidecar for specific service
app:
  networks:
    - app-private
traefik-sidecar:
  networks:
    - app-private       # Same network as app
    - external          # Exposed to outside
```

## Traefik's Role in Networking

### Why Traefik Excels Here

1. **Multi-Network Aware**: Can connect to multiple Docker networks
2. **Dynamic Discovery**: Finds services across networks automatically
3. **Network Bridging**: Routes between isolated networks safely
4. **Label-Based Config**: No need to expose ports

```yaml
# Traefik connects to multiple networks
traefik:
  networks:
    - external    # For incoming traffic
    - internal    # To reach backend services
```

### Security Benefits

| Feature | Benefit |
|---------|---------|
| Internal networks | Services not directly exposed |
| Label routing | No port mapping needed |
| Health checks | Automatic failover |
| Access logs | Full request visibility |

## Common Networking Issues

### Issue 1: Service Not Reachable

```bash
# Check Traefik is on same network
docker network inspect traefik-internal | grep traefik

# Verify service is on network
docker inspect app-web --format '{{json .NetworkSettings.Networks}}' | jq
```

### Issue 2: DNS Resolution Failing

```bash
# Test DNS from Traefik
docker exec traefik-networks nslookup app-web

# Ensure using correct network in labels
labels:
  - "traefik.docker.network=traefik-internal"
```

### Issue 3: Cannot Connect to External Service

```bash
# Internal networks cannot reach internet
# Solution: Use external network for services needing internet access
```

## Network Configuration Reference

```yaml
networks:
  # External network - accessible from outside
  external:
    driver: bridge

  # Internal network - container-to-container only
  internal:
    driver: bridge
    internal: true

  # Custom subnet
  custom:
    driver: bridge
    ipam:
      config:
        - subnet: 172.28.0.0/16

  # Existing network (created outside compose)
  existing:
    external: true
    name: my-existing-network
```

## Key Takeaways

| Concept | Internal Network | External Network |
|---------|------------------|------------------|
| Host Access | No | Yes |
| Internet Access | No | Yes |
| Container-to-Container | Yes | Yes |
| Security | High | Lower |
| Use Case | Backend services | Public services |

## Next Chapter

Chapter 04 compares Traefik with Nginx - understanding when to use each solution.
