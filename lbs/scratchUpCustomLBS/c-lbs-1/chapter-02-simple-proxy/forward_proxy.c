/**
 * Chapter 02: Simple Forward Proxy (HTTP CONNECT Tunnel)
 * =======================================================
 * A forward proxy that clients configure in their browser/system.
 * Supports HTTP CONNECT method for tunneling (HTTPS pass-through).
 *
 * DIFFERENCE FROM REVERSE PROXY:
 * - Forward Proxy: Client KNOWS about proxy, configures it explicitly
 * - Reverse Proxy: Client doesn't know, thinks proxy IS the server
 *
 * Usage: ./forward_proxy <listen_port>
 * Example: ./forward_proxy 8888
 *
 * Configure browser: HTTP Proxy = localhost:8888
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

/**
 * Log with timestamp
 */
void log_msg(const char *level, const char *format, ...) {
    time_t now = time(NULL);
    struct tm *tm_info = localtime(&now);
    char timestamp[32];
    strftime(timestamp, sizeof(timestamp), "%H:%M:%S", tm_info);

    printf("[%s] [%s] ", timestamp, level);

    va_list args;
    va_start(args, format);
    vprintf(format, args);
    va_end(args);

    printf("\n");
}

/**
 * Parse CONNECT request to extract host and port
 * Format: CONNECT host:port HTTP/1.1
 */
int parse_connect_request(const char *request, char *host, size_t host_size, int *port) {
    // Example: "CONNECT example.com:443 HTTP/1.1\r\n..."
    if (strncmp(request, "CONNECT ", 8) != 0) {
        return -1;  // Not a CONNECT request
    }

    const char *start = request + 8;
    const char *space = strchr(start, ' ');
    if (!space) return -1;

    const char *colon = strchr(start, ':');
    if (!colon || colon > space) {
        // No port specified, use default
        size_t len = space - start;
        if (len >= host_size) len = host_size - 1;
        strncpy(host, start, len);
        host[len] = '\0';
        *port = 80;
    } else {
        // Extract host
        size_t len = colon - start;
        if (len >= host_size) len = host_size - 1;
        strncpy(host, start, len);
        host[len] = '\0';

        // Extract port
        *port = atoi(colon + 1);
    }

    return 0;
}

/**
 * Parse regular HTTP request for host header
 * Used for non-CONNECT requests (plain HTTP)
 */
int parse_http_request(const char *request, char *host, size_t host_size,
                       int *port, char *path, size_t path_size) {
    // Example: "GET http://example.com/path HTTP/1.1\r\n..."
    // Or: "GET /path HTTP/1.1\r\nHost: example.com\r\n..."

    *port = 80;

    // Check for absolute URL
    if (strncmp(request + 4, "http://", 7) == 0) {
        const char *url_start = request + 11;  // After "GET http://"
        const char *path_start = strchr(url_start, '/');
        const char *host_end = path_start ? path_start : strchr(url_start, ' ');

        // Check for port in URL
        const char *colon = strchr(url_start, ':');
        if (colon && colon < host_end) {
            size_t len = colon - url_start;
            if (len >= host_size) len = host_size - 1;
            strncpy(host, url_start, len);
            host[len] = '\0';
            *port = atoi(colon + 1);
        } else {
            size_t len = host_end - url_start;
            if (len >= host_size) len = host_size - 1;
            strncpy(host, url_start, len);
            host[len] = '\0';
        }

        if (path_start) {
            const char *path_end = strchr(path_start, ' ');
            size_t len = path_end ? (size_t)(path_end - path_start) : strlen(path_start);
            if (len >= path_size) len = path_size - 1;
            strncpy(path, path_start, len);
            path[len] = '\0';
        } else {
            strcpy(path, "/");
        }

        return 0;
    }

    // Look for Host header
    const char *host_header = strstr(request, "Host: ");
    if (!host_header) {
        host_header = strstr(request, "host: ");
    }

    if (host_header) {
        host_header += 6;  // Skip "Host: "
        const char *end = strstr(host_header, "\r\n");
        if (end) {
            const char *colon = strchr(host_header, ':');
            if (colon && colon < end) {
                size_t len = colon - host_header;
                if (len >= host_size) len = host_size - 1;
                strncpy(host, host_header, len);
                host[len] = '\0';
                *port = atoi(colon + 1);
            } else {
                size_t len = end - host_header;
                if (len >= host_size) len = host_size - 1;
                strncpy(host, host_header, len);
                host[len] = '\0';
            }
            return 0;
        }
    }

    return -1;
}

/**
 * Connect to target server
 */
int connect_to_target(const char *host, int port) {
    struct addrinfo hints, *result, *rp;
    int sock_fd;
    char port_str[6];

    snprintf(port_str, sizeof(port_str), "%d", port);

    memset(&hints, 0, sizeof(hints));
    hints.ai_family = AF_INET;
    hints.ai_socktype = SOCK_STREAM;

    int status = getaddrinfo(host, port_str, &hints, &result);
    if (status != 0) {
        log_msg("ERROR", "getaddrinfo(%s): %s", host, gai_strerror(status));
        return -1;
    }

    for (rp = result; rp != NULL; rp = rp->ai_next) {
        sock_fd = socket(rp->ai_family, rp->ai_socktype, rp->ai_protocol);
        if (sock_fd < 0) continue;

        if (connect(sock_fd, rp->ai_addr, rp->ai_addrlen) == 0) {
            break;
        }

        close(sock_fd);
    }

    freeaddrinfo(result);

    if (rp == NULL) {
        log_msg("ERROR", "Could not connect to %s:%d", host, port);
        return -1;
    }

    return sock_fd;
}

/**
 * Handle CONNECT tunnel (for HTTPS)
 */
void handle_connect_tunnel(int client_fd, const char *host, int port) {
    log_msg("TUNNEL", "CONNECT %s:%d", host, port);

    int target_fd = connect_to_target(host, port);
    if (target_fd < 0) {
        const char *error = "HTTP/1.1 502 Bad Gateway\r\n\r\n";
        write(client_fd, error, strlen(error));
        return;
    }

    // Send 200 Connection Established
    const char *success = "HTTP/1.1 200 Connection Established\r\n\r\n";
    write(client_fd, success, strlen(success));

    log_msg("TUNNEL", "Established to %s:%d", host, port);

    // Bidirectional relay using select()
    fd_set read_fds;
    int max_fd = (client_fd > target_fd) ? client_fd : target_fd;
    char buffer[BUFFER_SIZE];

    while (1) {
        FD_ZERO(&read_fds);
        FD_SET(client_fd, &read_fds);
        FD_SET(target_fd, &read_fds);

        struct timeval timeout;
        timeout.tv_sec = 60;
        timeout.tv_usec = 0;

        int ready = select(max_fd + 1, &read_fds, NULL, NULL, &timeout);
        if (ready <= 0) break;

        // Client -> Target
        if (FD_ISSET(client_fd, &read_fds)) {
            ssize_t n = read(client_fd, buffer, sizeof(buffer));
            if (n <= 0) break;
            if (write(target_fd, buffer, n) != n) break;
        }

        // Target -> Client
        if (FD_ISSET(target_fd, &read_fds)) {
            ssize_t n = read(target_fd, buffer, sizeof(buffer));
            if (n <= 0) break;
            if (write(client_fd, buffer, n) != n) break;
        }
    }

    close(target_fd);
    log_msg("TUNNEL", "Closed %s:%d", host, port);
}

/**
 * Handle regular HTTP request
 */
void handle_http_request(int client_fd, const char *request, size_t request_len,
                         const char *host, int port) {
    log_msg("HTTP", "Request to %s:%d", host, port);

    int target_fd = connect_to_target(host, port);
    if (target_fd < 0) {
        const char *error = "HTTP/1.1 502 Bad Gateway\r\n\r\n";
        write(client_fd, error, strlen(error));
        return;
    }

    // Forward request
    write(target_fd, request, request_len);

    // Forward response
    char buffer[BUFFER_SIZE];
    ssize_t n;
    while ((n = read(target_fd, buffer, sizeof(buffer))) > 0) {
        write(client_fd, buffer, n);
    }

    close(target_fd);
}

/**
 * Handle client connection
 */
void handle_client(int client_fd, struct sockaddr_in *client_addr) {
    char client_ip[INET_ADDRSTRLEN];
    inet_ntop(AF_INET, &client_addr->sin_addr, client_ip, sizeof(client_ip));

    char buffer[BUFFER_SIZE];
    ssize_t n = read(client_fd, buffer, sizeof(buffer) - 1);
    if (n <= 0) {
        close(client_fd);
        return;
    }
    buffer[n] = '\0';

    char host[256];
    int port;
    char path[1024];

    // Check if it's a CONNECT request (for HTTPS tunneling)
    if (strncmp(buffer, "CONNECT ", 8) == 0) {
        if (parse_connect_request(buffer, host, sizeof(host), &port) == 0) {
            handle_connect_tunnel(client_fd, host, port);
        }
    } else {
        // Regular HTTP request
        if (parse_http_request(buffer, host, sizeof(host), &port, path, sizeof(path)) == 0) {
            handle_http_request(client_fd, buffer, n, host, port);
        } else {
            const char *error = "HTTP/1.1 400 Bad Request\r\n\r\n";
            write(client_fd, error, strlen(error));
        }
    }

    close(client_fd);
}

int main(int argc, char *argv[]) {
    if (argc != 2) {
        fprintf(stderr, "Usage: %s <listen_port>\n", argv[0]);
        fprintf(stderr, "Example: %s 8888\n", argv[0]);
        fprintf(stderr, "\nConfigure your browser's HTTP proxy to localhost:<port>\n");
        exit(EXIT_FAILURE);
    }

    int port = atoi(argv[1]);

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

    printf("\n");
    printf("╔═══════════════════════════════════════════════════════╗\n");
    printf("║           FORWARD PROXY (Chapter 02)                  ║\n");
    printf("╠═══════════════════════════════════════════════════════╣\n");
    printf("║  Listening on: 0.0.0.0:%-5d                          ║\n", port);
    printf("║                                                       ║\n");
    printf("║  Configure browser/system proxy to localhost:%-5d   ║\n", port);
    printf("║  Supports: HTTP and HTTPS (via CONNECT tunnel)        ║\n");
    printf("╚═══════════════════════════════════════════════════════╝\n");
    printf("\n");

    log_msg("INFO", "Forward proxy started on port %d", port);

    while (1) {
        struct sockaddr_in client_addr;
        socklen_t client_len = sizeof(client_addr);

        int client_fd = accept(server_fd, (struct sockaddr*)&client_addr, &client_len);
        if (client_fd < 0) {
            perror("accept");
            continue;
        }

        handle_client(client_fd, &client_addr);
    }

    close(server_fd);
    return 0;
}
