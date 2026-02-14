/**
 * Chapter 05: select() Fallback Implementation
 * ============================================
 *
 * Portable fallback using POSIX select().
 * Works everywhere but has limitations:
 * - FD_SETSIZE limit (usually 1024)
 * - O(n) scanning per iteration
 * - Must rebuild fd_set every time
 *
 * Compile with: gcc -DEVENT_LOOP_SELECT event_loop_select.c
 */

#ifdef EVENT_LOOP_SELECT

#include "event_loop.h"
#include <sys/select.h>
#include <stdlib.h>
#include <string.h>
#include <errno.h>

#ifndef FD_SETSIZE
#define FD_SETSIZE 1024
#endif

struct event_loop {
    int max_events;
    int max_fd;

    // Track registered fds and their data
    event_data_t *fd_data[FD_SETSIZE];
};

const char* event_loop_backend_name(void) {
    return "select";
}

event_loop_t* event_loop_create(int max_events) {
    (void)max_events;  // Ignored for select

    event_loop_t *loop = calloc(1, sizeof(event_loop_t));
    if (!loop) return NULL;

    loop->max_events = FD_SETSIZE;
    loop->max_fd = -1;

    return loop;
}

void event_loop_destroy(event_loop_t *loop) {
    if (!loop) return;

    for (int i = 0; i < FD_SETSIZE; i++) {
        if (loop->fd_data[i]) {
            free(loop->fd_data[i]);
        }
    }

    free(loop);
}

int event_loop_add(event_loop_t *loop, int fd, int events,
                   event_callback_t callback, void *user_data) {
    if (fd < 0 || fd >= FD_SETSIZE) return -1;

    // Allocate event data
    event_data_t *data = malloc(sizeof(event_data_t));
    if (!data) return -1;

    data->fd = fd;
    data->events = events;
    data->callback = callback;
    data->user_data = user_data;

    // Store in tracking array
    if (loop->fd_data[fd]) {
        free(loop->fd_data[fd]);
    }
    loop->fd_data[fd] = data;

    // Update max_fd
    if (fd > loop->max_fd) {
        loop->max_fd = fd;
    }

    return 0;
}

int event_loop_mod(event_loop_t *loop, int fd, int events) {
    if (fd < 0 || fd >= FD_SETSIZE) return -1;

    event_data_t *data = loop->fd_data[fd];
    if (!data) return -1;

    data->events = events;
    return 0;
}

int event_loop_del(event_loop_t *loop, int fd) {
    if (fd < 0 || fd >= FD_SETSIZE) return -1;

    if (loop->fd_data[fd]) {
        free(loop->fd_data[fd]);
        loop->fd_data[fd] = NULL;
    }

    // Recalculate max_fd
    if (fd == loop->max_fd) {
        loop->max_fd = -1;
        for (int i = fd - 1; i >= 0; i--) {
            if (loop->fd_data[i]) {
                loop->max_fd = i;
                break;
            }
        }
    }

    return 0;
}

int event_loop_run(event_loop_t *loop, int timeout_ms) {
    if (loop->max_fd < 0) {
        // No fds to monitor
        if (timeout_ms > 0) {
            struct timeval tv = {
                .tv_sec = timeout_ms / 1000,
                .tv_usec = (timeout_ms % 1000) * 1000
            };
            select(0, NULL, NULL, NULL, &tv);
        }
        return 0;
    }

    fd_set read_fds, write_fds, error_fds;
    FD_ZERO(&read_fds);
    FD_ZERO(&write_fds);
    FD_ZERO(&error_fds);

    // Build fd_sets
    for (int fd = 0; fd <= loop->max_fd; fd++) {
        event_data_t *data = loop->fd_data[fd];
        if (!data) continue;

        if (data->events & EVENT_READ) {
            FD_SET(fd, &read_fds);
        }
        if (data->events & EVENT_WRITE) {
            FD_SET(fd, &write_fds);
        }
        FD_SET(fd, &error_fds);
    }

    struct timeval tv;
    struct timeval *ptv = NULL;
    if (timeout_ms >= 0) {
        tv.tv_sec = timeout_ms / 1000;
        tv.tv_usec = (timeout_ms % 1000) * 1000;
        ptv = &tv;
    }

    int ready = select(loop->max_fd + 1, &read_fds, &write_fds, &error_fds, ptv);

    if (ready < 0) {
        if (errno == EINTR) return 0;
        return -1;
    }

    int processed = 0;
    for (int fd = 0; fd <= loop->max_fd && processed < ready; fd++) {
        event_data_t *data = loop->fd_data[fd];
        if (!data) continue;

        int events = 0;
        if (FD_ISSET(fd, &read_fds))  events |= EVENT_READ;
        if (FD_ISSET(fd, &write_fds)) events |= EVENT_WRITE;
        if (FD_ISSET(fd, &error_fds)) events |= EVENT_ERROR;

        if (events && data->callback) {
            data->callback(fd, events, data->user_data);
            processed++;
        }
    }

    return ready;
}

#endif // EVENT_LOOP_SELECT
