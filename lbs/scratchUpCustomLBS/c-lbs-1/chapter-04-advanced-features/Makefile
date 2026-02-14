# Chapter 04: Advanced Load Balancer
# Makefile

CC = gcc
CFLAGS = -Wall -Wextra -g -O2
LDFLAGS =

TARGETS = advanced_lb

all: $(TARGETS)

advanced_lb: advanced_lb.c
	$(CC) $(CFLAGS) -o $@ $< $(LDFLAGS)

clean:
	rm -f $(TARGETS)

test: all
	@echo "Starting test backends..."
	@python3 -m http.server 9001 --directory /tmp &
	@python3 -m http.server 9002 --directory /tmp &
	@python3 -m http.server 9003 --directory /tmp &
	@sleep 1
	@echo "Starting advanced load balancer..."
	@./advanced_lb 8080 127.0.0.1:9001:3 127.0.0.1:9002:2 127.0.0.1:9003:1 -a wrr &
	@sleep 1
	@echo "Testing weighted distribution..."
	@for i in 1 2 3 4 5 6; do curl -s http://localhost:8080/ > /dev/null && echo "Request $$i OK"; done
	@pkill -f "http.server 900" || true
	@pkill -f "advanced_lb 8080" || true
	@echo "Done!"

.PHONY: all clean test
