# Traefik Load Balancer Learning Project

A comprehensive, chapter-wise learning module for understanding Traefik as a modern reverse proxy and load balancer.

## Project Structure

```
traefik/
├── chapter-01-basics/          # Introduction to Traefik
├── chapter-02-load-balancing/  # Load balancing with 5 app nodes
├── chapter-03-networking/      # Internal vs External networks
├── chapter-04-comparison/      # Traefik vs Nginx comparison
├── app/                        # Sample application code
├── configs/                    # Traefik configuration files
└── README.md
```

## Chapters Overview

| Chapter | Topic | Key Concepts |
|---------|-------|--------------|
| 01 | Basics | Traefik architecture, entry points, routers, services |
| 02 | Load Balancing | 5-node cluster, health checks, sticky sessions |
| 03 | Networking | Internal vs external networks, Docker network modes |
| 04 | Comparison | Traefik vs Nginx - when to use what |

## Quick Start

```bash
# Chapter 1 - Basic setup
cd chapter-01-basics && docker-compose up -d

# Chapter 2 - Load balancing (5 nodes)
cd chapter-02-load-balancing && docker-compose up -d

# Chapter 3 - Networking concepts
cd chapter-03-networking && docker-compose up -d

# Chapter 4 - Comparison with Nginx
cd chapter-04-comparison && docker-compose up -d
```

## Prerequisites

- Docker & Docker Compose
- Basic understanding of containers and networking
- curl or a web browser for testing

## Key Traefik Concepts

### Architecture
```
                    ┌─────────────┐
    Internet ──────▶│ Entry Point │
                    └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │   Router    │ (Rules: Host, Path, Headers)
                    └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │ Middleware  │ (Auth, Rate Limit, Headers)
                    └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │   Service   │ (Load Balancer)
                    └──────┬──────┘
                           │
              ┌────────────┼────────────┐
              ▼            ▼            ▼
         ┌────────┐   ┌────────┐   ┌────────┐
         │ App 1  │   │ App 2  │   │ App N  │
         └────────┘   └────────┘   └────────┘
```

## Accessing the Dashboard

Traefik provides a built-in dashboard at: http://localhost:8080

## License

MIT - For educational purposes
