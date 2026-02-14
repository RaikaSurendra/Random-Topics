# Chapter 03: Load Balancer
# Makefile

CC = gcc
CFLAGS = -Wall -Wextra -g -O2
LDFLAGS =

TARGETS = load_balancer

all: $(TARGETS)

load_balancer: load_balancer.c
	$(CC) $(CFLAGS) -o $@ $< $(LDFLAGS)

clean:
	rm -f $(TARGETS)

# Build echo_server from Chapter 01 for testing
../chapter-01-fundamentals/echo_server:
	$(MAKE) -C ../chapter-01-fundamentals echo_server

# Test with 3 backends
test: all ../chapter-01-fundamentals/echo_server
	@echo "Starting 3 backend servers..."
	@../chapter-01-fundamentals/echo_server 9001 > /dev/null 2>&1 &
	@../chapter-01-fundamentals/echo_server 9002 > /dev/null 2>&1 &
	@../chapter-01-fundamentals/echo_server 9003 > /dev/null 2>&1 &
	@sleep 1
	@echo "Starting load balancer..."
	@./load_balancer 8080 127.0.0.1:9001 127.0.0.1:9002 127.0.0.1:9003 &
	@sleep 1
	@echo "\nSending 6 requests (should distribute 2 each)...\n"
	@for i in 1 2 3 4 5 6; do \
		echo "Request $$i" | nc -q1 localhost 8080 2>/dev/null || echo "Request $$i" | nc localhost 8080; \
	done
	@echo "\nStopping services..."
	@pkill -f "echo_server 900" || true
	@pkill -f "load_balancer 8080" || true
	@echo "Test complete!"

.PHONY: all clean test
