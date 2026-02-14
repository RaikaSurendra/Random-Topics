#!/bin/bash

# ============================================
# Chapter 04: Distributed Caching Patterns
# Required services: PostgreSQL, Redis (primary + replicas)
# ============================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

source "$PROJECT_ROOT/scripts/common.sh"

echo "============================================"
echo " Chapter 04: Distributed Caching Patterns"
echo " Consistent hashing, sharding, failover"
echo "============================================"
echo ""

check_docker
start_services postgres redis redis-replica-1 redis-replica-2
load_sample_data
build_chapter "chapter-04-distributed-patterns"
show_urls

echo "Chapter 04 Endpoints:"
echo "  GET  /api/distributed/{key}      - Get with consistent hashing"
echo "  GET  /api/distributed/node/{key} - Show which node owns key"
echo "  GET  /api/distributed/ring       - Visualize hash ring"
echo "  POST /api/distributed/failover   - Simulate node failure"
echo ""

run_chapter "chapter-04-distributed-patterns"
