# Java Caching Learning Project

A comprehensive, chapter-based learning project to master caching concepts using Java/Spring Boot with Redis, PostgreSQL, RabbitMQ, and NGINX.

## Overview

This project provides hands-on experience with production-grade caching patterns through 8 progressive chapters, each building on previous concepts.

## Technology Stack

| Component | Technology | Purpose |
|-----------|------------|---------|
| Language | Java 21 (LTS) | Core application development |
| Framework | Spring Boot 3.x | Application framework |
| Cache | Redis Stack | Primary distributed cache with Bloom filters |
| Database | PostgreSQL 15 | Source of truth |
| Messaging | RabbitMQ 3.x | Cache invalidation events |
| Proxy | NGINX | HTTP caching layer |
| Local Cache | Caffeine | L1 in-memory cache |
| Metrics | Micrometer + Prometheus | Metrics collection |
| Dashboards | Grafana | Visualization |

## Quick Start

### Prerequisites

- Java 21+ installed
- Docker and Docker Compose
- Maven 3.9+

### Running the Project

1. **Clone and navigate to the project:**
   ```bash
   cd java-caching-learning
   ```

2. **Start infrastructure services:**
   ```bash
   docker-compose up -d postgres redis rabbitmq prometheus grafana
   ```

3. **Run a specific chapter:**
   ```bash
   # Build all modules
   ./mvnw clean install -DskipTests

   # Run Chapter 01
   cd chapter-01-fundamentals
   ../mvnw spring-boot:run
   ```

4. **Access services:**
   - Application: http://localhost:8080
   - Swagger UI: http://localhost:8080/swagger-ui.html
   - Redis Insight: http://localhost:8001
   - RabbitMQ UI: http://localhost:15672 (cache_user/cache_password)
   - Prometheus: http://localhost:9090
   - Grafana: http://localhost:3000 (admin/admin)

## Chapters

| Chapter | Topic | Key Concepts |
|---------|-------|--------------|
| [01](chapter-01-fundamentals/README.md) | Caching Fundamentals | Cache-aside, TTL, hit/miss ratios |
| [02](chapter-02-bloom-filters/README.md) | Bloom Filters | Cache penetration protection |
| [03](chapter-03-race-conditions/README.md) | Race Conditions | Thundering herd, mutex, coalescing |
| [04](chapter-04-distributed-patterns/README.md) | Distributed Patterns | Consistent hashing, sharding |
| [05](chapter-05-consistency/README.md) | Cache Consistency | Write-through, write-behind, invalidation |
| [06](chapter-06-nginx-caching/README.md) | NGINX Caching | HTTP caching, proxy_cache |
| [07](chapter-07-read-replicas/README.md) | Read Replicas | L1/L2 caching, near-cache |
| [08](chapter-08-advanced-patterns/README.md) | Advanced Patterns | Hot keys, TTL jitter, versioning |

## Project Structure

```
java-caching-learning/
├── README.md                    # This file
├── COURSE.md                    # Learning curriculum
├── docker-compose.yml           # All services
├── docs/
│   └── HLD.md                   # System-wide architecture
├── common/                      # Shared module
├── chapter-01-fundamentals/     # Cache basics
├── chapter-02-bloom-filters/    # Bloom filter protection
├── chapter-03-race-conditions/  # Concurrency patterns
├── chapter-04-distributed-patterns/
├── chapter-05-consistency/
├── chapter-06-nginx-caching/
├── chapter-07-read-replicas/
├── chapter-08-advanced-patterns/
├── nginx/                       # NGINX configuration
├── grafana/                     # Dashboards
└── prometheus/                  # Metrics config
```

## Learning Path

See [COURSE.md](COURSE.md) for the recommended learning path and curriculum.

## Verification

Each chapter includes verification tests:

| Chapter | Verification |
|---------|--------------|
| 01 | Hit ratio > 90% after warmup |
| 02 | Bloom filter blocks 99%+ invalid keys |
| 03 | Single DB query under concurrent load |
| 04 | Graceful node failure handling |
| 05 | Invalidation propagates < 100ms |
| 06 | NGINX cache hit < 1ms |
| 07 | L1 < 100μs, L2 < 5ms |
| 08 | Zero-downtime version switch |

## Common Commands

```bash
# Build all modules
./mvnw clean install

# Run tests
./mvnw test

# Run specific chapter
cd chapter-XX-name && ../mvnw spring-boot:run

# Start all Docker services
docker-compose up -d

# View logs
docker-compose logs -f app

# Stop services
docker-compose down

# Reset volumes (clean state)
docker-compose down -v
```

## Metrics Endpoints

- `/actuator/prometheus` - Prometheus metrics
- `/actuator/health` - Health check
- `/api/metrics/cache` - Custom cache metrics

## Contributing

This is a learning project. Feel free to:
- Add more examples
- Improve documentation
- Fix bugs
- Suggest new patterns

## License

MIT License - Free for educational use.
