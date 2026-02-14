/**
 * Chapter 06: Connection Pooling - Pingora-Inspired
 * ==================================================
 *
 * Key insight from Pingora (Cloudflare):
 * - Nginx: 87.1% connection reuse (process-isolated pools)
 * - Pingora: 99.92% connection reuse (shared thread pool)
 *
 * This implementation provides:
 * - Shared connection pool across all requests
 * - Keep-alive connection management
 * - Automatic health validation
 * - LRU eviction of idle connections
 *
 * Usage:
 *   conn_pool_t *pool = conn_pool_create(32, 60);
 *   int fd = conn_pool_get(pool, backend);  // Get or create
 *   // ... use connection ...
 *   conn_pool_return(pool, fd, backend);    // Return to pool
 *   conn_pool_destroy(pool);
 */

#ifndef CONN_POOL_H
#define CONN_POOL_H

#include <time.h>
#include <pthread.h>

// Forward declaration
typedef struct Backend Backend;

// Pooled connection state
typedef enum {
    POOL_CONN_FREE,      // Available for use
    POOL_CONN_IN_USE,    // Currently in use
    POOL_CONN_CLOSING    // Being closed
} PoolConnState;

// Single pooled connection
typedef struct PooledConnection {
    int fd;                          // Socket file descriptor
    char backend_host[256];          // Backend identifier
    char backend_port[6];
    time_t last_used;                // For LRU eviction
    time_t created;                  // Connection age
    PoolConnState state;
    int requests_served;             // Connections can serve multiple requests
    struct PooledConnection *next;   // For linked list
    struct PooledConnection *prev;
} PooledConnection;

// Connection pool
typedef struct {
    PooledConnection *connections;   // Array of all connections
    int max_size;                    // Maximum pool size
    int current_size;                // Current number of connections
    int ttl_seconds;                 // Connection TTL
    int max_requests_per_conn;       // Max requests before recycle

    // LRU list (doubly linked)
    PooledConnection *lru_head;      // Most recently used
    PooledConnection *lru_tail;      // Least recently used

    // Statistics
    unsigned long pool_hits;         // Connections reused
    unsigned long pool_misses;       // New connections created
    unsigned long pool_evictions;    // Connections evicted (TTL/LRU)

    // Thread safety
    pthread_mutex_t lock;
} conn_pool_t;

/**
 * Create a new connection pool
 * @param max_size Maximum number of pooled connections
 * @param ttl_seconds Connection time-to-live (0 = no expiry)
 * @return Pool instance or NULL on failure
 */
conn_pool_t* conn_pool_create(int max_size, int ttl_seconds);

/**
 * Destroy connection pool and close all connections
 * @param pool Pool to destroy
 */
void conn_pool_destroy(conn_pool_t *pool);

/**
 * Get a connection from pool (or create new one)
 * @param pool Connection pool
 * @param host Backend host
 * @param port Backend port
 * @return File descriptor or -1 on failure
 */
int conn_pool_get(conn_pool_t *pool, const char *host, const char *port);

/**
 * Return connection to pool for reuse
 * @param pool Connection pool
 * @param fd File descriptor to return
 * @param host Backend host (for validation)
 * @param port Backend port
 */
void conn_pool_return(conn_pool_t *pool, int fd,
                      const char *host, const char *port);

/**
 * Close a connection (don't return to pool)
 * Use when connection is broken/errored
 * @param pool Connection pool
 * @param fd File descriptor to close
 */
void conn_pool_close(conn_pool_t *pool, int fd);

/**
 * Evict expired connections
 * Call periodically (e.g., every second)
 * @param pool Connection pool
 * @return Number of connections evicted
 */
int conn_pool_cleanup(conn_pool_t *pool);

/**
 * Get pool statistics
 */
typedef struct {
    int current_size;
    int max_size;
    unsigned long hits;
    unsigned long misses;
    unsigned long evictions;
    double hit_rate;  // hits / (hits + misses)
} conn_pool_stats_t;

void conn_pool_get_stats(conn_pool_t *pool, conn_pool_stats_t *stats);

/**
 * Check if a connection is still alive (non-blocking)
 * @param fd File descriptor to check
 * @return 1 if alive, 0 if closed/error
 */
int conn_is_alive(int fd);

#endif // CONN_POOL_H
