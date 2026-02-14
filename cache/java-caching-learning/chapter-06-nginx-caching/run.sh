#!/bin/bash

# ============================================
# Chapter 06: NGINX Caching
# Required services: PostgreSQL, Redis, NGINX
# ============================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

source "$PROJECT_ROOT/scripts/common.sh"

echo "============================================"
echo " Chapter 06: NGINX Caching"
echo " HTTP caching, proxy_cache, microcaching"
echo "============================================"
echo ""

check_docker

# Build NGINX first
log_info "Building NGINX image..."
cd "$PROJECT_ROOT"
docker-compose build nginx

start_services postgres redis
load_sample_data
build_chapter "chapter-06-nginx-caching"

# Start NGINX after app is built
log_info "Note: Start NGINX after application is running"
echo ""

show_urls

echo "Chapter 06 Endpoints (via NGINX on port 80):"
echo "  GET  /api/products/{sku}           - Cached (60s)"
echo "  GET  /api/products/no-cache/{sku}  - Not cached"
echo "  GET  /api/products/microcache/{sku} - Microcached (1s)"
echo ""
echo "Check X-Cache-Status header in responses:"
echo "  HIT     - Served from NGINX cache"
echo "  MISS    - Fetched from application"
echo "  EXPIRED - Cache expired, refreshing"
echo "  STALE   - Serving stale while updating"
echo ""
echo "After starting app, start NGINX with:"
echo "  docker-compose up -d nginx"
echo ""

run_chapter "chapter-06-nginx-caching"
