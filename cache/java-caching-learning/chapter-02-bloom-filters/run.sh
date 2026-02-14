#!/bin/bash

# ============================================
# Chapter 02: Bloom Filters
# Required services: PostgreSQL, Redis (with RedisBloom)
# ============================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

source "$PROJECT_ROOT/scripts/common.sh"

echo "============================================"
echo " Chapter 02: Bloom Filters"
echo " Cache penetration protection"
echo "============================================"
echo ""

# Check Docker
check_docker

# Start required services (redis-stack includes RedisBloom)
start_services postgres redis

# Load sample data
load_sample_data

# Build the chapter
build_chapter "chapter-02-bloom-filters"

# Show URLs
show_urls

# Additional chapter-specific endpoints
echo "Chapter 02 Endpoints:"
echo "  GET  /api/products/{sku}           - Get with Bloom protection"
echo "  GET  /api/products/unprotected/{sku} - Get without protection"
echo "  GET  /api/bloom/stats              - View Bloom filter stats"
echo "  POST /api/bloom/rebuild            - Rebuild Bloom filter"
echo "  POST /api/bloom/demo/attack        - Simulate penetration attack"
echo "  GET  /api/bloom/demo/false-positives - Measure false positive rate"
echo "  POST /api/bloom/demo/compare       - Compare protected vs unprotected"
echo ""

# Run the application
run_chapter "chapter-02-bloom-filters"
