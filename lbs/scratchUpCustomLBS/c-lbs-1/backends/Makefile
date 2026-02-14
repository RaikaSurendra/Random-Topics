# Backend Test Servers
# Makefile

CC = gcc
CFLAGS = -Wall -Wextra -g -O2

all: simple_http_backend

simple_http_backend: simple_http_backend.c
	$(CC) $(CFLAGS) -o $@ $<

clean:
	rm -f simple_http_backend

# Start 3 backends for testing
start:
	@./simple_http_backend 9001 backend-1 &
	@./simple_http_backend 9002 backend-2 &
	@./simple_http_backend 9003 backend-3 &
	@echo "Started backends on ports 9001, 9002, 9003"

stop:
	@pkill -f "simple_http_backend" || true
	@echo "Stopped all backends"

.PHONY: all clean start stop
