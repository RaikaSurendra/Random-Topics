#!/bin/bash

# Load sample data into PostgreSQL

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

source "$PROJECT_ROOT/scripts/common.sh"

log_info "Loading sample data into PostgreSQL..."

# Check if PostgreSQL is running
if ! docker-compose -f "$PROJECT_ROOT/docker-compose.yml" ps postgres | grep -q "running"; then
    log_error "PostgreSQL is not running. Start it first with: docker-compose up -d postgres"
    exit 1
fi

# Load the init script (if not already loaded)
log_info "Running init script..."
docker-compose -f "$PROJECT_ROOT/docker-compose.yml" exec -T postgres \
    psql -U cache_user -d caching_db -f /docker-entrypoint-initdb.d/01-init.sql 2>/dev/null || true

# Load additional sample data
log_info "Loading additional sample data..."
docker-compose -f "$PROJECT_ROOT/docker-compose.yml" exec -T postgres \
    psql -U cache_user -d caching_db < "$SCRIPT_DIR/sample-data.sql"

# Show counts
log_info "Data summary:"
docker-compose -f "$PROJECT_ROOT/docker-compose.yml" exec -T postgres \
    psql -U cache_user -d caching_db -c "SELECT 'Products: ' || COUNT(*) FROM products UNION ALL SELECT 'Users: ' || COUNT(*) FROM users UNION ALL SELECT 'Orders: ' || COUNT(*) FROM orders;"

log_success "Sample data loaded successfully!"
