/**
 * Chapter 01: TCP Echo Server
 * ============================
 * A simple server that echoes back whatever the client sends.
 * This is the foundation for understanding proxies and load balancers.
 *
 * Usage: ./echo_server <port>
 * Test:  echo "Hello" | nc localhost <port>
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <errno.h>

#define BUFFER_SIZE 4096
#define BACKLOG 10

/**
 * Print error message and exit
 */
void die(const char *msg) {
    perror(msg);
    exit(EXIT_FAILURE);
}

/**
 * Handle a single client connection
 * This is where we'll later add proxy logic
 */
void handle_client(int client_fd, struct sockaddr_in *client_addr) {
    char buffer[BUFFER_SIZE];
    ssize_t bytes_read;

    // Log connection
    char client_ip[INET_ADDRSTRLEN];
    inet_ntop(AF_INET, &client_addr->sin_addr, client_ip, sizeof(client_ip));
    printf("[INFO] Client connected: %s:%d\n", client_ip, ntohs(client_addr->sin_port));

    // Echo loop - read and write back
    while ((bytes_read = read(client_fd, buffer, sizeof(buffer))) > 0) {
        // Log received data (truncate for display)
        buffer[bytes_read] = '\0';
        printf("[RECV] %zd bytes: %.*s", bytes_read,
               (int)(bytes_read > 50 ? 50 : bytes_read), buffer);
        if (bytes_read > 50) printf("...");
        if (buffer[bytes_read-1] != '\n') printf("\n");

        // Echo back to client
        ssize_t bytes_written = write(client_fd, buffer, bytes_read);
        if (bytes_written < 0) {
            perror("[ERROR] write failed");
            break;
        }
        printf("[SEND] %zd bytes echoed\n", bytes_written);
    }

    if (bytes_read < 0) {
        perror("[ERROR] read failed");
    }

    printf("[INFO] Client disconnected: %s:%d\n", client_ip, ntohs(client_addr->sin_port));
    close(client_fd);
}

int main(int argc, char *argv[]) {
    if (argc != 2) {
        fprintf(stderr, "Usage: %s <port>\n", argv[0]);
        exit(EXIT_FAILURE);
    }

    int port = atoi(argv[1]);

    // =========================================
    // STEP 1: Create socket
    // =========================================
    // AF_INET     = IPv4
    // SOCK_STREAM = TCP (reliable, ordered)
    // 0           = Let OS choose protocol
    int server_fd = socket(AF_INET, SOCK_STREAM, 0);
    if (server_fd < 0) {
        die("socket() failed");
    }
    printf("[INIT] Socket created (fd=%d)\n", server_fd);

    // =========================================
    // STEP 2: Set socket options
    // =========================================
    // SO_REUSEADDR: Allow immediate reuse of port after restart
    // Without this, you get "Address already in use" for ~60 seconds
    int opt = 1;
    if (setsockopt(server_fd, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt)) < 0) {
        die("setsockopt(SO_REUSEADDR) failed");
    }
    printf("[INIT] Socket options set (SO_REUSEADDR)\n");

    // =========================================
    // STEP 3: Bind to address
    // =========================================
    struct sockaddr_in server_addr;
    memset(&server_addr, 0, sizeof(server_addr));
    server_addr.sin_family = AF_INET;           // IPv4
    server_addr.sin_addr.s_addr = INADDR_ANY;   // Bind to all interfaces (0.0.0.0)
    server_addr.sin_port = htons(port);         // Port (host to network byte order)

    if (bind(server_fd, (struct sockaddr*)&server_addr, sizeof(server_addr)) < 0) {
        die("bind() failed");
    }
    printf("[INIT] Bound to 0.0.0.0:%d\n", port);

    // =========================================
    // STEP 4: Listen for connections
    // =========================================
    // BACKLOG: Maximum queue length for pending connections
    if (listen(server_fd, BACKLOG) < 0) {
        die("listen() failed");
    }
    printf("[INIT] Listening with backlog=%d\n", BACKLOG);

    printf("\n========================================\n");
    printf("  Echo Server running on port %d\n", port);
    printf("  Test: echo \"Hello\" | nc localhost %d\n", port);
    printf("========================================\n\n");

    // =========================================
    // STEP 5: Accept loop
    // =========================================
    while (1) {
        struct sockaddr_in client_addr;
        socklen_t client_len = sizeof(client_addr);

        // accept() blocks until a client connects
        // Returns a NEW socket fd for this specific client
        // Original server_fd continues listening
        int client_fd = accept(server_fd, (struct sockaddr*)&client_addr, &client_len);
        if (client_fd < 0) {
            perror("[ERROR] accept() failed");
            continue;
        }

        // Handle this client (blocks - one client at a time)
        // In a real proxy, we'd fork() or use threads/epoll
        handle_client(client_fd, &client_addr);
    }

    // Cleanup (never reached in this example)
    close(server_fd);
    return 0;
}
