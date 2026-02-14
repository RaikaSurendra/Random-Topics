/**
 * Chapter 08: Metrics & Prometheus
 * =================================
 *
 * Production-grade observability with:
 * - Counters (monotonically increasing values)
 * - Gauges (point-in-time values)
 * - Histograms (latency distributions)
 * - Prometheus text format export
 *
 * Usage:
 *   metrics_t *m = metrics_create();
 *   metrics_counter_inc(m, "requests_total", labels);
 *   metrics_gauge_set(m, "connections_active", 42, labels);
 *   metrics_histogram_observe(m, "request_duration_seconds", 0.025, labels);
 *   metrics_expose(m, client_fd);  // Serve /metrics endpoint
 */

#ifndef METRICS_H
#define METRICS_H

#include <stdint.h>
#include <pthread.h>

#define MAX_METRICS 256
#define MAX_LABELS 8
#define MAX_HISTOGRAM_BUCKETS 16

// Metric types (Prometheus compatible)
typedef enum {
    METRIC_COUNTER,     // Monotonically increasing
    METRIC_GAUGE,       // Can go up or down
    METRIC_HISTOGRAM    // Distribution with buckets
} MetricType;

// Label key-value pair
typedef struct {
    char key[64];
    char value[128];
} Label;

// Single metric
typedef struct {
    char name[128];
    char help[256];
    MetricType type;

    // Labels for this metric instance
    Label labels[MAX_LABELS];
    int num_labels;

    // Value storage
    union {
        double counter;
        double gauge;
        struct {
            double buckets[MAX_HISTOGRAM_BUCKETS];
            double bucket_bounds[MAX_HISTOGRAM_BUCKETS];
            int num_buckets;
            double sum;
            uint64_t count;
        } histogram;
    } value;

    struct Metric *next;  // For hash chain
} Metric;

// Metrics registry
typedef struct {
    Metric *metrics[MAX_METRICS];
    int num_metrics;
    pthread_mutex_t lock;

    // Default histogram buckets (latency in seconds)
    double default_buckets[MAX_HISTOGRAM_BUCKETS];
    int num_default_buckets;
} metrics_t;

/**
 * Create metrics registry
 */
metrics_t* metrics_create(void);

/**
 * Destroy metrics registry
 */
void metrics_destroy(metrics_t *m);

/**
 * Register a new metric
 * @param m Metrics registry
 * @param name Metric name (e.g., "http_requests_total")
 * @param help Description
 * @param type Metric type
 * @return 0 on success
 */
int metrics_register(metrics_t *m, const char *name, const char *help,
                     MetricType type);

/**
 * Increment counter
 * @param labels NULL-terminated array of label strings: "key1", "value1", "key2", "value2", NULL
 */
void metrics_counter_inc(metrics_t *m, const char *name, const char **labels);

/**
 * Add to counter
 */
void metrics_counter_add(metrics_t *m, const char *name, double value,
                         const char **labels);

/**
 * Set gauge value
 */
void metrics_gauge_set(metrics_t *m, const char *name, double value,
                       const char **labels);

/**
 * Increment gauge
 */
void metrics_gauge_inc(metrics_t *m, const char *name, const char **labels);

/**
 * Decrement gauge
 */
void metrics_gauge_dec(metrics_t *m, const char *name, const char **labels);

/**
 * Record histogram observation
 */
void metrics_histogram_observe(metrics_t *m, const char *name, double value,
                               const char **labels);

/**
 * Expose metrics in Prometheus text format
 * @param m Metrics registry
 * @param fd File descriptor to write to
 * @return Bytes written
 */
int metrics_expose(metrics_t *m, int fd);

/**
 * Format metrics to buffer (for HTTP response)
 * @param m Metrics registry
 * @param buffer Output buffer
 * @param size Buffer size
 * @return Bytes written
 */
int metrics_format(metrics_t *m, char *buffer, size_t size);

// Standard load balancer metrics
void metrics_register_lb_defaults(metrics_t *m);

#endif // METRICS_H
