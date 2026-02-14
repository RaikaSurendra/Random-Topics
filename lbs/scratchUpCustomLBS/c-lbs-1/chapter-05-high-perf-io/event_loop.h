/**
 * Chapter 05: High-Performance I/O - Event Loop Abstraction
 * ==========================================================
 *
 * Cross-platform event loop supporting:
 * - epoll (Linux) - O(1) event notification
 * - kqueue (macOS/BSD) - O(1) event notification
 * - select (fallback) - O(n) but portable
 *
 * Usage:
 *   event_loop_t *loop = event_loop_create(1024);
 *   event_loop_add(loop, fd, EVENT_READ, callback, user_data);
 *   while (running) {
 *       event_loop_run(loop, 1000);  // 1 second timeout
 *   }
 *   event_loop_destroy(loop);
 */

#ifndef EVENT_LOOP_H
#define EVENT_LOOP_H

#include <stddef.h>
#include <sys/types.h>

// Event types (can be OR'd together)
#define EVENT_READ   (1 << 0)
#define EVENT_WRITE  (1 << 1)
#define EVENT_ERROR  (1 << 2)
#define EVENT_HUP    (1 << 3)

// Event callback signature
typedef void (*event_callback_t)(int fd, int events, void *user_data);

// Opaque event loop structure
typedef struct event_loop event_loop_t;

// Event data passed to callbacks
typedef struct {
    int fd;
    int events;
    void *user_data;
    event_callback_t callback;
} event_data_t;

/**
 * Create a new event loop
 * @param max_events Maximum number of events to handle per iteration
 * @return Event loop instance or NULL on failure
 */
event_loop_t* event_loop_create(int max_events);

/**
 * Destroy event loop and free resources
 * @param loop Event loop to destroy
 */
void event_loop_destroy(event_loop_t *loop);

/**
 * Add file descriptor to event loop
 * @param loop Event loop
 * @param fd File descriptor to monitor
 * @param events Events to monitor (EVENT_READ, EVENT_WRITE, etc.)
 * @param callback Function to call when event occurs
 * @param user_data User data passed to callback
 * @return 0 on success, -1 on failure
 */
int event_loop_add(event_loop_t *loop, int fd, int events,
                   event_callback_t callback, void *user_data);

/**
 * Modify events for existing file descriptor
 * @param loop Event loop
 * @param fd File descriptor
 * @param events New events to monitor
 * @return 0 on success, -1 on failure
 */
int event_loop_mod(event_loop_t *loop, int fd, int events);

/**
 * Remove file descriptor from event loop
 * @param loop Event loop
 * @param fd File descriptor to remove
 * @return 0 on success, -1 on failure
 */
int event_loop_del(event_loop_t *loop, int fd);

/**
 * Run event loop for one iteration
 * @param loop Event loop
 * @param timeout_ms Timeout in milliseconds (-1 for infinite)
 * @return Number of events processed, -1 on error
 */
int event_loop_run(event_loop_t *loop, int timeout_ms);

/**
 * Get the backend name (for logging)
 * @return "epoll", "kqueue", or "select"
 */
const char* event_loop_backend_name(void);

// Platform detection
#if defined(__linux__)
    #define EVENT_LOOP_EPOLL 1
#elif defined(__APPLE__) || defined(__FreeBSD__) || defined(__OpenBSD__) || defined(__NetBSD__)
    #define EVENT_LOOP_KQUEUE 1
#else
    #define EVENT_LOOP_SELECT 1
#endif

#endif // EVENT_LOOP_H
