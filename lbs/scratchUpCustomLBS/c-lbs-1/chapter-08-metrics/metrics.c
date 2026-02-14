/**
 * Chapter 08: Metrics Implementation
 * ===================================
 * Prometheus-compatible metrics collection
 */

#include "metrics.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

static unsigned int hash_metric(const char *name) {
    unsigned int hash = 5381;
    while (*name) {
        hash = ((hash << 5) + hash) + *name++;
    }
    return hash % MAX_METRICS;
}

metrics_t* metrics_create(void) {
    metrics_t *m = calloc(1, sizeof(metrics_t));
    if (!m) return NULL;

    pthread_mutex_init(&m->lock, NULL);

    // Default histogram buckets (request latency in seconds)
    double buckets[] = {0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0};
    m->num_default_buckets = sizeof(buckets) / sizeof(buckets[0]);
    memcpy(m->default_buckets, buckets, sizeof(buckets));

    return m;
}

void metrics_destroy(metrics_t *m) {
    if (!m) return;

    pthread_mutex_lock(&m->lock);

    for (int i = 0; i < MAX_METRICS; i++) {
        Metric *metric = m->metrics[i];
        while (metric) {
            Metric *next = (Metric*)metric->next;
            free(metric);
            metric = next;
        }
    }

    pthread_mutex_unlock(&m->lock);
    pthread_mutex_destroy(&m->lock);
    free(m);
}

static Metric* find_or_create_metric(metrics_t *m, const char *name,
                                      MetricType type, const char **labels) {
    unsigned int idx = hash_metric(name);

    // Build labels key
    char labels_key[1024] = "";
    if (labels) {
        for (int i = 0; labels[i] && labels[i + 1]; i += 2) {
            char buf[256];
            snprintf(buf, sizeof(buf), "%s=%s,", labels[i], labels[i + 1]);
            strncat(labels_key, buf, sizeof(labels_key) - strlen(labels_key) - 1);
        }
    }

    // Search for existing
    Metric *metric = m->metrics[idx];
    while (metric) {
        if (strcmp(metric->name, name) == 0) {
            // Check labels match
            char existing_key[1024] = "";
            for (int i = 0; i < metric->num_labels; i++) {
                char buf[256];
                snprintf(buf, sizeof(buf), "%s=%s,",
                        metric->labels[i].key, metric->labels[i].value);
                strncat(existing_key, buf, sizeof(existing_key) - strlen(existing_key) - 1);
            }
            if (strcmp(labels_key, existing_key) == 0) {
                return metric;
            }
        }
        metric = (Metric*)metric->next;
    }

    // Create new
    metric = calloc(1, sizeof(Metric));
    if (!metric) return NULL;

    strncpy(metric->name, name, sizeof(metric->name) - 1);
    metric->type = type;

    // Copy labels
    if (labels) {
        for (int i = 0; labels[i] && labels[i + 1] && metric->num_labels < MAX_LABELS; i += 2) {
            strncpy(metric->labels[metric->num_labels].key, labels[i], 63);
            strncpy(metric->labels[metric->num_labels].value, labels[i + 1], 127);
            metric->num_labels++;
        }
    }

    // Initialize histogram buckets
    if (type == METRIC_HISTOGRAM) {
        metric->value.histogram.num_buckets = m->num_default_buckets;
        memcpy(metric->value.histogram.bucket_bounds, m->default_buckets,
               m->num_default_buckets * sizeof(double));
    }

    metric->next = (struct Metric*)m->metrics[idx];
    m->metrics[idx] = metric;
    m->num_metrics++;

    return metric;
}

int metrics_register(metrics_t *m, const char *name, const char *help,
                     MetricType type) {
    pthread_mutex_lock(&m->lock);

    Metric *metric = find_or_create_metric(m, name, type, NULL);
    if (metric && help) {
        strncpy(metric->help, help, sizeof(metric->help) - 1);
    }

    pthread_mutex_unlock(&m->lock);
    return metric ? 0 : -1;
}

void metrics_counter_inc(metrics_t *m, const char *name, const char **labels) {
    metrics_counter_add(m, name, 1.0, labels);
}

void metrics_counter_add(metrics_t *m, const char *name, double value,
                         const char **labels) {
    pthread_mutex_lock(&m->lock);

    Metric *metric = find_or_create_metric(m, name, METRIC_COUNTER, labels);
    if (metric) {
        metric->value.counter += value;
    }

    pthread_mutex_unlock(&m->lock);
}

void metrics_gauge_set(metrics_t *m, const char *name, double value,
                       const char **labels) {
    pthread_mutex_lock(&m->lock);

    Metric *metric = find_or_create_metric(m, name, METRIC_GAUGE, labels);
    if (metric) {
        metric->value.gauge = value;
    }

    pthread_mutex_unlock(&m->lock);
}

void metrics_gauge_inc(metrics_t *m, const char *name, const char **labels) {
    pthread_mutex_lock(&m->lock);

    Metric *metric = find_or_create_metric(m, name, METRIC_GAUGE, labels);
    if (metric) {
        metric->value.gauge += 1.0;
    }

    pthread_mutex_unlock(&m->lock);
}

void metrics_gauge_dec(metrics_t *m, const char *name, const char **labels) {
    pthread_mutex_lock(&m->lock);

    Metric *metric = find_or_create_metric(m, name, METRIC_GAUGE, labels);
    if (metric) {
        metric->value.gauge -= 1.0;
    }

    pthread_mutex_unlock(&m->lock);
}

void metrics_histogram_observe(metrics_t *m, const char *name, double value,
                               const char **labels) {
    pthread_mutex_lock(&m->lock);

    Metric *metric = find_or_create_metric(m, name, METRIC_HISTOGRAM, labels);
    if (metric) {
        // Find bucket
        for (int i = 0; i < metric->value.histogram.num_buckets; i++) {
            if (value <= metric->value.histogram.bucket_bounds[i]) {
                metric->value.histogram.buckets[i]++;
            }
        }
        metric->value.histogram.sum += value;
        metric->value.histogram.count++;
    }

    pthread_mutex_unlock(&m->lock);
}

static int format_labels(char *buf, size_t size, Metric *metric) {
    if (metric->num_labels == 0) return 0;

    int len = snprintf(buf, size, "{");
    for (int i = 0; i < metric->num_labels; i++) {
        if (i > 0) len += snprintf(buf + len, size - len, ",");
        len += snprintf(buf + len, size - len, "%s=\"%s\"",
                       metric->labels[i].key, metric->labels[i].value);
    }
    len += snprintf(buf + len, size - len, "}");
    return len;
}

int metrics_format(metrics_t *m, char *buffer, size_t size) {
    pthread_mutex_lock(&m->lock);

    int offset = 0;

    for (int i = 0; i < MAX_METRICS; i++) {
        Metric *metric = m->metrics[i];
        while (metric) {
            char labels[512];
            format_labels(labels, sizeof(labels), metric);

            switch (metric->type) {
                case METRIC_COUNTER:
                    if (metric->help[0]) {
                        offset += snprintf(buffer + offset, size - offset,
                                          "# HELP %s %s\n", metric->name, metric->help);
                        offset += snprintf(buffer + offset, size - offset,
                                          "# TYPE %s counter\n", metric->name);
                    }
                    offset += snprintf(buffer + offset, size - offset,
                                      "%s%s %.0f\n", metric->name, labels,
                                      metric->value.counter);
                    break;

                case METRIC_GAUGE:
                    if (metric->help[0]) {
                        offset += snprintf(buffer + offset, size - offset,
                                          "# HELP %s %s\n", metric->name, metric->help);
                        offset += snprintf(buffer + offset, size - offset,
                                          "# TYPE %s gauge\n", metric->name);
                    }
                    offset += snprintf(buffer + offset, size - offset,
                                      "%s%s %.2f\n", metric->name, labels,
                                      metric->value.gauge);
                    break;

                case METRIC_HISTOGRAM:
                    if (metric->help[0]) {
                        offset += snprintf(buffer + offset, size - offset,
                                          "# HELP %s %s\n", metric->name, metric->help);
                        offset += snprintf(buffer + offset, size - offset,
                                          "# TYPE %s histogram\n", metric->name);
                    }
                    for (int j = 0; j < metric->value.histogram.num_buckets; j++) {
                        offset += snprintf(buffer + offset, size - offset,
                                          "%s_bucket{le=\"%.3f\"} %.0f\n",
                                          metric->name,
                                          metric->value.histogram.bucket_bounds[j],
                                          metric->value.histogram.buckets[j]);
                    }
                    offset += snprintf(buffer + offset, size - offset,
                                      "%s_bucket{le=\"+Inf\"} %lu\n",
                                      metric->name, metric->value.histogram.count);
                    offset += snprintf(buffer + offset, size - offset,
                                      "%s_sum %.6f\n", metric->name,
                                      metric->value.histogram.sum);
                    offset += snprintf(buffer + offset, size - offset,
                                      "%s_count %lu\n", metric->name,
                                      metric->value.histogram.count);
                    break;
            }

            metric = (Metric*)metric->next;
        }
    }

    pthread_mutex_unlock(&m->lock);
    return offset;
}

int metrics_expose(metrics_t *m, int fd) {
    char buffer[65536];
    int content_len = metrics_format(m, buffer, sizeof(buffer));

    char header[256];
    int header_len = snprintf(header, sizeof(header),
        "HTTP/1.1 200 OK\r\n"
        "Content-Type: text/plain; version=0.0.4\r\n"
        "Content-Length: %d\r\n"
        "\r\n", content_len);

    write(fd, header, header_len);
    write(fd, buffer, content_len);

    return header_len + content_len;
}

void metrics_register_lb_defaults(metrics_t *m) {
    metrics_register(m, "lb_requests_total", "Total requests", METRIC_COUNTER);
    metrics_register(m, "lb_requests_failed_total", "Failed requests", METRIC_COUNTER);
    metrics_register(m, "lb_connections_active", "Active connections", METRIC_GAUGE);
    metrics_register(m, "lb_backends_healthy", "Healthy backends", METRIC_GAUGE);
    metrics_register(m, "lb_request_duration_seconds", "Request latency", METRIC_HISTOGRAM);
    metrics_register(m, "lb_bytes_received_total", "Bytes received", METRIC_COUNTER);
    metrics_register(m, "lb_bytes_sent_total", "Bytes sent", METRIC_COUNTER);
    metrics_register(m, "lb_pool_hits_total", "Connection pool hits", METRIC_COUNTER);
    metrics_register(m, "lb_pool_misses_total", "Connection pool misses", METRIC_COUNTER);
}
