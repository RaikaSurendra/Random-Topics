#!/bin/bash

# UUID v4 vs v7 PostgreSQL Benchmark Runner
# ==========================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  UUID v4 vs v7 PostgreSQL Benchmark${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}Error: Docker is not running. Please start Docker first.${NC}"
    exit 1
fi

# Function to wait for PostgreSQL
wait_for_postgres() {
    echo -e "${YELLOW}Waiting for PostgreSQL to be ready...${NC}"
    for i in {1..30}; do
        if docker exec uuid_benchmark_db pg_isready -U benchmark -d uuid_test > /dev/null 2>&1; then
            echo -e "${GREEN}PostgreSQL is ready!${NC}"
            return 0
        fi
        echo -n "."
        sleep 1
    done
    echo -e "${RED}Timeout waiting for PostgreSQL${NC}"
    exit 1
}

# Parse command line arguments
case "${1:-}" in
    start)
        echo -e "${YELLOW}Starting PostgreSQL container...${NC}"
        docker-compose up -d
        wait_for_postgres
        echo ""
        echo -e "${GREEN}PostgreSQL is running!${NC}"
        echo "Connect with: psql -h localhost -U benchmark -d uuid_test"
        echo "Password: benchmark123"
        ;;

    stop)
        echo -e "${YELLOW}Stopping PostgreSQL container...${NC}"
        docker-compose down
        echo -e "${GREEN}Stopped.${NC}"
        ;;

    clean)
        echo -e "${YELLOW}Removing PostgreSQL container and data...${NC}"
        docker-compose down -v
        echo -e "${GREEN}Cleaned.${NC}"
        ;;

    benchmark)
        if ! docker ps | grep -q uuid_benchmark_db; then
            echo -e "${YELLOW}Container not running. Starting...${NC}"
            docker-compose up -d
            wait_for_postgres
        fi

        echo ""
        echo -e "${GREEN}Running benchmark...${NC}"
        echo ""

        docker exec -i uuid_benchmark_db psql -U benchmark -d uuid_test < benchmark.sql
        ;;

    interactive)
        if ! docker ps | grep -q uuid_benchmark_db; then
            echo -e "${YELLOW}Container not running. Starting...${NC}"
            docker-compose up -d
            wait_for_postgres
        fi

        echo -e "${GREEN}Opening interactive psql session...${NC}"
        docker exec -it uuid_benchmark_db psql -U benchmark -d uuid_test
        ;;

    logs)
        docker-compose logs -f
        ;;

    *)
        echo "Usage: $0 {start|stop|clean|benchmark|interactive|logs}"
        echo ""
        echo "Commands:"
        echo "  start       - Start PostgreSQL container"
        echo "  stop        - Stop PostgreSQL container"
        echo "  clean       - Remove container and all data"
        echo "  benchmark   - Run the UUID benchmark"
        echo "  interactive - Open psql shell"
        echo "  logs        - Follow container logs"
        echo ""
        echo "Quick start:"
        echo "  $0 start      # Start the database"
        echo "  $0 benchmark  # Run the benchmark"
        echo "  $0 clean      # Clean up when done"
        ;;
esac
