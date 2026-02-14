/*
 * Chapter 01: getaddrinfo() Demo
 *
 * Modern, portable address resolution using getaddrinfo()
 * instead of manually filling sockaddr_in structures.
 *
 * Benefits:
 * - Works with both IPv4 and IPv6
 * - Handles DNS resolution
 * - Protocol-agnostic
 * - Thread-safe
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <netdb.h>
#include <arpa/inet.h>

void print_usage(const char *prog) {
    fprintf(stderr, "Usage: %s <hostname> <port>\n", prog);
    fprintf(stderr, "\nExamples:\n");
    fprintf(stderr, "  %s localhost 8080\n", prog);
    fprintf(stderr, "  %s google.com 80\n", prog);
    fprintf(stderr, "  %s 127.0.0.1 9000\n", prog);
}

void print_addrinfo(struct addrinfo *ai) {
    char ipstr[INET6_ADDRSTRLEN];
    void *addr;
    const char *ipver;

    if (ai->ai_family == AF_INET) {
        struct sockaddr_in *ipv4 = (struct sockaddr_in *)ai->ai_addr;
        addr = &(ipv4->sin_addr);
        ipver = "IPv4";
    } else {
        struct sockaddr_in6 *ipv6 = (struct sockaddr_in6 *)ai->ai_addr;
        addr = &(ipv6->sin6_addr);
        ipver = "IPv6";
    }

    inet_ntop(ai->ai_family, addr, ipstr, sizeof(ipstr));
    printf("  %s: %s\n", ipver, ipstr);

    printf("    Family: %d (%s)\n", ai->ai_family,
           ai->ai_family == AF_INET ? "AF_INET" :
           ai->ai_family == AF_INET6 ? "AF_INET6" : "other");

    printf("    Socket Type: %d (%s)\n", ai->ai_socktype,
           ai->ai_socktype == SOCK_STREAM ? "SOCK_STREAM" :
           ai->ai_socktype == SOCK_DGRAM ? "SOCK_DGRAM" : "other");

    printf("    Protocol: %d (%s)\n", ai->ai_protocol,
           ai->ai_protocol == IPPROTO_TCP ? "TCP" :
           ai->ai_protocol == IPPROTO_UDP ? "UDP" : "other");
}

int main(int argc, char *argv[]) {
    if (argc != 3) {
        print_usage(argv[0]);
        return 1;
    }

    const char *hostname = argv[1];
    const char *port = argv[2];

    printf("=== getaddrinfo() Demo ===\n\n");
    printf("Resolving: %s:%s\n\n", hostname, port);

    // Setup hints - what we're looking for
    struct addrinfo hints;
    memset(&hints, 0, sizeof(hints));
    hints.ai_family = AF_UNSPEC;      // IPv4 or IPv6
    hints.ai_socktype = SOCK_STREAM;  // TCP stream sockets

    // Perform resolution
    struct addrinfo *result;
    int status = getaddrinfo(hostname, port, &hints, &result);

    if (status != 0) {
        fprintf(stderr, "getaddrinfo error: %s\n", gai_strerror(status));
        return 1;
    }

    // Print all results
    printf("Resolved addresses:\n");
    int count = 0;
    for (struct addrinfo *p = result; p != NULL; p = p->ai_next) {
        count++;
        printf("\n[%d]\n", count);
        print_addrinfo(p);
    }

    printf("\nTotal: %d address(es) found\n", count);

    // Demo: Connect using first result
    printf("\n--- Attempting connection using first result ---\n");

    struct addrinfo *p = result;
    int sockfd = socket(p->ai_family, p->ai_socktype, p->ai_protocol);
    if (sockfd < 0) {
        perror("socket");
        freeaddrinfo(result);
        return 1;
    }

    printf("Created socket (fd=%d)\n", sockfd);

    // Set connection timeout
    struct timeval timeout = {3, 0};  // 3 seconds
    setsockopt(sockfd, SOL_SOCKET, SO_SNDTIMEO, &timeout, sizeof(timeout));

    printf("Connecting to %s:%s...\n", hostname, port);

    if (connect(sockfd, p->ai_addr, p->ai_addrlen) < 0) {
        perror("connect");
        printf("(This is expected if no server is listening)\n");
    } else {
        printf("Connected successfully!\n");

        // Get peer info
        char peer_ip[INET6_ADDRSTRLEN];
        if (p->ai_family == AF_INET) {
            struct sockaddr_in *s = (struct sockaddr_in *)p->ai_addr;
            inet_ntop(AF_INET, &s->sin_addr, peer_ip, sizeof(peer_ip));
            printf("Connected to: %s:%d\n", peer_ip, ntohs(s->sin_port));
        } else {
            struct sockaddr_in6 *s = (struct sockaddr_in6 *)p->ai_addr;
            inet_ntop(AF_INET6, &s->sin6_addr, peer_ip, sizeof(peer_ip));
            printf("Connected to: [%s]:%d\n", peer_ip, ntohs(s->sin6_port));
        }
    }

    close(sockfd);
    freeaddrinfo(result);

    printf("\n=== Key Takeaways ===\n");
    printf("1. getaddrinfo() handles both IPv4 and IPv6\n");
    printf("2. It performs DNS resolution automatically\n");
    printf("3. Returns a linked list - try each until one works\n");
    printf("4. Always call freeaddrinfo() when done\n");
    printf("5. Use gai_strerror() for error messages\n");

    return 0;
}
