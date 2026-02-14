# Chapter 08: Metrics & Prometheus

CC = gcc
CFLAGS = -Wall -Wextra -g -O2 -pthread
LDFLAGS = -pthread

all: metrics_test

metrics_test: metrics.c metrics.h
	$(CC) $(CFLAGS) -DMETRICS_TEST -o $@ metrics.c $(LDFLAGS)

clean:
	rm -f metrics_test

test: metrics_test
	@echo "=== Chapter 08: Metrics Test ==="
	./metrics_test

.PHONY: all clean test
