#!/bin/bash

# ============================================
# Chapter 07: Read Replicas & Two-Level Caching
# Required services: PostgreSQL, Redis (primary + replicas)
# ============================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

source "$PROJECT_ROOT/scripts/common.sh"

echo "============================================"
echo " Chapter 07: Read Replicas & Two-Level Cache"
echo " L1 (Caffeine) + L2 (Redis), replica reads"
echo "============================================"
echo ""

check_docker
start_services postgres redis redis-replica-1 redis-replica-2
load_sample_data
build_chapter "chapter-07-read-replicas"
show_urls

echo "Chapter 07 Endpoints:"
echo "  GET  /api/products/{sku}     - Get with two-level cache"
echo "  GET  /api/replicas/stats     - View L1/L2 cache stats"
echo "  GET  /api/replicas/latency   - Compare L1 vs L2 latency"
echo "  POST /api/replicas/evict     - Evict from both levels"
echo ""
echo "Performance Targets:"
echo "  L1 (Caffeine):  < 100μs"
echo "  L2 (Redis):     < 5ms"
echo "  Database:       < 50ms"
echo ""

run_chapter "chapter-07-read-replicas"
