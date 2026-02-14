/**
 * Simple HTTP Backend Server
 * ==========================
 * A minimal HTTP server that returns JSON with server identity.
 * Useful for testing load balancer distribution.
 *
 * Usage: ./simple_http_backend <port> [server_id]
 * Example: ./simple_http_backend 9001 backend-1
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <time.h>

#define BUFFER_SIZE 4096

int request_count = 0;

void handle_request(int client_fd, const char *server_id, int port) {
    char buffer[BUFFER_SIZE];
    char response[BUFFER_SIZE];
    char body[1024];

    // Read request (we don't parse it, just consume it)
    read(client_fd, buffer, sizeof(buffer));
    request_count++;

    // Get timestamp
    time_t now = time(NULL);
    char timestamp[64];
    strftime(timestamp, sizeof(timestamp), "%Y-%m-%dT%H:%M:%SZ", gmtime(&now));

    // Create JSON body
    int body_len = snprintf(body, sizeof(body),
        "{\n"
        "  \"server_id\": \"%s\",\n"
        "  \"port\": %d,\n"
        "  \"request_number\": %d,\n"
        "  \"timestamp\": \"%s\",\n"
        "  \"message\": \"Hello from %s!\"\n"
        "}\n",
        server_id, port, request_count, timestamp, server_id);

    // Create HTTP response
    int response_len = snprintf(response, sizeof(response),
        "HTTP/1.1 200 OK\r\n"
        "Content-Type: application/json\r\n"
        "Content-Length: %d\r\n"
        "Connection: close\r\n"
        "X-Server-ID: %s\r\n"
        "\r\n"
        "%s",
        body_len, server_id, body);

    write(client_fd, response, response_len);

    printf("[%s] Request #%d served\n", server_id, request_count);
}

int main(int argc, char *argv[]) {
    if (argc < 2) {
        fprintf(stderr, "Usage: %s <port> [server_id]\n", argv[0]);
        exit(EXIT_FAILURE);
    }

    int port = atoi(argv[1]);
    const char *server_id = argc > 2 ? argv[2] : "backend";

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

    if (listen(server_fd, 10) < 0) {
        perror("listen");
        exit(EXIT_FAILURE);
    }

    printf("Backend '%s' listening on port %d\n", server_id, port);

    while (1) {
        struct sockaddr_in client_addr;
        socklen_t client_len = sizeof(client_addr);

        int client_fd = accept(server_fd, (struct sockaddr*)&client_addr, &client_len);
        if (client_fd < 0) {
            perror("accept");
            continue;
        }

        handle_request(client_fd, server_id, port);
        close(client_fd);
    }

    close(server_fd);
    return 0;
}
