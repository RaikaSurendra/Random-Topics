# Chapter 01: Fundamentals
# Makefile for socket programming basics

CC = gcc
CFLAGS = -Wall -Wextra -g -O2
LDFLAGS =

# Targets
TARGETS = echo_server echo_client getaddrinfo_demo

all: $(TARGETS)

echo_server: echo_server.c
	$(CC) $(CFLAGS) -o $@ $< $(LDFLAGS)

echo_client: echo_client.c
	$(CC) $(CFLAGS) -o $@ $< $(LDFLAGS)

getaddrinfo_demo: getaddrinfo_demo.c
	$(CC) $(CFLAGS) -o $@ $< $(LDFLAGS)

clean:
	rm -f $(TARGETS)

# Test target
test: all
	@echo "Starting echo server on port 8080..."
	@./echo_server 8080 &
	@sleep 1
	@echo "Testing with echo client..."
	@echo "Hello World" | ./echo_client 127.0.0.1 8080
	@pkill -f "echo_server 8080" || true
	@echo "Test complete!"

.PHONY: all clean test
