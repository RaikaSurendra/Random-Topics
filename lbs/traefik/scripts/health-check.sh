#!/bin/bash
# =============================================================================
# Health Check Script
# =============================================================================
# Checks the health of all services in the stack
# =============================================================================

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "=========================================="
echo "  Service Health Check"
echo "=========================================="
echo ""

# Function to check HTTP endpoint
check_http() {
    local name=$1
    local url=$2
    local timeout=${3:-5}

    response=$(curl -s -o /dev/null -w "%{http_code}" --max-time $timeout "$url" 2>/dev/null)

    if [ "$response" == "200" ]; then
        echo -e "${GREEN}✓${NC} $name: ${GREEN}Healthy${NC} (HTTP $response)"
        return 0
    else
        echo -e "${RED}✗${NC} $name: ${RED}Unhealthy${NC} (HTTP $response)"
        return 1
    fi
}

# Function to check container
check_container() {
    local name=$1

    if docker ps --format '{{.Names}}' | grep -q "^${name}$"; then
        status=$(docker inspect --format '{{.State.Status}}' "$name" 2>/dev/null)
        if [ "$status" == "running" ]; then
            echo -e "${GREEN}✓${NC} Container $name: ${GREEN}Running${NC}"
            return 0
        fi
    fi
    echo -e "${RED}✗${NC} Container $name: ${RED}Not running${NC}"
    return 1
}

echo "Checking Traefik..."
check_http "Traefik Dashboard" "http://localhost:8080/api/overview" || true
check_container "traefik-lb" || check_container "traefik-basics" || check_container "traefik-networks" || true

echo ""
echo "Checking Application Nodes..."
for i in 1 2 3 4 5; do
    container_name="app-node-$i"
    if docker ps --format '{{.Names}}' | grep -q "^${container_name}$"; then
        check_container "$container_name" || true
    fi
done

echo ""
echo "Checking Load Balancer Endpoint..."
check_http "Load Balancer" "http://localhost" || check_http "Load Balancer (8000)" "http://localhost:8000" || true

echo ""
echo "=========================================="
echo "  Network Status"
echo "=========================================="
echo ""

for network in traefik-lb-net traefik-basics-net traefik-internal traefik-external backend-network comparison-net; do
    if docker network ls --format '{{.Name}}' | grep -q "^${network}$"; then
        containers=$(docker network inspect "$network" --format '{{range .Containers}}{{.Name}} {{end}}' 2>/dev/null)
        echo -e "${GREEN}✓${NC} Network $network: $containers"
    fi
done

echo ""
