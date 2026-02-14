# Chapter 06: Connection Pooling
# Pingora-inspired backend connection reuse

CC = gcc
CFLAGS = -Wall -Wextra -g -O2 -pthread
LDFLAGS = -pthread

# Platform detection for event loop
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Linux)
    EVENT_BACKEND = ../chapter-05-high-perf-io/event_loop_epoll.c
    BACKEND_FLAG = -DEVENT_LOOP_EPOLL
else ifeq ($(UNAME_S),Darwin)
    EVENT_BACKEND = ../chapter-05-high-perf-io/event_loop_kqueue.c
    BACKEND_FLAG = -DEVENT_LOOP_KQUEUE
else
    EVENT_BACKEND = ../chapter-05-high-perf-io/event_loop_select.c
    BACKEND_FLAG = -DEVENT_LOOP_SELECT
endif

TARGETS = pooled_lb

all: $(TARGETS)

pooled_lb: pooled_lb.c conn_pool.c conn_pool.h $(EVENT_BACKEND)
	$(CC) $(CFLAGS) $(BACKEND_FLAG) -I../chapter-05-high-perf-io \
		-o $@ pooled_lb.c conn_pool.c $(EVENT_BACKEND) $(LDFLAGS)

clean:
	rm -f $(TARGETS)

test: all
	@echo "=== Chapter 06: Connection Pooling Test ==="
	@echo "Starting backends..."
	@python3 -m http.server 9001 --directory /tmp &
	@python3 -m http.server 9002 --directory /tmp &
	@sleep 1
	@echo "Starting pooled load balancer..."
	@./pooled_lb 8080 127.0.0.1:9001 127.0.0.1:9002 -p 8 &
	@sleep 1
	@echo "Sending 20 requests (should see high pool hit rate)..."
	@for i in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20; do \
		curl -s http://localhost:8080/ > /dev/null; \
	done
	@sleep 1
	@echo "Checking stats (send SIGUSR1)..."
	@pkill -USR1 -f "pooled_lb" 2>/dev/null || true
	@sleep 2
	@echo "Cleanup..."
	@pkill -f "http.server 900" 2>/dev/null || true
	@pkill -f "pooled_lb" 2>/dev/null || true
	@echo "Test complete!"

.PHONY: all clean test
