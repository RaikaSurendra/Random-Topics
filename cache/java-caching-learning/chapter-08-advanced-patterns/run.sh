#!/bin/bash

# ============================================
# Chapter 08: Advanced Caching Patterns
# Required services: PostgreSQL, Redis
# ============================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

source "$PROJECT_ROOT/scripts/common.sh"

echo "============================================"
echo " Chapter 08: Advanced Caching Patterns"
echo " Warming, hot keys, TTL jitter, versioning"
echo "============================================"
echo ""

check_docker
start_services postgres redis
load_sample_data
build_chapter "chapter-08-advanced-patterns"
show_urls

echo "Chapter 08 Endpoints:"
echo "  GET  /api/advanced/products/{sku}  - Get product"
echo "  GET  /api/advanced/warmer/stats    - View warming stats"
echo "  POST /api/advanced/warmer/warm     - Trigger cache warming"
echo "  GET  /api/advanced/hotkeys         - View detected hot keys"
echo "  GET  /api/advanced/hotkeys/top     - Get top accessed keys"
echo "  POST /api/advanced/version/switch  - Switch cache version"
echo "  GET  /api/advanced/version         - Get current version"
echo ""
echo "On startup, watch for cache warming logs:"
echo "  'Starting cache warming...'"
echo "  'Cache warming complete: N products'"
echo ""

run_chapter "chapter-08-advanced-patterns"
