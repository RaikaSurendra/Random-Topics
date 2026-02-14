#!/bin/bash

# ============================================
# Chapter 05: Cache Consistency
# Required services: PostgreSQL, Redis, RabbitMQ
# ============================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

source "$PROJECT_ROOT/scripts/common.sh"

echo "============================================"
echo " Chapter 05: Cache Consistency"
echo " Event-driven invalidation, write patterns"
echo "============================================"
echo ""

check_docker
start_services postgres redis rabbitmq
load_sample_data
build_chapter "chapter-05-consistency"
show_urls

echo "Chapter 05 Endpoints:"
echo "  GET  /api/products/{sku}         - Get product"
echo "  PUT  /api/products/{sku}         - Update (triggers invalidation event)"
echo "  GET  /api/consistency/stats      - View invalidation stats"
echo "  POST /api/consistency/demo       - Demo write-through/behind"
echo ""
echo "RabbitMQ Management: http://localhost:15672"
echo "  Username: cache_user"
echo "  Password: cache_password"
echo ""

run_chapter "chapter-05-consistency"
