/**
 * Chapter 09: Zero-Copy I/O
 * =========================
 *
 * High-performance data transfer without user-space copies:
 * - sendfile() for file-to-socket (Linux/macOS)
 * - splice() for socket-to-socket (Linux only)
 * - Fallback read/write for other platforms
 *
 * Performance impact:
 * - 50-70% CPU reduction for large transfers
 * - 2-3x throughput improvement at high bandwidth
 *
 * Usage:
 *   // File to socket (e.g., static files)
 *   zero_copy_file_to_socket(socket_fd, file_fd, 0, file_size);
 *
 *   // Socket to socket (proxy relay)
 *   zero_copy_relay(client_fd, backend_fd, buffer_size);
 */

#ifndef ZERO_COPY_H
#define ZERO_COPY_H

#include <sys/types.h>
#include <stddef.h>

// Platform detection
#if defined(__linux__)
    #define HAVE_SENDFILE 1
    #define HAVE_SPLICE 1
#elif defined(__APPLE__) || defined(__FreeBSD__)
    #define HAVE_SENDFILE 1
    #define HAVE_SPLICE 0
#else
    #define HAVE_SENDFILE 0
    #define HAVE_SPLICE 0
#endif

// Minimum size for zero-copy (below this, regular I/O is faster)
#define ZERO_COPY_THRESHOLD (64 * 1024)  // 64KB

/**
 * Transfer file to socket using zero-copy
 * @param socket_fd Destination socket
 * @param file_fd Source file descriptor
 * @param offset Starting offset in file (updated on return)
 * @param count Number of bytes to transfer
 * @return Bytes transferred, -1 on error
 */
ssize_t zero_copy_file_to_socket(int socket_fd, int file_fd,
                                  off_t *offset, size_t count);

/**
 * Relay data between sockets using zero-copy (Linux splice)
 * Falls back to read/write on other platforms
 * @param dest_fd Destination socket
 * @param src_fd Source socket
 * @param count Maximum bytes to transfer
 * @return Bytes transferred, -1 on error
 */
ssize_t zero_copy_socket_relay(int dest_fd, int src_fd, size_t count);

/**
 * Check if zero-copy is available on this platform
 * @return Bitmask: 1=sendfile, 2=splice
 */
int zero_copy_available(void);

/**
 * Get zero-copy backend name
 * @return "sendfile+splice", "sendfile", or "none"
 */
const char* zero_copy_backend_name(void);

// Statistics
typedef struct {
    unsigned long sendfile_calls;
    unsigned long sendfile_bytes;
    unsigned long splice_calls;
    unsigned long splice_bytes;
    unsigned long fallback_calls;
    unsigned long fallback_bytes;
} zero_copy_stats_t;

extern zero_copy_stats_t g_zero_copy_stats;

#endif // ZERO_COPY_H
