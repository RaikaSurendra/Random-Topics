/**
 * Chapter 05: kqueue Implementation (macOS/BSD)
 * ==============================================
 *
 * kqueue provides O(1) event notification on BSD systems.
 * Similar to epoll but with different API:
 * - Uses kevent() instead of separate add/wait functions
 * - Supports more event types (file changes, signals, etc.)
 *
 * Compile with: gcc -DEVENT_LOOP_KQUEUE event_loop_kqueue.c
 */

#ifdef EVENT_LOOP_KQUEUE

#include "event_loop.h"
#include <sys/event.h>
#include <sys/time.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <errno.h>

#define MAX_FD_TRACKING 65536

struct event_loop {
    int kqueue_fd;
    struct kevent *events;
    int max_events;

    // Track registered callbacks
    event_data_t *fd_data[MAX_FD_TRACKING];
};

const char* event_loop_backend_name(void) {
    return "kqueue";
}

event_loop_t* event_loop_create(int max_events) {
    event_loop_t *loop = calloc(1, sizeof(event_loop_t));
    if (!loop) return NULL;

    loop->kqueue_fd = kqueue();
    if (loop->kqueue_fd < 0) {
        free(loop);
        return NULL;
    }

    loop->events = calloc(max_events, sizeof(struct kevent));
    if (!loop->events) {
        close(loop->kqueue_fd);
        free(loop);
        return NULL;
    }

    loop->max_events = max_events;
    return loop;
}

void event_loop_destroy(event_loop_t *loop) {
    if (!loop) return;

    for (int i = 0; i < MAX_FD_TRACKING; i++) {
        if (loop->fd_data[i]) {
            free(loop->fd_data[i]);
        }
    }

    close(loop->kqueue_fd);
    free(loop->events);
    free(loop);
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

    // Add to kqueue
    struct kevent changes[2];
    int nchanges = 0;

    if (events & EVENT_READ) {
        EV_SET(&changes[nchanges++], fd, EVFILT_READ,
               EV_ADD | EV_ENABLE, 0, 0, data);
    }
    if (events & EVENT_WRITE) {
        EV_SET(&changes[nchanges++], fd, EVFILT_WRITE,
               EV_ADD | EV_ENABLE, 0, 0, data);
    }

    if (kevent(loop->kqueue_fd, changes, nchanges, NULL, 0, NULL) < 0) {
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

    int old_events = data->events;
    data->events = events;

    struct kevent changes[4];
    int nchanges = 0;

    // Remove old events
    if ((old_events & EVENT_READ) && !(events & EVENT_READ)) {
        EV_SET(&changes[nchanges++], fd, EVFILT_READ, EV_DELETE, 0, 0, NULL);
    }
    if ((old_events & EVENT_WRITE) && !(events & EVENT_WRITE)) {
        EV_SET(&changes[nchanges++], fd, EVFILT_WRITE, EV_DELETE, 0, 0, NULL);
    }

    // Add new events
    if (!(old_events & EVENT_READ) && (events & EVENT_READ)) {
        EV_SET(&changes[nchanges++], fd, EVFILT_READ,
               EV_ADD | EV_ENABLE, 0, 0, data);
    }
    if (!(old_events & EVENT_WRITE) && (events & EVENT_WRITE)) {
        EV_SET(&changes[nchanges++], fd, EVFILT_WRITE,
               EV_ADD | EV_ENABLE, 0, 0, data);
    }

    if (nchanges > 0) {
        if (kevent(loop->kqueue_fd, changes, nchanges, NULL, 0, NULL) < 0) {
            return -1;
        }
    }

    return 0;
}

int event_loop_del(event_loop_t *loop, int fd) {
    if (fd < 0 || fd >= MAX_FD_TRACKING) return -1;

    event_data_t *data = loop->fd_data[fd];
    if (!data) return 0;

    struct kevent changes[2];
    int nchanges = 0;

    if (data->events & EVENT_READ) {
        EV_SET(&changes[nchanges++], fd, EVFILT_READ, EV_DELETE, 0, 0, NULL);
    }
    if (data->events & EVENT_WRITE) {
        EV_SET(&changes[nchanges++], fd, EVFILT_WRITE, EV_DELETE, 0, 0, NULL);
    }

    // Ignore errors on delete (fd might already be closed)
    kevent(loop->kqueue_fd, changes, nchanges, NULL, 0, NULL);

    free(loop->fd_data[fd]);
    loop->fd_data[fd] = NULL;

    return 0;
}

int event_loop_run(event_loop_t *loop, int timeout_ms) {
    struct timespec ts;
    struct timespec *pts = NULL;

    if (timeout_ms >= 0) {
        ts.tv_sec = timeout_ms / 1000;
        ts.tv_nsec = (timeout_ms % 1000) * 1000000;
        pts = &ts;
    }

    int nfds = kevent(loop->kqueue_fd, NULL, 0,
                      loop->events, loop->max_events, pts);

    if (nfds < 0) {
        if (errno == EINTR) return 0;
        return -1;
    }

    for (int i = 0; i < nfds; i++) {
        event_data_t *data = loop->events[i].udata;
        if (data && data->callback) {
            int events = 0;

            if (loop->events[i].filter == EVFILT_READ) {
                events |= EVENT_READ;
            }
            if (loop->events[i].filter == EVFILT_WRITE) {
                events |= EVENT_WRITE;
            }
            if (loop->events[i].flags & EV_EOF) {
                events |= EVENT_HUP;
            }
            if (loop->events[i].flags & EV_ERROR) {
                events |= EVENT_ERROR;
            }

            data->callback(data->fd, events, data->user_data);
        }
    }

    return nfds;
}

#endif // EVENT_LOOP_KQUEUE
