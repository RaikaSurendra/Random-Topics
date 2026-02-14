# Chapter 07: Rate Limiting

CC = gcc
CFLAGS = -Wall -Wextra -g -O2 -pthread
LDFLAGS = -pthread

all: rate_limiter_test

rate_limiter_test: rate_limiter.c rate_limiter.h
	$(CC) $(CFLAGS) -DRATE_LIMITER_TEST -o $@ rate_limiter.c $(LDFLAGS)

clean:
	rm -f rate_limiter_test

test: rate_limiter_test
	@echo "=== Chapter 07: Rate Limiting Test ==="
	./rate_limiter_test

.PHONY: all clean test
