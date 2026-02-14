#!/bin/bash
# =============================================================================
# Load Balancing Test Script
# =============================================================================
# This script helps visualize load distribution across backend nodes
# =============================================================================

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Default values
URL="${1:-http://localhost}"
REQUESTS="${2:-20}"

echo -e "${CYAN}========================================${NC}"
echo -e "${CYAN}  Traefik Load Balancing Test${NC}"
echo -e "${CYAN}========================================${NC}"
echo ""
echo -e "URL: ${YELLOW}$URL${NC}"
echo -e "Requests: ${YELLOW}$REQUESTS${NC}"
echo ""

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo -e "${RED}Error: jq is required but not installed.${NC}"
    echo "Install with: brew install jq"
    exit 1
fi

# Arrays to track distribution
declare -A node_counts

echo -e "${GREEN}Starting requests...${NC}"
echo ""

for i in $(seq 1 $REQUESTS); do
    response=$(curl -s "$URL" 2>/dev/null)

    if [ $? -eq 0 ] && [ -n "$response" ]; then
        instance_id=$(echo "$response" | jq -r '.instance.instance_id // "unknown"')
        hostname=$(echo "$response" | jq -r '.instance.hostname // "unknown"')

        # Increment counter
        ((node_counts[$instance_id]++)) || node_counts[$instance_id]=1

        # Color based on node
        case $instance_id in
            *node-1*|*1*) color=$BLUE ;;
            *node-2*|*2*) color=$GREEN ;;
            *node-3*|*3*) color=$YELLOW ;;
            *node-4*|*4*) color=$PURPLE ;;
            *node-5*|*5*) color=$CYAN ;;
            *) color=$NC ;;
        esac

        printf "Request %2d: ${color}%-15s${NC} (Host: %s)\n" "$i" "$instance_id" "$hostname"
    else
        echo -e "Request $i: ${RED}FAILED${NC}"
    fi

    sleep 0.1
done

echo ""
echo -e "${CYAN}========================================${NC}"
echo -e "${CYAN}  Distribution Summary${NC}"
echo -e "${CYAN}========================================${NC}"
echo ""

total=0
for node in "${!node_counts[@]}"; do
    count=${node_counts[$node]}
    ((total += count))
done

for node in $(echo "${!node_counts[@]}" | tr ' ' '\n' | sort); do
    count=${node_counts[$node]}
    percentage=$((count * 100 / total))
    bar=$(printf '█%.0s' $(seq 1 $((percentage / 2))))

    case $node in
        *node-1*|*1*) color=$BLUE ;;
        *node-2*|*2*) color=$GREEN ;;
        *node-3*|*3*) color=$YELLOW ;;
        *node-4*|*4*) color=$PURPLE ;;
        *node-5*|*5*) color=$CYAN ;;
        *) color=$NC ;;
    esac

    printf "${color}%-15s${NC}: %3d requests (%3d%%) %s\n" "$node" "$count" "$percentage" "$bar"
done

echo ""
echo -e "Total requests: ${GREEN}$total${NC}"
echo ""
