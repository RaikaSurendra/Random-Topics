# Chapter 10: Hot Reload & Configuration

CC = gcc
CFLAGS = -Wall -Wextra -g -O2
LDFLAGS =

all: hot_reload_lb config_test

hot_reload_lb: config.c config.h
	$(CC) $(CFLAGS) -DHOT_RELOAD_LB -o $@ config.c $(LDFLAGS)

config_test: config.c config.h
	$(CC) $(CFLAGS) -DCONFIG_TEST -o $@ config.c $(LDFLAGS)

clean:
	rm -f hot_reload_lb config_test

test: hot_reload_lb
	@echo "=== Chapter 10: Hot Reload Test ==="
	./hot_reload_lb lb.json

.PHONY: all clean test
