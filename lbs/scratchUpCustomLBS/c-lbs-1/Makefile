# C Load Balancer Learning Project
# Main Makefile

CHAPTERS = chapter-01-fundamentals \
           chapter-02-simple-proxy \
           chapter-03-load-balancer \
           chapter-04-advanced-features \
           chapter-05-high-perf-io \
           chapter-06-connection-pooling \
           chapter-07-rate-limiting \
           chapter-08-metrics \
           chapter-09-zero-copy \
           chapter-10-hot-reload

.PHONY: all clean test $(CHAPTERS)

all: $(CHAPTERS)

$(CHAPTERS):
	@echo "Building $@..."
	@$(MAKE) -C $@ all

clean:
	@for dir in $(CHAPTERS); do \
		echo "Cleaning $$dir..."; \
		$(MAKE) -C $$dir clean; \
	done

# ============================================================================
# Test Targets
# ============================================================================

# Chapter 1: Echo server
test-ch1:
	@$(MAKE) -C chapter-01-fundamentals all
	@echo "Starting echo server on port 8080..."
	@./chapter-01-fundamentals/echo_server 8080

# Chapter 2: Reverse proxy
test-ch2:
	@$(MAKE) -C chapter-01-fundamentals all
	@$(MAKE) -C chapter-02-simple-proxy all
	@echo "Starting backend on 9000 and proxy on 8080..."
	@./chapter-01-fundamentals/echo_server 9000 &
	@sleep 1
	@./chapter-02-simple-proxy/reverse_proxy 8080 127.0.0.1 9000

# Chapter 3: Load balancer
test-ch3:
	@$(MAKE) -C chapter-01-fundamentals all
	@$(MAKE) -C chapter-03-load-balancer all
	@echo "Starting 3 backends and load balancer..."
	@./chapter-01-fundamentals/echo_server 9001 &
	@./chapter-01-fundamentals/echo_server 9002 &
	@./chapter-01-fundamentals/echo_server 9003 &
	@sleep 1
	@./chapter-03-load-balancer/load_balancer 8080 127.0.0.1:9001 127.0.0.1:9002 127.0.0.1:9003

# Chapter 4: Advanced LB
test-ch4:
	@$(MAKE) -C chapter-01-fundamentals all
	@$(MAKE) -C chapter-04-advanced-features all
	@echo "Starting 3 weighted backends and advanced LB..."
	@./chapter-01-fundamentals/echo_server 9001 &
	@./chapter-01-fundamentals/echo_server 9002 &
	@./chapter-01-fundamentals/echo_server 9003 &
	@sleep 1
	@./chapter-04-advanced-features/advanced_lb 8080 127.0.0.1:9001:3 127.0.0.1:9002:2 127.0.0.1:9003:1

# Chapter 5: High-performance event-driven LB
test-ch5:
	@$(MAKE) -C chapter-01-fundamentals all
	@$(MAKE) -C chapter-05-high-perf-io all
	@echo "Starting backends and high-perf event-driven LB..."
	@./chapter-01-fundamentals/echo_server 9001 &
	@./chapter-01-fundamentals/echo_server 9002 &
	@./chapter-01-fundamentals/echo_server 9003 &
	@sleep 1
	@./chapter-05-high-perf-io/high_perf_lb 8080 127.0.0.1:9001 127.0.0.1:9002 127.0.0.1:9003

# Chapter 6: Connection pooling LB
test-ch6:
	@$(MAKE) -C chapter-01-fundamentals all
	@$(MAKE) -C chapter-06-connection-pooling all
	@echo "Starting backends and pooled LB..."
	@./chapter-01-fundamentals/echo_server 9001 &
	@./chapter-01-fundamentals/echo_server 9002 &
	@./chapter-01-fundamentals/echo_server 9003 &
	@sleep 1
	@./chapter-06-connection-pooling/pooled_lb 8080 127.0.0.1:9001 127.0.0.1:9002 127.0.0.1:9003

# Chapter 7: Rate limiting (library test)
test-ch7:
	@$(MAKE) -C chapter-07-rate-limiting all
	@echo "Running rate limiter tests..."
	@./chapter-07-rate-limiting/rate_limiter_test

# Chapter 8: Metrics (library test)
test-ch8:
	@$(MAKE) -C chapter-08-metrics all
	@echo "Running metrics tests..."
	@./chapter-08-metrics/metrics_test

# Chapter 9: Zero-copy (library test)
test-ch9:
	@$(MAKE) -C chapter-09-zero-copy all
	@echo "Running zero-copy tests..."
	@./chapter-09-zero-copy/zero_copy_test

# Chapter 10: Hot reload demo
test-ch10:
	@$(MAKE) -C chapter-10-hot-reload all
	@echo "Running hot reload demo..."
	@./chapter-10-hot-reload/hot_reload_lb chapter-10-hot-reload/lb.json

# Kill all test processes
stop:
	@pkill -f "echo_server" || true
	@pkill -f "reverse_proxy" || true
	@pkill -f "load_balancer" || true
	@pkill -f "advanced_lb" || true
	@pkill -f "high_perf_lb" || true
	@pkill -f "pooled_lb" || true
	@pkill -f "hot_reload_lb" || true
	@echo "Stopped all processes"

help:
	@echo "C Load Balancer Learning Project"
	@echo "================================="
	@echo ""
	@echo "Build targets:"
	@echo "  make all        - Build all chapters"
	@echo "  make clean      - Clean all builds"
	@echo ""
	@echo "Test targets (Part II - Core):"
	@echo "  make test-ch1   - Chapter 1: Echo server"
	@echo "  make test-ch2   - Chapter 2: Reverse proxy"
	@echo "  make test-ch3   - Chapter 3: Load balancer"
	@echo "  make test-ch4   - Chapter 4: Advanced LB"
	@echo ""
	@echo "Test targets (Part III - Advanced):"
	@echo "  make test-ch5   - Chapter 5: High-perf event-driven LB"
	@echo "  make test-ch6   - Chapter 6: Connection pooling LB"
	@echo "  make test-ch7   - Chapter 7: Rate limiting tests"
	@echo "  make test-ch8   - Chapter 8: Metrics tests"
	@echo "  make test-ch9   - Chapter 9: Zero-copy tests"
	@echo "  make test-ch10  - Chapter 10: Hot reload demo"
	@echo ""
	@echo "  make stop       - Kill all test processes"
	@echo ""
	@echo "In another terminal, test with:"
	@echo "  echo 'Hello' | nc localhost 8080"
	@echo "  curl http://localhost:8080/"
