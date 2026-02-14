# Chapter 05: High-Performance I/O
# Cross-platform event loop with epoll/kqueue/select

CC = gcc
CFLAGS = -Wall -Wextra -g -O2
LDFLAGS =

# Detect platform
UNAME_S := $(shell uname -s)

ifeq ($(UNAME_S),Linux)
    EVENT_BACKEND = event_loop_epoll.c
    BACKEND_FLAG = -DEVENT_LOOP_EPOLL
else ifeq ($(UNAME_S),Darwin)
    EVENT_BACKEND = event_loop_kqueue.c
    BACKEND_FLAG = -DEVENT_LOOP_KQUEUE
else
    EVENT_BACKEND = event_loop_select.c
    BACKEND_FLAG = -DEVENT_LOOP_SELECT
endif

TARGETS = high_perf_lb

all: $(TARGETS)

high_perf_lb: high_perf_lb.c $(EVENT_BACKEND) event_loop.h
	$(CC) $(CFLAGS) $(BACKEND_FLAG) -o $@ high_perf_lb.c $(EVENT_BACKEND) $(LDFLAGS)

# Build with specific backend (for testing)
high_perf_lb_select: high_perf_lb.c event_loop_select.c event_loop.h
	$(CC) $(CFLAGS) -DEVENT_LOOP_SELECT -o $@ high_perf_lb.c event_loop_select.c $(LDFLAGS)

clean:
	rm -f $(TARGETS) high_perf_lb_select

# Test with multiple backends
test: all
	@echo "=== Chapter 05: High-Performance I/O Test ==="
	@echo "Starting test backends..."
	@python3 -m http.server 9001 --directory /tmp &
	@python3 -m http.server 9002 --directory /tmp &
	@sleep 1
	@echo "Starting load balancer..."
	@./high_perf_lb 8080 127.0.0.1:9001:2 127.0.0.1:9002:1 -a wrr &
	@sleep 1
	@echo "Testing round-robin distribution..."
	@for i in 1 2 3 4 5 6; do curl -s http://localhost:8080/ > /dev/null && echo "Request $$i: OK"; done
	@echo "Killing test processes..."
	@pkill -f "http.server 900" 2>/dev/null || true
	@pkill -f "high_perf_lb" 2>/dev/null || true
	@echo "Test complete!"

# Stress test
stress: all
	@echo "Starting stress test..."
	@./high_perf_lb 8080 127.0.0.1:9001 127.0.0.1:9002 &
	@sleep 1
	@echo "Running wrk (if available)..."
	@wrk -t4 -c100 -d10s http://localhost:8080/ || echo "wrk not installed, skipping"
	@pkill -f "high_perf_lb" 2>/dev/null || true

.PHONY: all clean test stress
