/**
 * Chapter 05: epoll Implementation (Linux)
 * =========================================
 *
 * epoll provides O(1) event notification on Linux.
 * Key advantages over select():
 * - No FD_SETSIZE limit
 * - Kernel maintains watch list (no rebuild per iteration)
 * - Edge-triggered mode for efficiency
 *
 * Compile with: gcc -DEVENT_LOOP_EPOLL event_loop_epoll.c
 */

#ifdef EVENT_LOOP_EPOLL

#include "event_loop.h"
#include <sys/epoll.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <errno.h>

#define MAX_FD_TRACKING 65536

struct event_loop {
    int epoll_fd;
    struct epoll_event *events;
    int max_events;

    // Track registered callbacks
    event_data_t *fd_data[MAX_FD_TRACKING];
};

const char* event_loop_backend_name(void) {
    return "epoll";
}

event_loop_t* event_loop_create(int max_events) {
    event_loop_t *loop = calloc(1, sizeof(event_loop_t));
    if (!loop) return NULL;

    loop->epoll_fd = epoll_create1(EPOLL_CLOEXEC);
    if (loop->epoll_fd < 0) {
        free(loop);
        return NULL;
    }

    loop->events = calloc(max_events, sizeof(struct epoll_event));
    if (!loop->events) {
        close(loop->epoll_fd);
        free(loop);
        return NULL;
    }

    loop->max_events = max_events;
    return loop;
}

void event_loop_destroy(event_loop_t *loop) {
    if (!loop) return;

    // Free all tracked fd data
    for (int i = 0; i < MAX_FD_TRACKING; i++) {
        if (loop->fd_data[i]) {
            free(loop->fd_data[i]);
        }
    }

    close(loop->epoll_fd);
    free(loop->events);
    free(loop);
}

static uint32_t events_to_epoll(int events) {
    uint32_t ep_events = 0;
    if (events & EVENT_READ)  ep_events |= EPOLLIN;
    if (events & EVENT_WRITE) ep_events |= EPOLLOUT;
    // Always monitor for errors and hangups
    ep_events |= EPOLLERR | EPOLLHUP;
    return ep_events;
}

static int epoll_to_events(uint32_t ep_events) {
    int events = 0;
    if (ep_events & EPOLLIN)  events |= EVENT_READ;
    if (ep_events & EPOLLOUT) events |= EVENT_WRITE;
    if (ep_events & EPOLLERR) events |= EVENT_ERROR;
    if (ep_events & EPOLLHUP) events |= EVENT_HUP;
    return events;
}

int event_loop_add(event_loop_t *loop, int fd, int events,
                   event_callback_t callback, void *user_data) {
    if (fd < 0 || fd >= MAX_FD_TRACKING) return -1;

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

    // Add to epoll
    struct epoll_event ev;
    ev.events = events_to_epoll(events);
    ev.data.ptr = data;

    if (epoll_ctl(loop->epoll_fd, EPOLL_CTL_ADD, fd, &ev) < 0) {
        // Try modify if already exists
        if (errno == EEXIST) {
            return epoll_ctl(loop->epoll_fd, EPOLL_CTL_MOD, fd, &ev);
        }
        free(data);
        loop->fd_data[fd] = NULL;
        return -1;
    }

    return 0;
}

int event_loop_mod(event_loop_t *loop, int fd, int events) {
    if (fd < 0 || fd >= MAX_FD_TRACKING) return -1;

    event_data_t *data = loop->fd_data[fd];
    if (!data) return -1;

    data->events = events;

    struct epoll_event ev;
    ev.events = events_to_epoll(events);
    ev.data.ptr = data;

    return epoll_ctl(loop->epoll_fd, EPOLL_CTL_MOD, fd, &ev);
}

int event_loop_del(event_loop_t *loop, int fd) {
    if (fd < 0 || fd >= MAX_FD_TRACKING) return -1;

    if (loop->fd_data[fd]) {
        free(loop->fd_data[fd]);
        loop->fd_data[fd] = NULL;
    }

    return epoll_ctl(loop->epoll_fd, EPOLL_CTL_DEL, fd, NULL);
}

int event_loop_run(event_loop_t *loop, int timeout_ms) {
    int nfds = epoll_wait(loop->epoll_fd, loop->events,
                          loop->max_events, timeout_ms);

    if (nfds < 0) {
        if (errno == EINTR) return 0;
        return -1;
    }

    for (int i = 0; i < nfds; i++) {
        event_data_t *data = loop->events[i].data.ptr;
        if (data && data->callback) {
            int events = epoll_to_events(loop->events[i].events);
            data->callback(data->fd, events, data->user_data);
        }
    }

    return nfds;
}

#endif // EVENT_LOOP_EPOLL
