/**
 * Chapter 07: Rate Limiting
 * =========================
 *
 * Protects backends from overload with multiple algorithms:
 * - Token Bucket: Smooth rate limiting with burst allowance
 * - Sliding Window: Accurate per-second rate tracking
 * - Fixed Window: Simple counter-based limiting
 *
 * Usage:
 *   rate_limiter_t *limiter = rate_limiter_create(RATE_TOKEN_BUCKET, 100, 10);
 *   if (rate_limiter_allow(limiter, client_ip)) {
 *       // Process request
 *   } else {
 *       // Return 429 Too Many Requests
 *   }
 */

#ifndef RATE_LIMITER_H
#define RATE_LIMITER_H

#include <time.h>
#include <pthread.h>

// Rate limiting algorithms
typedef enum {
    RATE_TOKEN_BUCKET,      // Smooth limiting with burst
    RATE_SLIDING_WINDOW,    // Accurate per-second
    RATE_FIXED_WINDOW       // Simple counters
} RateLimitAlgorithm;

// Per-client rate state
typedef struct {
    char key[64];           // Client identifier (IP, API key, etc.)
    double tokens;          // Token bucket: current tokens
    long window_count;      // Window: requests in window
    time_t window_start;    // Window start time
    time_t last_update;     // Last token refill
    struct RateEntry *next; // Hash chain
} RateEntry;

// Rate limiter
typedef struct {
    RateLimitAlgorithm algorithm;
    double rate;            // Requests per second
    double burst;           // Max burst size (token bucket)
    int window_size;        // Window size in seconds

    RateEntry **buckets;    // Hash table
    int num_buckets;

    // Global limits
    long global_count;
    long global_limit;
    time_t global_window_start;

    pthread_mutex_t lock;

    // Statistics
    unsigned long allowed;
    unsigned long denied;
} rate_limiter_t;

/**
 * Create rate limiter
 * @param algorithm Rate limiting algorithm
 * @param rate Requests per second allowed
 * @param burst Burst size (for token bucket) or window size (for windows)
 * @return Rate limiter or NULL
 */
rate_limiter_t* rate_limiter_create(RateLimitAlgorithm algorithm,
                                     double rate, double burst);

/**
 * Destroy rate limiter
 */
void rate_limiter_destroy(rate_limiter_t *limiter);

/**
 * Check if request should be allowed
 * @param limiter Rate limiter
 * @param key Client identifier (IP address, etc.)
 * @return 1 if allowed, 0 if rate limited
 */
int rate_limiter_allow(rate_limiter_t *limiter, const char *key);

/**
 * Set global rate limit (across all clients)
 * @param limiter Rate limiter
 * @param limit Global requests per second
 */
void rate_limiter_set_global(rate_limiter_t *limiter, long limit);

/**
 * Get remaining tokens/requests for a key
 */
double rate_limiter_remaining(rate_limiter_t *limiter, const char *key);

/**
 * Get rate limiter statistics
 */
typedef struct {
    unsigned long allowed;
    unsigned long denied;
    double denial_rate;
    int active_clients;
} rate_limiter_stats_t;

void rate_limiter_get_stats(rate_limiter_t *limiter, rate_limiter_stats_t *stats);

/**
 * Cleanup expired entries
 */
int rate_limiter_cleanup(rate_limiter_t *limiter);

#endif // RATE_LIMITER_H
