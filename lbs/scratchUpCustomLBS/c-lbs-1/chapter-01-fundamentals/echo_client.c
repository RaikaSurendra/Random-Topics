/**
 * Chapter 01: TCP Echo Client
 * ============================
 * A simple client to test our echo server.
 *
 * Usage: ./echo_client <host> <port>
 * Example: ./echo_client 127.0.0.1 8080
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <netdb.h>

#define BUFFER_SIZE 4096

void die(const char *msg) {
    perror(msg);
    exit(EXIT_FAILURE);
}

int main(int argc, char *argv[]) {
    if (argc != 3) {
        fprintf(stderr, "Usage: %s <host> <port>\n", argv[0]);
        exit(EXIT_FAILURE);
    }

    const char *host = argv[1];
    int port = atoi(argv[2]);

    // =========================================
    // STEP 1: Create socket
    // =========================================
    int sock_fd = socket(AF_INET, SOCK_STREAM, 0);
    if (sock_fd < 0) {
        die("socket() failed");
    }

    // =========================================
    // STEP 2: Resolve hostname (modern way)
    // =========================================
    // getaddrinfo() is preferred over gethostbyname()
    // - Thread-safe
    // - Supports IPv4 and IPv6
    // - Returns linked list of addresses to try
    struct addrinfo hints, *result;
    memset(&hints, 0, sizeof(hints));
    hints.ai_family = AF_INET;        // IPv4
    hints.ai_socktype = SOCK_STREAM;  // TCP

    char port_str[6];
    snprintf(port_str, sizeof(port_str), "%d", port);

    int status = getaddrinfo(host, port_str, &hints, &result);
    if (status != 0) {
        fprintf(stderr, "getaddrinfo: %s\n", gai_strerror(status));
        exit(EXIT_FAILURE);
    }

    // =========================================
    // STEP 3: Connect to server
    // =========================================
    printf("[INFO] Connecting to %s:%d...\n", host, port);

    if (connect(sock_fd, result->ai_addr, result->ai_addrlen) < 0) {
        die("connect() failed");
    }
    printf("[INFO] Connected!\n\n");

    freeaddrinfo(result);  // Done with address info

    // =========================================
    // STEP 4: Interactive echo loop
    // =========================================
    char buffer[BUFFER_SIZE];

    printf("Type messages to send (Ctrl+D to quit):\n");
    printf("─────────────────────────────────────────\n");

    while (1) {
        printf("> ");
        fflush(stdout);

        // Read from stdin
        if (fgets(buffer, sizeof(buffer), stdin) == NULL) {
            printf("\n[INFO] EOF - closing connection\n");
            break;
        }

        size_t len = strlen(buffer);

        // Send to server
        ssize_t sent = write(sock_fd, buffer, len);
        if (sent < 0) {
            die("write() failed");
        }
        printf("[SENT] %zd bytes\n", sent);

        // Read response
        ssize_t received = read(sock_fd, buffer, sizeof(buffer) - 1);
        if (received < 0) {
            die("read() failed");
        }
        if (received == 0) {
            printf("[INFO] Server closed connection\n");
            break;
        }

        buffer[received] = '\0';
        printf("[RECV] %zd bytes: %s", received, buffer);
        if (buffer[received-1] != '\n') printf("\n");
    }

    // =========================================
    // STEP 5: Cleanup
    // =========================================
    close(sock_fd);
    printf("[INFO] Connection closed\n");

    return 0;
}
