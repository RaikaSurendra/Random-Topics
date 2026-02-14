#!/bin/bash

# ============================================
# Chapter 03: Cache Race Conditions
# Required services: PostgreSQL, Redis
# ============================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

source "$PROJECT_ROOT/scripts/common.sh"

echo "============================================"
echo " Chapter 03: Cache Race Conditions"
echo " Thundering herd, mutex, coalescing"
echo "============================================"
echo ""

# Check Docker
check_docker

# Start required services
start_services postgres redis

# Load sample data
load_sample_data

# Build the chapter
build_chapter "chapter-03-race-conditions"

# Show URLs
show_urls

# Additional chapter-specific endpoints
echo "Chapter 03 Endpoints:"
echo "  POST /api/demo/stampede            - Simulate cache stampede"
echo "       ?requests=1000&strategy=vulnerable|mutex|probabilistic|coalescing"
echo "  GET  /api/demo/mutex/{sku}         - Get with mutex lock"
echo "  GET  /api/demo/probabilistic/{sku} - Get with early refresh"
echo "  GET  /api/demo/coalescing/{sku}    - Get with request coalescing"
echo "  POST /api/demo/compare             - Compare all strategies"
echo ""

# Run the application
run_chapter "chapter-03-race-conditions"
