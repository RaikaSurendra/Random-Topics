#!/bin/bash

# ============================================
# Chapter 01: Caching Fundamentals
# Required services: PostgreSQL, Redis
# ============================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

source "$PROJECT_ROOT/scripts/common.sh"

echo "============================================"
echo " Chapter 01: Caching Fundamentals"
echo " Cache-aside pattern, TTL, hit/miss metrics"
echo "============================================"
echo ""

# Check Docker
check_docker

# Start required services
start_services postgres redis

# Load sample data
load_sample_data

# Build the chapter
build_chapter "chapter-01-fundamentals"

# Show URLs
show_urls

# Additional chapter-specific endpoints
echo "Chapter 01 Endpoints:"
echo "  GET  /api/products/{sku}          - Get product (cached)"
echo "  GET  /api/products/uncached/{sku} - Get product (uncached)"
echo "  GET  /api/metrics/cache           - View cache stats"
echo "  DEL  /api/products/cache/{sku}    - Evict cache entry"
echo "  POST /api/demo/load-test          - Run load test"
echo "  GET  /api/demo/compare/{sku}      - Compare cached vs uncached"
echo ""

# Run the application
run_chapter "chapter-01-fundamentals"
