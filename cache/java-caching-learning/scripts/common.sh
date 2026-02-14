#!/bin/bash

# Common functions for chapter scripts

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Docker is running
check_docker() {
    if ! docker info > /dev/null 2>&1; then
        log_error "Docker is not running. Please start Docker first."
        exit 1
    fi
}

# Wait for a service to be healthy
wait_for_service() {
    local service=$1
    local max_attempts=${2:-30}
    local attempt=1

    log_info "Waiting for $service to be ready..."

    while [ $attempt -le $max_attempts ]; do
        if docker-compose -f "$PROJECT_ROOT/docker-compose.yml" ps "$service" 2>/dev/null | grep -q "healthy\|running"; then
            log_success "$service is ready!"
            return 0
        fi
        echo -n "."
        sleep 2
        attempt=$((attempt + 1))
    done

    log_error "$service failed to start within timeout"
    return 1
}

# Start specific Docker services
start_services() {
    local services="$@"
    log_info "Starting services: $services"

    cd "$PROJECT_ROOT"
    docker-compose up -d $services

    # Wait for each service
    for service in $services; do
        wait_for_service "$service" || return 1
    done
}

# Stop all Docker services
stop_services() {
    log_info "Stopping all services..."
    cd "$PROJECT_ROOT"
    docker-compose down
}

# Load sample data
load_sample_data() {
    log_info "Loading sample data..."

    # Wait for PostgreSQL
    wait_for_service "postgres" || return 1

    # Check if data already exists
    local count=$(docker-compose -f "$PROJECT_ROOT/docker-compose.yml" exec -T postgres \
        psql -U cache_user -d caching_db -t -c "SELECT COUNT(*) FROM products;" 2>/dev/null | tr -d ' ')

    if [ "$count" -gt "0" ] 2>/dev/null; then
        log_info "Sample data already loaded ($count products)"
        return 0
    fi

    # Load additional data
    docker-compose -f "$PROJECT_ROOT/docker-compose.yml" exec -T postgres \
        psql -U cache_user -d caching_db < "$PROJECT_ROOT/scripts/data/sample-data.sql"

    log_success "Sample data loaded successfully"
}

# Build a specific chapter
build_chapter() {
    local chapter=$1
    log_info "Building $chapter..."

    cd "$PROJECT_ROOT"
    ./mvnw -pl common,$chapter -am clean package -DskipTests -q

    if [ $? -eq 0 ]; then
        log_success "$chapter built successfully"
    else
        log_error "Failed to build $chapter"
        return 1
    fi
}

# Run a specific chapter
run_chapter() {
    local chapter=$1
    log_info "Starting $chapter application..."

    cd "$PROJECT_ROOT/$chapter"
    ../mvnw spring-boot:run
}

# Show chapter URLs
show_urls() {
    echo ""
    log_info "Available URLs:"
    echo "  Application:    http://localhost:8080"
    echo "  Swagger UI:     http://localhost:8080/swagger-ui.html"
    echo "  Actuator:       http://localhost:8080/actuator"
    echo "  Prometheus:     http://localhost:8080/actuator/prometheus"
    echo ""
    echo "  Redis Insight:  http://localhost:8001"
    echo "  RabbitMQ UI:    http://localhost:15672 (cache_user/cache_password)"
    echo "  Grafana:        http://localhost:3000 (admin/admin)"
    echo "  Prometheus UI:  http://localhost:9090"
    echo ""
}
