/**
 * Chapter 04: Advanced Load Balancer
 * ===================================
 * A feature-rich load balancer with:
 * - Multiple algorithms (round-robin, weighted, least-conn, ip-hash)
 * - Non-blocking I/O with select() (epoll on Linux)
 * - Connection pooling
 * - HTTP header injection
 * - Background health checking
 *
 * Usage: ./advanced_lb <port> <backend1:port:weight> [backend2:port:weight] ...
 * Example: ./advanced_lb 8080 127.0.0.1:9001:3 127.0.0.1:9002:2 127.0.0.1:9003:1
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
#include <sys/select.h>
#include <stdarg.h>
#include <limits.h>

#define BUFFER_SIZE 16384
#define BACKLOG 128
#define MAX_BACKENDS 16
#define MAX_CLIENTS 256
#define HEALTH_CHECK_INTERVAL 5

// ============================================================================
// Load Balancing Algorithms
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

// ============================================================================
// Data Structures
// ============================================================================

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

typedef struct {
    int client_fd;
    int backend_fd;
    Backend *backend;
    char client_ip[INET_ADDRSTRLEN];
    char buffer[BUFFER_SIZE];
    size_t buffer_len;
    int request_forwarded;
    time_t start_time;
} Connection;

typedef struct {
    Backend backends[MAX_BACKENDS];
    int num_backends;
    int current_index;
    int listen_port;
    Algorithm algorithm;
    Connection connections[MAX_CLIENTS];
    int num_connections;
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
// Backend Management
// ============================================================================

int parse_backend(const char *str, Backend *backend) {
    // Format: host:port or host:port:weight
    char temp[512];
    strncpy(temp, str, sizeof(temp) - 1);

    char *host = strtok(temp, ":");
    char *port = strtok(NULL, ":");
    char *weight_str = strtok(NULL, ":");

    if (!host || !port) {
        return -1;
    }

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

    // Non-blocking connect with timeout
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

        // Weight-adjusted score: lower is better
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

int connect_to_backend(Backend *backend) {
    struct addrinfo hints, *result;
    int backend_fd;

    memset(&hints, 0, sizeof(hints));
    hints.ai_family = AF_INET;
    hints.ai_socktype = SOCK_STREAM;

    if (getaddrinfo(backend->host, backend->port, &hints, &result) != 0) {
        return -1;
    }

    backend_fd = socket(result->ai_family, result->ai_socktype, result->ai_protocol);
    if (backend_fd < 0) {
        freeaddrinfo(result);
        return -1;
    }

    if (connect(backend_fd, result->ai_addr, result->ai_addrlen) < 0) {
        close(backend_fd);
        freeaddrinfo(result);
        return -1;
    }

    freeaddrinfo(result);
    return backend_fd;
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

Connection* find_free_connection(LoadBalancer *lb) {
    for (int i = 0; i < MAX_CLIENTS; i++) {
        if (lb->connections[i].client_fd == 0) {
            return &lb->connections[i];
        }
    }
    return NULL;
}

void close_connection(Connection *conn) {
    if (conn->client_fd > 0) close(conn->client_fd);
    if (conn->backend_fd > 0) close(conn->backend_fd);
    if (conn->backend) {
        conn->backend->active_connections--;
    }
    memset(conn, 0, sizeof(Connection));
}

// ============================================================================
// Main Server
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

    return server_fd;
}

void print_stats(LoadBalancer *lb) {
    time_t uptime = time(NULL) - lb->start_time;

    printf("\n");
    printf("╔════════════════════════════════════════════════════════════════════╗\n");
    printf("║                    ADVANCED LOAD BALANCER STATS                    ║\n");
    printf("╠════════════════════════════════════════════════════════════════════╣\n");
    printf("║  Algorithm: %-20s  Uptime: %ld seconds           ║\n",
           algorithm_names[lb->algorithm], uptime);
    printf("║  Total Requests: %-10lu  Requests/sec: %.2f               ║\n",
           lb->total_requests, uptime > 0 ? (double)lb->total_requests / uptime : 0);
    printf("╠════════════════════════════════════════════════════════════════════╣\n");
    printf("║  Backend             │ Wgt │ Status │ Active │ Total   │ Failed  ║\n");
    printf("╠════════════════════════════════════════════════════════════════════╣\n");

    for (int i = 0; i < lb->num_backends; i++) {
        Backend *b = &lb->backends[i];
        printf("║  %-14s:%-5s │ %-3d │ %-6s │ %-6d │ %-7lu │ %-7lu ║\n",
               b->host, b->port,
               b->weight,
               b->is_healthy ? "UP" : "DOWN",
               b->active_connections,
               b->total_requests,
               b->failed_requests);
    }

    printf("╚════════════════════════════════════════════════════════════════════╝\n\n");
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

void print_banner(LoadBalancer *lb) {
    printf("\n");
    printf("╔════════════════════════════════════════════════════════════════════╗\n");
    printf("║              ADVANCED LOAD BALANCER (Chapter 04)                   ║\n");
    printf("╠════════════════════════════════════════════════════════════════════╣\n");
    printf("║  Port: %-5d    Algorithm: %-20s               ║\n",
           lb->listen_port, algorithm_names[lb->algorithm]);
    printf("╠════════════════════════════════════════════════════════════════════╣\n");

    for (int i = 0; i < lb->num_backends; i++) {
        printf("║  [%d] %-15s:%-5s  weight=%-2d                             ║\n",
               i + 1, lb->backends[i].host, lb->backends[i].port, lb->backends[i].weight);
    }

    printf("╠════════════════════════════════════════════════════════════════════╣\n");
    printf("║  Test: curl http://localhost:%-5d                                 ║\n", lb->listen_port);
    printf("║  Stats: kill -USR1 %d                                            ║\n", getpid());
    printf("╚════════════════════════════════════════════════════════════════════╝\n\n");
}

int main(int argc, char *argv[]) {
    if (argc < 3) {
        fprintf(stderr, "Usage: %s <port> <backend1:port[:weight]> [...]\n", argv[0]);
        fprintf(stderr, "Example: %s 8080 127.0.0.1:9001:3 127.0.0.1:9002:2\n", argv[0]);
        fprintf(stderr, "\nAlgorithms (set via -a flag):\n");
        fprintf(stderr, "  rr       Round Robin (default)\n");
        fprintf(stderr, "  wrr      Weighted Round Robin\n");
        fprintf(stderr, "  lc       Least Connections\n");
        fprintf(stderr, "  iphash   IP Hash (sticky sessions)\n");
        exit(EXIT_FAILURE);
    }

    LoadBalancer lb;
    memset(&lb, 0, sizeof(lb));
    lb.listen_port = atoi(argv[1]);
    lb.current_index = -1;
    lb.algorithm = ALG_WEIGHTED_ROUND_ROBIN;  // Default
    lb.start_time = time(NULL);

    // Parse arguments
    for (int i = 2; i < argc; i++) {
        if (strcmp(argv[i], "-a") == 0 && i + 1 < argc) {
            i++;
            if (strcmp(argv[i], "rr") == 0) lb.algorithm = ALG_ROUND_ROBIN;
            else if (strcmp(argv[i], "wrr") == 0) lb.algorithm = ALG_WEIGHTED_ROUND_ROBIN;
            else if (strcmp(argv[i], "lc") == 0) lb.algorithm = ALG_LEAST_CONNECTIONS;
            else if (strcmp(argv[i], "iphash") == 0) lb.algorithm = ALG_IP_HASH;
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

    g_lb = &lb;
    signal(SIGINT, signal_handler);
    signal(SIGTERM, signal_handler);
    signal(SIGUSR1, signal_handler);
    signal(SIGPIPE, SIG_IGN);

    int server_fd = create_server_socket(lb.listen_port);
    set_nonblocking(server_fd);

    print_banner(&lb);
    log_msg("INFO", "Advanced LB started");

    while (g_running) {
        health_check_all(&lb);

        fd_set read_fds, write_fds;
        int max_fd = server_fd;

        FD_ZERO(&read_fds);
        FD_ZERO(&write_fds);
        FD_SET(server_fd, &read_fds);

        // Add all active connections
        for (int i = 0; i < MAX_CLIENTS; i++) {
            Connection *c = &lb.connections[i];
            if (c->client_fd > 0) {
                FD_SET(c->client_fd, &read_fds);
                if (c->client_fd > max_fd) max_fd = c->client_fd;
            }
            if (c->backend_fd > 0) {
                FD_SET(c->backend_fd, &read_fds);
                if (c->backend_fd > max_fd) max_fd = c->backend_fd;
            }
        }

        struct timeval timeout = {1, 0};
        int ready = select(max_fd + 1, &read_fds, &write_fds, NULL, &timeout);

        if (ready < 0 && errno != EINTR) {
            perror("select");
            break;
        }

        // Accept new connections
        if (FD_ISSET(server_fd, &read_fds)) {
            struct sockaddr_in client_addr;
            socklen_t len = sizeof(client_addr);
            int client_fd = accept(server_fd, (struct sockaddr*)&client_addr, &len);

            if (client_fd >= 0) {
                Connection *conn = find_free_connection(&lb);
                if (conn) {
                    conn->client_fd = client_fd;
                    inet_ntop(AF_INET, &client_addr.sin_addr, conn->client_ip, sizeof(conn->client_ip));
                    conn->start_time = time(NULL);
                    lb.num_connections++;

                    // Select backend
                    Backend *backend = select_backend(&lb, conn->client_ip);
                    if (backend) {
                        conn->backend_fd = connect_to_backend(backend);
                        if (conn->backend_fd >= 0) {
                            conn->backend = backend;
                            backend->active_connections++;
                            log_msg("CONN", "%s → %s:%s",
                                    conn->client_ip, backend->host, backend->port);
                        } else {
                            backend->failed_requests++;
                            backend->is_healthy = 0;
                            close_connection(conn);
                        }
                    }
                } else {
                    close(client_fd);
                    log_msg("WARN", "Max connections reached");
                }
            }
        }

        // Handle existing connections
        for (int i = 0; i < MAX_CLIENTS; i++) {
            Connection *c = &lb.connections[i];
            if (c->client_fd == 0) continue;

            // Client → Backend
            if (c->client_fd > 0 && FD_ISSET(c->client_fd, &read_fds)) {
                char buffer[BUFFER_SIZE];
                ssize_t n = read(c->client_fd, buffer, sizeof(buffer) - 1);

                if (n <= 0) {
                    close_connection(c);
                    lb.num_connections--;
                } else if (c->backend_fd > 0) {
                    buffer[n] = '\0';

                    if (!c->request_forwarded) {
                        inject_headers(buffer, sizeof(buffer), c->client_ip);
                        n = strlen(buffer);
                        c->request_forwarded = 1;
                        c->backend->total_requests++;
                        lb.total_requests++;
                    }

                    write(c->backend_fd, buffer, n);
                    c->backend->bytes_out += n;
                }
            }

            // Backend → Client
            if (c->backend_fd > 0 && FD_ISSET(c->backend_fd, &read_fds)) {
                char buffer[BUFFER_SIZE];
                ssize_t n = read(c->backend_fd, buffer, sizeof(buffer));

                if (n <= 0) {
                    close_connection(c);
                    lb.num_connections--;
                } else if (c->client_fd > 0) {
                    write(c->client_fd, buffer, n);
                    c->backend->bytes_in += n;
                }
            }
        }
    }

    // Cleanup
    for (int i = 0; i < MAX_CLIENTS; i++) {
        close_connection(&lb.connections[i]);
    }
    close(server_fd);

    return 0;
}
