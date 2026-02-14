# Chapter 09: Zero-Copy I/O

CC = gcc
CFLAGS = -Wall -Wextra -g -O2
LDFLAGS =

all: zero_copy_test

zero_copy_test: zero_copy.c zero_copy.h
	$(CC) $(CFLAGS) -DZERO_COPY_TEST -o $@ zero_copy.c $(LDFLAGS)

clean:
	rm -f zero_copy_test

test: zero_copy_test
	@echo "=== Chapter 09: Zero-Copy Test ==="
	@echo "Backend: $$(./zero_copy_test backend)"
	./zero_copy_test

.PHONY: all clean test
