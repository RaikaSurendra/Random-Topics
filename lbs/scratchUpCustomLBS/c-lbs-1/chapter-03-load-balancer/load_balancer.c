/**
 * Chapter 03: Round-Robin Load Balancer
 * ======================================
 * A load balancer that distributes requests across multiple backend servers.
 *
 * Features:
 * - Round-robin algorithm
 * - Basic health checking (on connection failure)
 * - Statistics tracking
 * - Graceful failover
 *
 * Usage: ./load_balancer <listen_port> [backend1:port] [backend2:port] ...
 * Example: ./load_balancer 8080 127.0.0.1:9001 127.0.0.1:9002 127.0.0.1:9003
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
#include <stdarg.h>

#define BUFFER_SIZE 8192
#define BACKLOG 10
#define MAX_BACKENDS 10
#define HEALTH_CHECK_INTERVAL 10  // seconds

// ============================================================================
// Data Structures
// ============================================================================

typedef struct {
    char host[256];
    char port[6];
    int is_healthy;
    int active_connections;
    unsigned long total_requests;
    unsigned long failed_requests;
    time_t last_health_check;
    time_t last_failure;
} Backend;

typedef struct {
    Backend backends[MAX_BACKENDS];
    int num_backends;
    int current_index;  // For round-robin
    int listen_port;
    unsigned long total_requests;
    time_t start_time;
} LoadBalancer;

// Global for signal handler
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

/**
 * Parse backend string "host:port"
 */
int parse_backend(const char *str, Backend *backend) {
    char *colon = strchr(str, ':');
    if (!colon) {
        fprintf(stderr, "Invalid backend format: %s (expected host:port)\n", str);
        return -1;
    }

    size_t host_len = colon - str;
    if (host_len >= sizeof(backend->host)) {
        host_len = sizeof(backend->host) - 1;
    }

    strncpy(backend->host, str, host_len);
    backend->host[host_len] = '\0';
    strncpy(backend->port, colon + 1, sizeof(backend->port) - 1);

    backend->is_healthy = 1;  // Assume healthy initially
    backend->active_connections = 0;
    backend->total_requests = 0;
    backend->failed_requests = 0;
    backend->last_health_check = 0;
    backend->last_failure = 0;

    return 0;
}

/**
 * Check if backend is healthy by attempting connection
 */
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

    // Set short timeout for health check
    struct timeval timeout = {2, 0};  // 2 seconds
    setsockopt(sock_fd, SOL_SOCKET, SO_SNDTIMEO, &timeout, sizeof(timeout));
    setsockopt(sock_fd, SOL_SOCKET, SO_RCVTIMEO, &timeout, sizeof(timeout));

    int connected = (connect(sock_fd, result->ai_addr, result->ai_addrlen) == 0);

    close(sock_fd);
    freeaddrinfo(result);

    return connected;
}

/**
 * Periodic health check for all backends
 */
void health_check_all(LoadBalancer *lb) {
    time_t now = time(NULL);

    for (int i = 0; i < lb->num_backends; i++) {
        Backend *b = &lb->backends[i];

        // Only check if enough time has passed
        if (now - b->last_health_check < HEALTH_CHECK_INTERVAL) {
            continue;
        }

        b->last_health_check = now;
        int was_healthy = b->is_healthy;
        b->is_healthy = check_backend_health(b);

        if (was_healthy && !b->is_healthy) {
            log_msg("WARN", "Backend %s:%s is DOWN", b->host, b->port);
        } else if (!was_healthy && b->is_healthy) {
            log_msg("INFO", "Backend %s:%s is UP", b->host, b->port);
        }
    }
}

/**
 * Select next backend using round-robin
 */
Backend* select_backend(LoadBalancer *lb) {
    if (lb->num_backends == 0) return NULL;

    int start = lb->current_index;
    int attempts = 0;

    do {
        lb->current_index = (lb->current_index + 1) % lb->num_backends;
        Backend *b = &lb->backends[lb->current_index];

        if (b->is_healthy) {
            return b;
        }

        attempts++;
    } while (attempts < lb->num_backends);

    // All backends unhealthy - try the next one anyway
    lb->current_index = (start + 1) % lb->num_backends;
    return &lb->backends[lb->current_index];
}

/**
 * Connect to a backend server
 */
int connect_to_backend(Backend *backend) {
    struct addrinfo hints, *result, *rp;
    int backend_fd;

    memset(&hints, 0, sizeof(hints));
    hints.ai_family = AF_INET;
    hints.ai_socktype = SOCK_STREAM;

    if (getaddrinfo(backend->host, backend->port, &hints, &result) != 0) {
        return -1;
    }

    for (rp = result; rp != NULL; rp = rp->ai_next) {
        backend_fd = socket(rp->ai_family, rp->ai_socktype, rp->ai_protocol);
        if (backend_fd < 0) continue;

        if (connect(backend_fd, rp->ai_addr, rp->ai_addrlen) == 0) {
            break;
        }

        close(backend_fd);
    }

    freeaddrinfo(result);

    if (rp == NULL) {
        return -1;
    }

    return backend_fd;
}

// ============================================================================
// Request Handling
// ============================================================================

/**
 * Relay data between client and backend
 */
void relay_data(int client_fd, int backend_fd, Backend *backend, const char *client_ip) {
    char buffer[BUFFER_SIZE];
    ssize_t bytes_read, bytes_written;
    size_t total_request = 0, total_response = 0;

    // Forward client request to backend
    bytes_read = read(client_fd, buffer, sizeof(buffer) - 1);
    if (bytes_read <= 0) {
        return;
    }
    total_request = bytes_read;

    // Log request summary
    buffer[bytes_read] = '\0';
    char *first_line_end = strstr(buffer, "\r\n");
    if (first_line_end) {
        *first_line_end = '\0';
        log_msg("REQ", "[%s:%s] %s → %s",
                backend->host, backend->port, client_ip, buffer);
        *first_line_end = '\r';
    }

    bytes_written = write(backend_fd, buffer, bytes_read);
    if (bytes_written < 0) {
        log_msg("ERROR", "Write to backend failed");
        return;
    }

    // Forward backend response to client
    while ((bytes_read = read(backend_fd, buffer, sizeof(buffer))) > 0) {
        total_response += bytes_read;
        bytes_written = write(client_fd, buffer, bytes_read);
        if (bytes_written < 0) {
            break;
        }
    }

    log_msg("RESP", "[%s:%s] %zu bytes request, %zu bytes response",
            backend->host, backend->port, total_request, total_response);
}

/**
 * Handle a single client connection
 */
void handle_client(int client_fd, struct sockaddr_in *client_addr, LoadBalancer *lb) {
    char client_ip[INET_ADDRSTRLEN];
    inet_ntop(AF_INET, &client_addr->sin_addr, client_ip, sizeof(client_ip));

    // Select a backend
    Backend *backend = select_backend(lb);
    if (!backend) {
        log_msg("ERROR", "No backends available");
        const char *error = "HTTP/1.1 503 Service Unavailable\r\n\r\nNo backends available";
        write(client_fd, error, strlen(error));
        close(client_fd);
        return;
    }

    // Connect to backend
    int backend_fd = connect_to_backend(backend);
    if (backend_fd < 0) {
        // Mark backend as unhealthy
        backend->is_healthy = 0;
        backend->last_failure = time(NULL);
        backend->failed_requests++;

        log_msg("ERROR", "Backend %s:%s connection failed", backend->host, backend->port);

        // Try another backend
        backend = select_backend(lb);
        if (backend) {
            backend_fd = connect_to_backend(backend);
        }

        if (backend_fd < 0) {
            const char *error = "HTTP/1.1 502 Bad Gateway\r\n\r\nBackend unavailable";
            write(client_fd, error, strlen(error));
            close(client_fd);
            return;
        }
    }

    // Track connection
    backend->active_connections++;
    lb->total_requests++;

    // Relay data
    relay_data(client_fd, backend_fd, backend, client_ip);

    // Cleanup
    backend->active_connections--;
    backend->total_requests++;
    close(backend_fd);
    close(client_fd);
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

    return server_fd;
}

void print_stats(LoadBalancer *lb) {
    printf("\n");
    printf("╔═══════════════════════════════════════════════════════════════╗\n");
    printf("║                   LOAD BALANCER STATISTICS                    ║\n");
    printf("╠═══════════════════════════════════════════════════════════════╣\n");
    printf("║  Total Requests: %-10lu  Uptime: %ld seconds             ║\n",
           lb->total_requests, time(NULL) - lb->start_time);
    printf("╠═══════════════════════════════════════════════════════════════╣\n");
    printf("║  Backend              │ Status │ Active │ Total  │ Failed    ║\n");
    printf("╠═══════════════════════════════════════════════════════════════╣\n");

    for (int i = 0; i < lb->num_backends; i++) {
        Backend *b = &lb->backends[i];
        printf("║  %-15s:%-5s │ %-6s │ %-6d │ %-6lu │ %-6lu    ║\n",
               b->host, b->port,
               b->is_healthy ? "UP" : "DOWN",
               b->active_connections,
               b->total_requests,
               b->failed_requests);
    }

    printf("╚═══════════════════════════════════════════════════════════════╝\n");
    printf("\n");
}

void signal_handler(int sig) {
    if (sig == SIGINT || sig == SIGTERM) {
        printf("\n");
        log_msg("INFO", "Shutting down...");
        if (g_lb) {
            print_stats(g_lb);
        }
        g_running = 0;
    } else if (sig == SIGUSR1 && g_lb) {
        print_stats(g_lb);
    }
}

void print_banner(LoadBalancer *lb) {
    printf("\n");
    printf("╔═══════════════════════════════════════════════════════════════╗\n");
    printf("║            ROUND-ROBIN LOAD BALANCER (Chapter 03)             ║\n");
    printf("╠═══════════════════════════════════════════════════════════════╣\n");
    printf("║  Listening on: 0.0.0.0:%-5d                                  ║\n", lb->listen_port);
    printf("║  Backends: %-3d                                                ║\n", lb->num_backends);
    printf("╠═══════════════════════════════════════════════════════════════╣\n");

    for (int i = 0; i < lb->num_backends; i++) {
        printf("║    [%d] %s:%-5s                                         ║\n",
               i + 1, lb->backends[i].host, lb->backends[i].port);
    }

    printf("╠═══════════════════════════════════════════════════════════════╣\n");
    printf("║  Test: curl http://localhost:%-5d/                           ║\n", lb->listen_port);
    printf("║  Stats: kill -USR1 %d                                       ║\n", getpid());
    printf("╚═══════════════════════════════════════════════════════════════╝\n");
    printf("\n");
}

int main(int argc, char *argv[]) {
    if (argc < 3) {
        fprintf(stderr, "Usage: %s <listen_port> <backend1:port> [backend2:port] ...\n", argv[0]);
        fprintf(stderr, "Example: %s 8080 127.0.0.1:9001 127.0.0.1:9002 127.0.0.1:9003\n", argv[0]);
        exit(EXIT_FAILURE);
    }

    // Initialize load balancer
    LoadBalancer lb;
    memset(&lb, 0, sizeof(lb));
    lb.listen_port = atoi(argv[1]);
    lb.current_index = -1;  // Will wrap to 0 on first select
    lb.start_time = time(NULL);

    // Parse backends
    for (int i = 2; i < argc && lb.num_backends < MAX_BACKENDS; i++) {
        if (parse_backend(argv[i], &lb.backends[lb.num_backends]) == 0) {
            lb.num_backends++;
        }
    }

    if (lb.num_backends == 0) {
        fprintf(stderr, "No valid backends specified\n");
        exit(EXIT_FAILURE);
    }

    // Setup signal handlers
    g_lb = &lb;
    signal(SIGINT, signal_handler);
    signal(SIGTERM, signal_handler);
    signal(SIGUSR1, signal_handler);
    signal(SIGPIPE, SIG_IGN);  // Ignore broken pipe

    // Create server socket
    int server_fd = create_server_socket(lb.listen_port);

    print_banner(&lb);
    log_msg("INFO", "Load balancer started with %d backends", lb.num_backends);

    // Main loop
    while (g_running) {
        // Periodic health check
        health_check_all(&lb);

        struct sockaddr_in client_addr;
        socklen_t client_len = sizeof(client_addr);

        // Use select with timeout for graceful shutdown
        fd_set read_fds;
        FD_ZERO(&read_fds);
        FD_SET(server_fd, &read_fds);

        struct timeval timeout = {1, 0};  // 1 second timeout
        int ready = select(server_fd + 1, &read_fds, NULL, NULL, &timeout);

        if (ready < 0 && errno != EINTR) {
            perror("select");
            break;
        }

        if (ready > 0 && FD_ISSET(server_fd, &read_fds)) {
            int client_fd = accept(server_fd, (struct sockaddr*)&client_addr, &client_len);
            if (client_fd < 0) {
                if (errno != EINTR) {
                    perror("accept");
                }
                continue;
            }

            handle_client(client_fd, &client_addr, &lb);
        }
    }

    close(server_fd);
    return 0;
}
