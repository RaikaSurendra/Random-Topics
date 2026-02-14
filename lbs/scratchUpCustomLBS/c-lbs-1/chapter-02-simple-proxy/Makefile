# Chapter 02: Forward & Reverse Proxy
# Makefile

CC = gcc
CFLAGS = -Wall -Wextra -g -O2
LDFLAGS =

TARGETS = reverse_proxy forward_proxy

all: $(TARGETS)

reverse_proxy: reverse_proxy.c
	$(CC) $(CFLAGS) -o $@ $< $(LDFLAGS)

forward_proxy: forward_proxy.c
	$(CC) $(CFLAGS) -o $@ $< $(LDFLAGS)

clean:
	rm -f $(TARGETS)

# Test reverse proxy
test-reverse: reverse_proxy
	@echo "Starting Python HTTP server on port 9000..."
	@python3 -m http.server 9000 &
	@sleep 1
	@echo "Starting reverse proxy on port 8080..."
	@./reverse_proxy 8080 127.0.0.1 9000 &
	@sleep 1
	@echo "Testing..."
	@curl -s http://localhost:8080/ | head -5
	@pkill -f "http.server 9000" || true
	@pkill -f "reverse_proxy 8080" || true
	@echo "\nTest complete!"

.PHONY: all clean test-reverse
