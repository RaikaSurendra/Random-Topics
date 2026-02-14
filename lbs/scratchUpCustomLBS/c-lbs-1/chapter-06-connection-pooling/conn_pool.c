/**
 * Chapter 06: Connection Pool Implementation
 * ==========================================
 *
 * Pingora achieved 99.92% connection reuse by:
 * 1. Sharing pool across threads (vs Nginx per-process)
 * 2. Keeping connections alive longer
 * 3. Validating connections before reuse
 *
 * This implementation uses:
 * - LRU list for efficient eviction
 * - Mutex for thread safety
 * - Non-blocking health checks
 */

#include "conn_pool.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <netdb.h>
#include <fcntl.h>
#include <errno.h>
#include <poll.h>

// ============================================================================
// Internal Helpers
// ============================================================================

// Check if connection is still alive using poll()
int conn_is_alive(int fd) {
    if (fd < 0) return 0;

    struct pollfd pfd;
    pfd.fd = fd;
    pfd.events = POLLIN;
    pfd.revents = 0;

    // Non-blocking check
    int ret = poll(&pfd, 1, 0);

    if (ret < 0) return 0;  // Error

    if (ret > 0) {
        // Activity on socket
        if (pfd.revents & (POLLERR | POLLHUP | POLLNVAL)) {
            return 0;  // Error or hangup
        }
        if (pfd.revents & POLLIN) {
            // Data available - could be:
            // 1. Server sent data (unexpected for idle connection)
            // 2. Connection closed (read returns 0)
            char buf[1];
            ssize_t n = recv(fd, buf, 1, MSG_PEEK | MSG_DONTWAIT);
            if (n == 0) return 0;  // Connection closed
            if (n < 0 && errno != EAGAIN && errno != EWOULDBLOCK) {
                return 0;  // Error
            }
        }
    }

    return 1;  // Connection appears alive
}

// Create a new connection to backend
static int create_connection(const char *host, const char *port) {
    struct addrinfo hints, *result;
    int fd;

    memset(&hints, 0, sizeof(hints));
    hints.ai_family = AF_INET;
    hints.ai_socktype = SOCK_STREAM;

    if (getaddrinfo(host, port, &hints, &result) != 0) {
        return -1;
    }

    fd = socket(result->ai_family, result->ai_socktype, result->ai_protocol);
    if (fd < 0) {
        freeaddrinfo(result);
        return -1;
    }

    // Set TCP keepalive for pooled connections
    int optval = 1;
    setsockopt(fd, SOL_SOCKET, SO_KEEPALIVE, &optval, sizeof(optval));

    if (connect(fd, result->ai_addr, result->ai_addrlen) < 0) {
        close(fd);
        freeaddrinfo(result);
        return -1;
    }

    freeaddrinfo(result);
    return fd;
}

// Move connection to front of LRU list (most recently used)
static void lru_touch(conn_pool_t *pool, PooledConnection *conn) {
    if (conn == pool->lru_head) return;  // Already at front

    // Remove from current position
    if (conn->prev) conn->prev->next = conn->next;
    if (conn->next) conn->next->prev = conn->prev;
    if (conn == pool->lru_tail) pool->lru_tail = conn->prev;

    // Insert at front
    conn->prev = NULL;
    conn->next = pool->lru_head;
    if (pool->lru_head) pool->lru_head->prev = conn;
    pool->lru_head = conn;
    if (!pool->lru_tail) pool->lru_tail = conn;
}

// Remove connection from LRU list
static void lru_remove(conn_pool_t *pool, PooledConnection *conn) {
    if (conn->prev) conn->prev->next = conn->next;
    if (conn->next) conn->next->prev = conn->prev;
    if (conn == pool->lru_head) pool->lru_head = conn->next;
    if (conn == pool->lru_tail) pool->lru_tail = conn->prev;
    conn->prev = conn->next = NULL;
}

// Find connection by fd
static PooledConnection* find_by_fd(conn_pool_t *pool, int fd) {
    for (int i = 0; i < pool->max_size; i++) {
        if (pool->connections[i].fd == fd) {
            return &pool->connections[i];
        }
    }
    return NULL;
}

// Find free slot in pool
static PooledConnection* find_free_slot(conn_pool_t *pool) {
    for (int i = 0; i < pool->max_size; i++) {
        if (pool->connections[i].state == POOL_CONN_FREE &&
            pool->connections[i].fd < 0) {
            return &pool->connections[i];
        }
    }
    return NULL;
}

// ============================================================================
// Public API
// ============================================================================

conn_pool_t* conn_pool_create(int max_size, int ttl_seconds) {
    conn_pool_t *pool = calloc(1, sizeof(conn_pool_t));
    if (!pool) return NULL;

    pool->connections = calloc(max_size, sizeof(PooledConnection));
    if (!pool->connections) {
        free(pool);
        return NULL;
    }

    pool->max_size = max_size;
    pool->ttl_seconds = ttl_seconds;
    pool->max_requests_per_conn = 1000;  // Default: recycle after 1000 requests

    // Initialize all connections as free
    for (int i = 0; i < max_size; i++) {
        pool->connections[i].fd = -1;
        pool->connections[i].state = POOL_CONN_FREE;
    }

    pthread_mutex_init(&pool->lock, NULL);

    return pool;
}

void conn_pool_destroy(conn_pool_t *pool) {
    if (!pool) return;

    pthread_mutex_lock(&pool->lock);

    // Close all connections
    for (int i = 0; i < pool->max_size; i++) {
        if (pool->connections[i].fd >= 0) {
            close(pool->connections[i].fd);
        }
    }

    pthread_mutex_unlock(&pool->lock);
    pthread_mutex_destroy(&pool->lock);

    free(pool->connections);
    free(pool);
}

int conn_pool_get(conn_pool_t *pool, const char *host, const char *port) {
    pthread_mutex_lock(&pool->lock);

    time_t now = time(NULL);

    // First, try to find an existing pooled connection to this backend
    for (int i = 0; i < pool->max_size; i++) {
        PooledConnection *conn = &pool->connections[i];

        if (conn->state == POOL_CONN_FREE && conn->fd >= 0 &&
            strcmp(conn->backend_host, host) == 0 &&
            strcmp(conn->backend_port, port) == 0) {

            // Check TTL
            if (pool->ttl_seconds > 0 &&
                (now - conn->created) > pool->ttl_seconds) {
                // Expired - close and continue searching
                close(conn->fd);
                conn->fd = -1;
                pool->current_size--;
                pool->pool_evictions++;
                continue;
            }

            // Check max requests
            if (conn->requests_served >= pool->max_requests_per_conn) {
                close(conn->fd);
                conn->fd = -1;
                pool->current_size--;
                continue;
            }

            // Verify connection is still alive
            if (!conn_is_alive(conn->fd)) {
                close(conn->fd);
                conn->fd = -1;
                pool->current_size--;
                continue;
            }

            // Found a valid pooled connection!
            conn->state = POOL_CONN_IN_USE;
            conn->last_used = now;
            conn->requests_served++;
            lru_touch(pool, conn);
            pool->pool_hits++;

            int fd = conn->fd;
            pthread_mutex_unlock(&pool->lock);
            return fd;
        }
    }

    // No pooled connection available - create new one
    pool->pool_misses++;

    // Find a free slot (or evict LRU if full)
    PooledConnection *slot = find_free_slot(pool);

    if (!slot && pool->lru_tail) {
        // Pool is full - evict least recently used
        slot = pool->lru_tail;
        if (slot->fd >= 0) {
            close(slot->fd);
            pool->current_size--;
            pool->pool_evictions++;
        }
        lru_remove(pool, slot);
    }

    if (!slot) {
        pthread_mutex_unlock(&pool->lock);
        // Can't find slot even after eviction - create connection without pooling
        return create_connection(host, port);
    }

    pthread_mutex_unlock(&pool->lock);

    // Create new connection (outside lock to avoid blocking)
    int fd = create_connection(host, port);

    pthread_mutex_lock(&pool->lock);

    if (fd >= 0) {
        slot->fd = fd;
        strncpy(slot->backend_host, host, sizeof(slot->backend_host) - 1);
        strncpy(slot->backend_port, port, sizeof(slot->backend_port) - 1);
        slot->created = now;
        slot->last_used = now;
        slot->state = POOL_CONN_IN_USE;
        slot->requests_served = 1;
        pool->current_size++;
        lru_touch(pool, slot);
    }

    pthread_mutex_unlock(&pool->lock);
    return fd;
}

void conn_pool_return(conn_pool_t *pool, int fd,
                      const char *host, const char *port) {
    if (fd < 0) return;

    pthread_mutex_lock(&pool->lock);

    PooledConnection *conn = find_by_fd(pool, fd);

    if (conn) {
        // Validate it's the same backend
        if (strcmp(conn->backend_host, host) == 0 &&
            strcmp(conn->backend_port, port) == 0) {

            // Check if connection is still usable
            if (conn_is_alive(fd) &&
                conn->requests_served < pool->max_requests_per_conn) {

                conn->state = POOL_CONN_FREE;
                conn->last_used = time(NULL);
                lru_touch(pool, conn);
                pthread_mutex_unlock(&pool->lock);
                return;
            }
        }

        // Connection is bad or backend mismatch - close it
        close(fd);
        conn->fd = -1;
        conn->state = POOL_CONN_FREE;
        lru_remove(pool, conn);
        pool->current_size--;
    } else {
        // Not in pool - just close
        close(fd);
    }

    pthread_mutex_unlock(&pool->lock);
}

void conn_pool_close(conn_pool_t *pool, int fd) {
    if (fd < 0) return;

    pthread_mutex_lock(&pool->lock);

    PooledConnection *conn = find_by_fd(pool, fd);

    if (conn) {
        close(fd);
        conn->fd = -1;
        conn->state = POOL_CONN_FREE;
        lru_remove(pool, conn);
        pool->current_size--;
    } else {
        close(fd);
    }

    pthread_mutex_unlock(&pool->lock);
}

int conn_pool_cleanup(conn_pool_t *pool) {
    pthread_mutex_lock(&pool->lock);

    int evicted = 0;
    time_t now = time(NULL);

    for (int i = 0; i < pool->max_size; i++) {
        PooledConnection *conn = &pool->connections[i];

        if (conn->state == POOL_CONN_FREE && conn->fd >= 0) {
            int should_evict = 0;

            // Check TTL
            if (pool->ttl_seconds > 0 &&
                (now - conn->created) > pool->ttl_seconds) {
                should_evict = 1;
            }

            // Check idle time (evict if idle > 30 seconds)
            if ((now - conn->last_used) > 30) {
                should_evict = 1;
            }

            // Check connection health
            if (!conn_is_alive(conn->fd)) {
                should_evict = 1;
            }

            if (should_evict) {
                close(conn->fd);
                conn->fd = -1;
                lru_remove(pool, conn);
                pool->current_size--;
                pool->pool_evictions++;
                evicted++;
            }
        }
    }

    pthread_mutex_unlock(&pool->lock);
    return evicted;
}

void conn_pool_get_stats(conn_pool_t *pool, conn_pool_stats_t *stats) {
    pthread_mutex_lock(&pool->lock);

    stats->current_size = pool->current_size;
    stats->max_size = pool->max_size;
    stats->hits = pool->pool_hits;
    stats->misses = pool->pool_misses;
    stats->evictions = pool->pool_evictions;

    unsigned long total = stats->hits + stats->misses;
    stats->hit_rate = total > 0 ? (double)stats->hits / total * 100.0 : 0.0;

    pthread_mutex_unlock(&pool->lock);
}
