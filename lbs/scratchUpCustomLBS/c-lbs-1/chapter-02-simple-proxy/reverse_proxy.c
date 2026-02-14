/**
 * Chapter 02: Simple Reverse Proxy
 * =================================
 * A single-threaded reverse proxy that forwards requests to a backend server.
 *
 * This is the foundation for a load balancer - we just need to add:
 * - Multiple backends
 * - Backend selection algorithm
 *
 * Usage: ./reverse_proxy <listen_port> <backend_host> <backend_port>
 * Example: ./reverse_proxy 8080 127.0.0.1 9000
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
#include <stdarg.h>

#define BUFFER_SIZE 8192
#define BACKLOG 10

// Configuration
typedef struct {
    int listen_port;
    char backend_host[256];
    char backend_port[6];
} ProxyConfig;

/**
 * Get current timestamp for logging
 */
void get_timestamp(char *buffer, size_t size) {
    time_t now = time(NULL);
    struct tm *tm_info = localtime(&now);
    strftime(buffer, size, "%Y-%m-%d %H:%M:%S", tm_info);
}

/**
 * Log with timestamp
 */
void log_info(const char *format, ...) {
    char timestamp[32];
    get_timestamp(timestamp, sizeof(timestamp));

    printf("[%s] ", timestamp);

    va_list args;
    va_start(args, format);
    vprintf(format, args);
    va_end(args);

    printf("\n");
}

/**
 * Print error and exit
 */
void die(const char *msg) {
    perror(msg);
    exit(EXIT_FAILURE);
}

/**
 * Connect to backend server
 * Returns socket fd on success, -1 on failure
 */
int connect_to_backend(const char *host, const char *port) {
    struct addrinfo hints, *result, *rp;
    int backend_fd;

    memset(&hints, 0, sizeof(hints));
    hints.ai_family = AF_INET;        // IPv4
    hints.ai_socktype = SOCK_STREAM;  // TCP

    int status = getaddrinfo(host, port, &hints, &result);
    if (status != 0) {
        fprintf(stderr, "[ERROR] getaddrinfo: %s\n", gai_strerror(status));
        return -1;
    }

    // Try each address until we connect
    for (rp = result; rp != NULL; rp = rp->ai_next) {
        backend_fd = socket(rp->ai_family, rp->ai_socktype, rp->ai_protocol);
        if (backend_fd < 0) continue;

        if (connect(backend_fd, rp->ai_addr, rp->ai_addrlen) == 0) {
            break;  // Success
        }

        close(backend_fd);
    }

    freeaddrinfo(result);

    if (rp == NULL) {
        fprintf(stderr, "[ERROR] Could not connect to backend %s:%s\n", host, port);
        return -1;
    }

    return backend_fd;
}

/**
 * Relay data between client and backend
 *
 * This is a simple implementation that:
 * 1. Reads all data from client
 * 2. Sends to backend
 * 3. Reads all response from backend
 * 4. Sends back to client
 *
 * For production, you'd use select/poll/epoll for bidirectional streaming.
 */
void relay_data(int client_fd, int backend_fd, const char *client_ip) {
    char buffer[BUFFER_SIZE];
    ssize_t bytes_read, bytes_written;
    size_t total_request = 0, total_response = 0;

    // =========================================
    // Phase 1: Forward client request to backend
    // =========================================

    // Read from client
    bytes_read = read(client_fd, buffer, sizeof(buffer) - 1);
    if (bytes_read <= 0) {
        if (bytes_read < 0) perror("[ERROR] read from client");
        return;
    }
    total_request = bytes_read;

    // Log request (first line only)
    buffer[bytes_read] = '\0';
    char *first_line_end = strstr(buffer, "\r\n");
    if (first_line_end) {
        *first_line_end = '\0';
        log_info("REQUEST from %s: %s", client_ip, buffer);
        *first_line_end = '\r';  // Restore
    }

    // Forward to backend
    bytes_written = write(backend_fd, buffer, bytes_read);
    if (bytes_written < 0) {
        perror("[ERROR] write to backend");
        return;
    }

    // =========================================
    // Phase 2: Forward backend response to client
    // =========================================

    // Read response from backend and forward to client
    while ((bytes_read = read(backend_fd, buffer, sizeof(buffer))) > 0) {
        total_response += bytes_read;

        bytes_written = write(client_fd, buffer, bytes_read);
        if (bytes_written < 0) {
            perror("[ERROR] write to client");
            break;
        }
    }

    log_info("COMPLETE: %zu bytes request, %zu bytes response", total_request, total_response);
}

/**
 * Handle a single client connection
 */
void handle_client(int client_fd, struct sockaddr_in *client_addr, ProxyConfig *config) {
    char client_ip[INET_ADDRSTRLEN];
    inet_ntop(AF_INET, &client_addr->sin_addr, client_ip, sizeof(client_ip));

    log_info("CONNECT from %s:%d", client_ip, ntohs(client_addr->sin_port));

    // Connect to backend
    int backend_fd = connect_to_backend(config->backend_host, config->backend_port);
    if (backend_fd < 0) {
        // Send 502 Bad Gateway response
        const char *error_response =
            "HTTP/1.1 502 Bad Gateway\r\n"
            "Content-Type: text/plain\r\n"
            "Content-Length: 23\r\n"
            "Connection: close\r\n"
            "\r\n"
            "502 - Backend Unavailable";

        write(client_fd, error_response, strlen(error_response));
        close(client_fd);
        return;
    }

    log_info("BACKEND connected to %s:%s (fd=%d)",
             config->backend_host, config->backend_port, backend_fd);

    // Relay data between client and backend
    relay_data(client_fd, backend_fd, client_ip);

    // Cleanup
    close(backend_fd);
    close(client_fd);

    log_info("DISCONNECT from %s", client_ip);
}

/**
 * Create and configure the listening socket
 */
int create_server_socket(int port) {
    int server_fd = socket(AF_INET, SOCK_STREAM, 0);
    if (server_fd < 0) {
        die("socket() failed");
    }

    // Allow port reuse
    int opt = 1;
    if (setsockopt(server_fd, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt)) < 0) {
        die("setsockopt(SO_REUSEADDR) failed");
    }

    // Bind to port
    struct sockaddr_in server_addr;
    memset(&server_addr, 0, sizeof(server_addr));
    server_addr.sin_family = AF_INET;
    server_addr.sin_addr.s_addr = INADDR_ANY;
    server_addr.sin_port = htons(port);

    if (bind(server_fd, (struct sockaddr*)&server_addr, sizeof(server_addr)) < 0) {
        die("bind() failed");
    }

    if (listen(server_fd, BACKLOG) < 0) {
        die("listen() failed");
    }

    return server_fd;
}

void print_banner(ProxyConfig *config) {
    printf("\n");
    printf("╔═══════════════════════════════════════════════════════╗\n");
    printf("║          SIMPLE REVERSE PROXY (Chapter 02)            ║\n");
    printf("╠═══════════════════════════════════════════════════════╣\n");
    printf("║  Listening on    : 0.0.0.0:%-5d                      ║\n", config->listen_port);
    printf("║  Backend server  : %s:%-5s                     ║\n",
           config->backend_host, config->backend_port);
    printf("╠═══════════════════════════════════════════════════════╣\n");
    printf("║  Test with: curl http://localhost:%d/                ║\n", config->listen_port);
    printf("╚═══════════════════════════════════════════════════════╝\n");
    printf("\n");
}

int main(int argc, char *argv[]) {
    if (argc != 4) {
        fprintf(stderr, "Usage: %s <listen_port> <backend_host> <backend_port>\n", argv[0]);
        fprintf(stderr, "Example: %s 8080 127.0.0.1 9000\n", argv[0]);
        exit(EXIT_FAILURE);
    }

    // Parse configuration
    ProxyConfig config;
    config.listen_port = atoi(argv[1]);
    strncpy(config.backend_host, argv[2], sizeof(config.backend_host) - 1);
    strncpy(config.backend_port, argv[3], sizeof(config.backend_port) - 1);

    // Create server socket
    int server_fd = create_server_socket(config.listen_port);

    print_banner(&config);
    log_info("Reverse proxy started");

    // Main accept loop
    while (1) {
        struct sockaddr_in client_addr;
        socklen_t client_len = sizeof(client_addr);

        int client_fd = accept(server_fd, (struct sockaddr*)&client_addr, &client_len);
        if (client_fd < 0) {
            perror("[ERROR] accept() failed");
            continue;
        }

        // Handle client (blocking - one at a time)
        // In Chapter 03, we'll handle multiple clients
        handle_client(client_fd, &client_addr, &config);
    }

    close(server_fd);
    return 0;
}
