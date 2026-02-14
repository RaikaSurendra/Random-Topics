/**
 * Chapter 06: Connection-Pooled Load Balancer
 * ============================================
 *
 * Pingora-inspired connection pooling for maximum backend reuse.
 *
 * Key improvements over Chapter 05:
 * - Backend connections are pooled and reused
 * - Keep-alive management
 * - LRU eviction for idle connections
 * - 99%+ connection reuse achievable
 *
 * Usage: ./pooled_lb <port> <backend1:port[:weight]> [...] [-a algorithm] [-p pool_size]
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <netdb.h>
#include <errno.h>
#include <time.h>
#include <signal.h>
#include <fcntl.h>
#include <stdarg.h>
#include <limits.h>

#include "conn_pool.h"

// Use the event loop from Chapter 05
#include "../chapter-05-high-perf-io/event_loop.h"

// ============================================================================
// Configuration
// ============================================================================

#define BUFFER_SIZE 16384
#define BACKLOG 128
#define MAX_BACKENDS 16
#define MAX_CLIENTS 4096
#define HEALTH_CHECK_INTERVAL 5
#define DEFAULT_POOL_SIZE 64
#define DEFAULT_POOL_TTL 60

// ============================================================================
// Data Structures
// ============================================================================

typedef enum {
    ALG_ROUND_ROBIN,
    ALG_WEIGHTED_ROUND_ROBIN,
    ALG_LEAST_CONNECTIONS,
    ALG_IP_HASH
} Algorithm;

const char* algorithm_names[] = {
    "Round Robin",
    "Weighted Round Robin",
    "Least Connections",
    "IP Hash"
};

typedef struct {
    char host[256];
    char port[6];
    int weight;
    int current_weight;
    int is_healthy;
    int active_connections;
    unsigned long total_requests;
    unsigned long failed_requests;
    unsigned long bytes_in;
    unsigned long bytes_out;
    time_t last_health_check;
} Backend;

typedef struct Connection {
    int client_fd;
    int backend_fd;
    Backend *backend;
    char client_ip[INET_ADDRSTRLEN];
    char buffer[BUFFER_SIZE];
    size_t buffer_len;
    int request_forwarded;
    int keep_alive;  // Track if client wants keep-alive
    time_t start_time;
    struct Connection *next;
} Connection;

typedef struct {
    Backend backends[MAX_BACKENDS];
    int num_backends;
    int current_index;
    int listen_port;
    Algorithm algorithm;

    // Event loop
    event_loop_t *event_loop;
    int server_fd;

    // Connection management (client connections)
    Connection *connections;
    Connection *free_list;
    int num_connections;
    int max_connections;

    // Backend connection pool (NEW!)
    conn_pool_t *backend_pool;

    // Statistics
    unsigned long total_requests;
    unsigned long total_bytes;
    time_t start_time;
} LoadBalancer;

LoadBalancer *g_lb = NULL;
volatile int g_running = 1;

// ============================================================================
// Logging
// ============================================================================

void log_msg(const char *level, const char *format, ...) {
    time_t now = time(NULL);
    struct tm *tm_info = localtime(&now);
    char timestamp[32];
    strftime(timestamp, sizeof(timestamp), "%H:%M:%S", tm_info);

    printf("[%s] [%-5s] ", timestamp, level);

    va_list args;
    va_start(args, format);
    vprintf(format, args);
    va_end(args);

    printf("\n");
    fflush(stdout);
}

// ============================================================================
// Client Connection Pool (unchanged from Ch 05)
// ============================================================================

int init_connection_pool(LoadBalancer *lb, int max_connections) {
    lb->connections = calloc(max_connections, sizeof(Connection));
    if (!lb->connections) return -1;

    lb->max_connections = max_connections;
    lb->num_connections = 0;

    lb->free_list = &lb->connections[0];
    for (int i = 0; i < max_connections - 1; i++) {
        lb->connections[i].client_fd = -1;
        lb->connections[i].backend_fd = -1;
        lb->connections[i].next = &lb->connections[i + 1];
    }
    lb->connections[max_connections - 1].client_fd = -1;
    lb->connections[max_connections - 1].backend_fd = -1;
    lb->connections[max_connections - 1].next = NULL;

    return 0;
}

Connection* alloc_connection(LoadBalancer *lb) {
    if (!lb->free_list) return NULL;

    Connection *conn = lb->free_list;
    lb->free_list = conn->next;
    conn->next = NULL;
    lb->num_connections++;

    return conn;
}

void free_connection(LoadBalancer *lb, Connection *conn) {
    if (conn->client_fd >= 0) {
        event_loop_del(lb->event_loop, conn->client_fd);
        close(conn->client_fd);
    }

    // Return backend connection to pool instead of closing!
    if (conn->backend_fd >= 0) {
        event_loop_del(lb->event_loop, conn->backend_fd);

        if (conn->backend && conn->keep_alive) {
            // Return to pool for reuse
            conn_pool_return(lb->backend_pool, conn->backend_fd,
                           conn->backend->host, conn->backend->port);
        } else {
            // Close if not keep-alive or errored
            conn_pool_close(lb->backend_pool, conn->backend_fd);
        }
    }

    if (conn->backend) {
        conn->backend->active_connections--;
    }

    memset(conn, 0, sizeof(Connection));
    conn->client_fd = -1;
    conn->backend_fd = -1;
    conn->next = lb->free_list;
    lb->free_list = conn;
    lb->num_connections--;
}

// ============================================================================
// Backend Management
// ============================================================================

int parse_backend(const char *str, Backend *backend) {
    char temp[512];
    strncpy(temp, str, sizeof(temp) - 1);

    char *host = strtok(temp, ":");
    char *port = strtok(NULL, ":");
    char *weight_str = strtok(NULL, ":");

    if (!host || !port) return -1;

    strncpy(backend->host, host, sizeof(backend->host) - 1);
    strncpy(backend->port, port, sizeof(backend->port) - 1);
    backend->weight = weight_str ? atoi(weight_str) : 1;
    if (backend->weight < 1) backend->weight = 1;

    backend->current_weight = 0;
    backend->is_healthy = 1;
    backend->active_connections = 0;
    backend->total_requests = 0;
    backend->failed_requests = 0;
    backend->bytes_in = 0;
    backend->bytes_out = 0;
    backend->last_health_check = 0;

    return 0;
}

int check_backend_health(Backend *backend) {
    struct addrinfo hints, *result;
    int sock_fd;

    memset(&hints, 0, sizeof(hints));
    hints.ai_family = AF_INET;
    hints.ai_socktype = SOCK_STREAM;

    if (getaddrinfo(backend->host, backend->port, &hints, &result) != 0) {
        return 0;
    }

    sock_fd = socket(result->ai_family, result->ai_socktype, result->ai_protocol);
    if (sock_fd < 0) {
        freeaddrinfo(result);
        return 0;
    }

    fcntl(sock_fd, F_SETFL, O_NONBLOCK);

    int connected = 0;
    if (connect(sock_fd, result->ai_addr, result->ai_addrlen) == 0) {
        connected = 1;
    } else if (errno == EINPROGRESS) {
        fd_set write_fds;
        FD_ZERO(&write_fds);
        FD_SET(sock_fd, &write_fds);

        struct timeval timeout = {2, 0};
        if (select(sock_fd + 1, NULL, &write_fds, NULL, &timeout) > 0) {
            int error = 0;
            socklen_t len = sizeof(error);
            getsockopt(sock_fd, SOL_SOCKET, SO_ERROR, &error, &len);
            connected = (error == 0);
        }
    }

    close(sock_fd);
    freeaddrinfo(result);
    return connected;
}

void health_check_all(LoadBalancer *lb) {
    time_t now = time(NULL);

    for (int i = 0; i < lb->num_backends; i++) {
        Backend *b = &lb->backends[i];

        if (now - b->last_health_check < HEALTH_CHECK_INTERVAL) {
            continue;
        }

        b->last_health_check = now;
        int was_healthy = b->is_healthy;
        b->is_healthy = check_backend_health(b);

        if (was_healthy && !b->is_healthy) {
            log_msg("WARN", "Backend %s:%s marked DOWN", b->host, b->port);
        } else if (!was_healthy && b->is_healthy) {
            log_msg("INFO", "Backend %s:%s marked UP", b->host, b->port);
        }
    }

    // Also cleanup idle pool connections
    conn_pool_cleanup(lb->backend_pool);
}

// ============================================================================
// Load Balancing Algorithms
// ============================================================================

Backend* select_round_robin(LoadBalancer *lb) {
    int start = lb->current_index;
    int attempts = 0;

    do {
        lb->current_index = (lb->current_index + 1) % lb->num_backends;
        if (lb->backends[lb->current_index].is_healthy) {
            return &lb->backends[lb->current_index];
        }
        attempts++;
    } while (attempts < lb->num_backends);

    return &lb->backends[(start + 1) % lb->num_backends];
}

Backend* select_weighted_round_robin(LoadBalancer *lb) {
    int total_weight = 0;
    Backend *best = NULL;

    for (int i = 0; i < lb->num_backends; i++) {
        Backend *b = &lb->backends[i];
        if (!b->is_healthy) continue;

        b->current_weight += b->weight;
        total_weight += b->weight;

        if (best == NULL || b->current_weight > best->current_weight) {
            best = b;
        }
    }

    if (best) {
        best->current_weight -= total_weight;
    }

    return best ? best : select_round_robin(lb);
}

Backend* select_least_connections(LoadBalancer *lb) {
    Backend *best = NULL;
    int min_score = INT_MAX;

    for (int i = 0; i < lb->num_backends; i++) {
        Backend *b = &lb->backends[i];
        if (!b->is_healthy) continue;

        int score = (b->active_connections * 100) / b->weight;
        if (score < min_score) {
            min_score = score;
            best = b;
        }
    }

    return best ? best : select_round_robin(lb);
}

Backend* select_ip_hash(LoadBalancer *lb, const char *client_ip) {
    unsigned int hash = 0;
    for (const char *p = client_ip; *p; p++) {
        hash = hash * 31 + *p;
    }

    int start = hash % lb->num_backends;
    int index = start;

    do {
        if (lb->backends[index].is_healthy) {
            return &lb->backends[index];
        }
        index = (index + 1) % lb->num_backends;
    } while (index != start);

    return &lb->backends[start];
}

Backend* select_backend(LoadBalancer *lb, const char *client_ip) {
    switch (lb->algorithm) {
        case ALG_WEIGHTED_ROUND_ROBIN:
            return select_weighted_round_robin(lb);
        case ALG_LEAST_CONNECTIONS:
            return select_least_connections(lb);
        case ALG_IP_HASH:
            return select_ip_hash(lb, client_ip);
        case ALG_ROUND_ROBIN:
        default:
            return select_round_robin(lb);
    }
}

// ============================================================================
// Connection Handling
// ============================================================================

int set_nonblocking(int fd) {
    int flags = fcntl(fd, F_GETFL, 0);
    return fcntl(fd, F_SETFL, flags | O_NONBLOCK);
}

// Get backend connection from pool (key difference!)
int get_backend_connection(LoadBalancer *lb, Backend *backend) {
    return conn_pool_get(lb->backend_pool, backend->host, backend->port);
}

void inject_headers(char *request, size_t max_size, const char *client_ip) {
    char *header_pos = strstr(request, "\r\n");
    if (!header_pos) return;
    header_pos += 2;

    char headers[256];
    int len = snprintf(headers, sizeof(headers),
                       "X-Forwarded-For: %s\r\n"
                       "X-Real-IP: %s\r\n",
                       client_ip, client_ip);

    size_t remaining = strlen(header_pos);
    size_t current_len = header_pos - request;

    if (current_len + len + remaining < max_size) {
        memmove(header_pos + len, header_pos, remaining + 1);
        memcpy(header_pos, headers, len);
    }
}

// Check for Connection: keep-alive header
int check_keep_alive(const char *request) {
    // HTTP/1.1 defaults to keep-alive
    if (strstr(request, "HTTP/1.1")) {
        if (strstr(request, "Connection: close")) {
            return 0;
        }
        return 1;
    }
    // HTTP/1.0 defaults to close
    if (strstr(request, "Connection: keep-alive") ||
        strstr(request, "Connection: Keep-Alive")) {
        return 1;
    }
    return 0;
}

// ============================================================================
// Event Callbacks
// ============================================================================

void on_client_event(int fd, int events, void *user_data);
void on_backend_event(int fd, int events, void *user_data);

void on_client_event(int fd, int events, void *user_data) {
    Connection *conn = (Connection*)user_data;
    LoadBalancer *lb = g_lb;

    if (events & (EVENT_ERROR | EVENT_HUP)) {
        conn->keep_alive = 0;  // Don't reuse backend on error
        free_connection(lb, conn);
        return;
    }

    if (events & EVENT_READ) {
        char buffer[BUFFER_SIZE];
        ssize_t n = read(fd, buffer, sizeof(buffer) - 1);

        if (n <= 0) {
            conn->keep_alive = 0;
            free_connection(lb, conn);
            return;
        }

        buffer[n] = '\0';

        // Check keep-alive on first request
        if (!conn->request_forwarded) {
            conn->keep_alive = check_keep_alive(buffer);
            inject_headers(buffer, sizeof(buffer), conn->client_ip);
            n = strlen(buffer);
            conn->request_forwarded = 1;
            conn->backend->total_requests++;
            lb->total_requests++;
        }

        if (conn->backend_fd >= 0) {
            write(conn->backend_fd, buffer, n);
            conn->backend->bytes_out += n;
        }
    }
}

void on_backend_event(int fd, int events, void *user_data) {
    Connection *conn = (Connection*)user_data;
    LoadBalancer *lb = g_lb;

    if (events & (EVENT_ERROR | EVENT_HUP)) {
        conn->keep_alive = 0;  // Don't reuse on error
        free_connection(lb, conn);
        return;
    }

    if (events & EVENT_READ) {
        char buffer[BUFFER_SIZE];
        ssize_t n = read(fd, buffer, sizeof(buffer));

        if (n <= 0) {
            free_connection(lb, conn);
            return;
        }

        if (conn->client_fd >= 0) {
            write(conn->client_fd, buffer, n);
            conn->backend->bytes_in += n;
        }
    }
}

void on_server_event(int fd, int events, void *user_data) {
    LoadBalancer *lb = (LoadBalancer*)user_data;
    (void)events;

    struct sockaddr_in client_addr;
    socklen_t len = sizeof(client_addr);
    int client_fd = accept(fd, (struct sockaddr*)&client_addr, &len);

    if (client_fd < 0) return;

    Connection *conn = alloc_connection(lb);
    if (!conn) {
        close(client_fd);
        log_msg("WARN", "Max connections reached (%d)", lb->max_connections);
        return;
    }

    conn->client_fd = client_fd;
    inet_ntop(AF_INET, &client_addr.sin_addr, conn->client_ip, sizeof(conn->client_ip));
    conn->start_time = time(NULL);
    conn->keep_alive = 1;  // Assume keep-alive until proven otherwise

    Backend *backend = select_backend(lb, conn->client_ip);
    if (!backend) {
        free_connection(lb, conn);
        return;
    }

    // Get connection from pool (key difference!)
    conn->backend_fd = get_backend_connection(lb, backend);
    if (conn->backend_fd < 0) {
        backend->failed_requests++;
        backend->is_healthy = 0;
        free_connection(lb, conn);
        return;
    }

    conn->backend = backend;
    backend->active_connections++;
    set_nonblocking(client_fd);
    set_nonblocking(conn->backend_fd);

    event_loop_add(lb->event_loop, client_fd, EVENT_READ, on_client_event, conn);
    event_loop_add(lb->event_loop, conn->backend_fd, EVENT_READ, on_backend_event, conn);

    log_msg("CONN", "%s -> %s:%s (pooled)", conn->client_ip, backend->host, backend->port);
}

// ============================================================================
// Server Setup
// ============================================================================

int create_server_socket(int port) {
    int server_fd = socket(AF_INET, SOCK_STREAM, 0);
    if (server_fd < 0) {
        perror("socket");
        exit(EXIT_FAILURE);
    }

    int opt = 1;
    setsockopt(server_fd, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt));

    struct sockaddr_in addr;
    memset(&addr, 0, sizeof(addr));
    addr.sin_family = AF_INET;
    addr.sin_addr.s_addr = INADDR_ANY;
    addr.sin_port = htons(port);

    if (bind(server_fd, (struct sockaddr*)&addr, sizeof(addr)) < 0) {
        perror("bind");
        exit(EXIT_FAILURE);
    }

    if (listen(server_fd, BACKLOG) < 0) {
        perror("listen");
        exit(EXIT_FAILURE);
    }

    set_nonblocking(server_fd);
    return server_fd;
}

// ============================================================================
// Statistics & Signals
// ============================================================================

void print_stats(LoadBalancer *lb) {
    time_t uptime = time(NULL) - lb->start_time;

    conn_pool_stats_t pool_stats;
    conn_pool_get_stats(lb->backend_pool, &pool_stats);

    printf("\n");
    printf("====================================================================\n");
    printf("  CONNECTION-POOLED LOAD BALANCER STATS (Chapter 06)\n");
    printf("====================================================================\n");
    printf("  Algorithm: %-20s  Uptime: %ld seconds\n",
           algorithm_names[lb->algorithm], uptime);
    printf("  Total Requests: %-10lu  Requests/sec: %.2f\n",
           lb->total_requests, uptime > 0 ? (double)lb->total_requests / uptime : 0);
    printf("  Active Connections: %d / %d\n", lb->num_connections, lb->max_connections);
    printf("--------------------------------------------------------------------\n");
    printf("  CONNECTION POOL STATS (Pingora-style):\n");
    printf("    Pool Size: %d / %d\n", pool_stats.current_size, pool_stats.max_size);
    printf("    Pool Hits: %lu  Misses: %lu  Evictions: %lu\n",
           pool_stats.hits, pool_stats.misses, pool_stats.evictions);
    printf("    HIT RATE: %.2f%% (target: 99%%+)\n", pool_stats.hit_rate);
    printf("--------------------------------------------------------------------\n");
    printf("  Backend             | Wgt | Status | Active | Total   | Failed\n");
    printf("--------------------------------------------------------------------\n");

    for (int i = 0; i < lb->num_backends; i++) {
        Backend *b = &lb->backends[i];
        printf("  %-14s:%-5s | %-3d | %-6s | %-6d | %-7lu | %-7lu\n",
               b->host, b->port,
               b->weight,
               b->is_healthy ? "UP" : "DOWN",
               b->active_connections,
               b->total_requests,
               b->failed_requests);
    }

    printf("====================================================================\n\n");
}

void signal_handler(int sig) {
    if (sig == SIGINT || sig == SIGTERM) {
        printf("\n");
        log_msg("INFO", "Shutting down...");
        if (g_lb) print_stats(g_lb);
        g_running = 0;
    } else if (sig == SIGUSR1 && g_lb) {
        print_stats(g_lb);
    }
}

void print_banner(LoadBalancer *lb, int pool_size) {
    printf("\n");
    printf("====================================================================\n");
    printf("  CONNECTION-POOLED LOAD BALANCER (Chapter 06)\n");
    printf("====================================================================\n");
    printf("  Port: %-5d    Algorithm: %-20s\n",
           lb->listen_port, algorithm_names[lb->algorithm]);
    printf("  Backend Pool Size: %d  TTL: %d seconds\n", pool_size, DEFAULT_POOL_TTL);
    printf("--------------------------------------------------------------------\n");

    for (int i = 0; i < lb->num_backends; i++) {
        printf("  [%d] %-15s:%-5s  weight=%d\n",
               i + 1, lb->backends[i].host, lb->backends[i].port,
               lb->backends[i].weight);
    }

    printf("--------------------------------------------------------------------\n");
    printf("  Test: curl http://localhost:%d\n", lb->listen_port);
    printf("  Stats: kill -USR1 %d\n", getpid());
    printf("====================================================================\n\n");
}

// ============================================================================
// Main
// ============================================================================

int main(int argc, char *argv[]) {
    if (argc < 3) {
        fprintf(stderr, "Usage: %s <port> <backend1:port[:weight]> [...] [-a alg] [-p pool]\n", argv[0]);
        fprintf(stderr, "Example: %s 8080 127.0.0.1:9001:3 127.0.0.1:9002:2 -a wrr -p 64\n", argv[0]);
        fprintf(stderr, "\nAlgorithms: rr, wrr, lc, iphash\n");
        fprintf(stderr, "Pool size: Number of backend connections to pool (default: %d)\n",
                DEFAULT_POOL_SIZE);
        exit(EXIT_FAILURE);
    }

    LoadBalancer lb;
    memset(&lb, 0, sizeof(lb));
    lb.listen_port = atoi(argv[1]);
    lb.current_index = -1;
    lb.algorithm = ALG_WEIGHTED_ROUND_ROBIN;
    lb.start_time = time(NULL);

    int pool_size = DEFAULT_POOL_SIZE;

    // Parse arguments
    for (int i = 2; i < argc; i++) {
        if (strcmp(argv[i], "-a") == 0 && i + 1 < argc) {
            i++;
            if (strcmp(argv[i], "rr") == 0) lb.algorithm = ALG_ROUND_ROBIN;
            else if (strcmp(argv[i], "wrr") == 0) lb.algorithm = ALG_WEIGHTED_ROUND_ROBIN;
            else if (strcmp(argv[i], "lc") == 0) lb.algorithm = ALG_LEAST_CONNECTIONS;
            else if (strcmp(argv[i], "iphash") == 0) lb.algorithm = ALG_IP_HASH;
        } else if (strcmp(argv[i], "-p") == 0 && i + 1 < argc) {
            i++;
            pool_size = atoi(argv[i]);
        } else if (lb.num_backends < MAX_BACKENDS) {
            if (parse_backend(argv[i], &lb.backends[lb.num_backends]) == 0) {
                lb.num_backends++;
            }
        }
    }

    if (lb.num_backends == 0) {
        fprintf(stderr, "No valid backends\n");
        exit(EXIT_FAILURE);
    }

    // Initialize client connection pool
    if (init_connection_pool(&lb, MAX_CLIENTS) < 0) {
        fprintf(stderr, "Failed to initialize connection pool\n");
        exit(EXIT_FAILURE);
    }

    // Initialize backend connection pool (NEW!)
    lb.backend_pool = conn_pool_create(pool_size, DEFAULT_POOL_TTL);
    if (!lb.backend_pool) {
        fprintf(stderr, "Failed to create backend pool\n");
        exit(EXIT_FAILURE);
    }

    // Create event loop
    lb.event_loop = event_loop_create(MAX_CLIENTS);
    if (!lb.event_loop) {
        fprintf(stderr, "Failed to create event loop\n");
        exit(EXIT_FAILURE);
    }

    g_lb = &lb;
    signal(SIGINT, signal_handler);
    signal(SIGTERM, signal_handler);
    signal(SIGUSR1, signal_handler);
    signal(SIGPIPE, SIG_IGN);

    lb.server_fd = create_server_socket(lb.listen_port);
    event_loop_add(lb.event_loop, lb.server_fd, EVENT_READ, on_server_event, &lb);

    print_banner(&lb, pool_size);
    log_msg("INFO", "Connection-pooled LB started");

    while (g_running) {
        health_check_all(&lb);
        event_loop_run(lb.event_loop, 1000);
    }

    // Cleanup
    event_loop_destroy(lb.event_loop);
    conn_pool_destroy(lb.backend_pool);
    free(lb.connections);
    close(lb.server_fd);

    return 0;
}
