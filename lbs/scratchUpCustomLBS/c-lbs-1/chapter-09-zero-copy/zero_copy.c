/**
 * Chapter 09: Zero-Copy Implementation
 * =====================================
 * Platform-specific zero-copy I/O
 */

#include "zero_copy.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <errno.h>
#include <fcntl.h>

#if defined(__linux__)
#include <sys/sendfile.h>
#endif

#if defined(__APPLE__) || defined(__FreeBSD__)
#include <sys/types.h>
#include <sys/socket.h>
#include <sys/uio.h>
#endif

// Global statistics
zero_copy_stats_t g_zero_copy_stats = {0};

int zero_copy_available(void) {
    int available = 0;
#if HAVE_SENDFILE
    available |= 1;
#endif
#if HAVE_SPLICE
    available |= 2;
#endif
    return available;
}

const char* zero_copy_backend_name(void) {
#if HAVE_SENDFILE && HAVE_SPLICE
    return "sendfile+splice";
#elif HAVE_SENDFILE
    return "sendfile";
#else
    return "none (fallback)";
#endif
}

// Fallback implementation using read/write
static ssize_t fallback_copy(int dest_fd, int src_fd, size_t count) {
    char buffer[65536];
    ssize_t total = 0;

    while (count > 0) {
        size_t to_read = count < sizeof(buffer) ? count : sizeof(buffer);
        ssize_t n = read(src_fd, buffer, to_read);

        if (n < 0) {
            if (errno == EINTR) continue;
            if (errno == EAGAIN || errno == EWOULDBLOCK) break;
            return total > 0 ? total : -1;
        }
        if (n == 0) break;  // EOF

        ssize_t written = 0;
        while (written < n) {
            ssize_t w = write(dest_fd, buffer + written, n - written);
            if (w < 0) {
                if (errno == EINTR) continue;
                if (errno == EAGAIN || errno == EWOULDBLOCK) break;
                return total > 0 ? total : -1;
            }
            written += w;
        }

        total += written;
        count -= n;

        if (written < n) break;  // Couldn't write all
    }

    g_zero_copy_stats.fallback_calls++;
    g_zero_copy_stats.fallback_bytes += total;

    return total;
}

ssize_t zero_copy_file_to_socket(int socket_fd, int file_fd,
                                  off_t *offset, size_t count) {
#if defined(__linux__) && HAVE_SENDFILE
    // Linux sendfile: sendfile(out_fd, in_fd, offset, count)
    ssize_t total = 0;
    off_t off = offset ? *offset : 0;

    while (count > 0) {
        ssize_t n = sendfile(socket_fd, file_fd, &off, count);

        if (n < 0) {
            if (errno == EINTR) continue;
            if (errno == EAGAIN || errno == EWOULDBLOCK) {
                break;  // Non-blocking would block
            }
            return total > 0 ? total : -1;
        }
        if (n == 0) break;

        total += n;
        count -= n;
    }

    if (offset) *offset = off;

    g_zero_copy_stats.sendfile_calls++;
    g_zero_copy_stats.sendfile_bytes += total;

    return total;

#elif (defined(__APPLE__) || defined(__FreeBSD__)) && HAVE_SENDFILE
    // macOS/BSD sendfile: sendfile(in_fd, out_fd, offset, &len, hdtr, flags)
    off_t sent = count;
    off_t off = offset ? *offset : 0;

    int result = sendfile(file_fd, socket_fd, off, &sent, NULL, 0);

    if (result < 0 && errno != EAGAIN && errno != EINTR) {
        return -1;
    }

    if (offset) *offset = off + sent;

    g_zero_copy_stats.sendfile_calls++;
    g_zero_copy_stats.sendfile_bytes += sent;

    return sent;

#else
    // Fallback to read/write
    (void)offset;  // Unused in fallback
    return fallback_copy(socket_fd, file_fd, count);
#endif
}

ssize_t zero_copy_socket_relay(int dest_fd, int src_fd, size_t count) {
#if defined(__linux__) && HAVE_SPLICE
    // Linux splice: socket → pipe → socket (no user-space copy)
    int pipefd[2];
    if (pipe(pipefd) < 0) {
        return fallback_copy(dest_fd, src_fd, count);
    }

    ssize_t total = 0;

    while (count > 0) {
        // Step 1: splice from src_fd to pipe
        ssize_t n = splice(src_fd, NULL, pipefd[1], NULL,
                          count, SPLICE_F_MOVE | SPLICE_F_NONBLOCK);

        if (n < 0) {
            if (errno == EINTR) continue;
            if (errno == EAGAIN || errno == EWOULDBLOCK) break;
            close(pipefd[0]);
            close(pipefd[1]);
            return total > 0 ? total : fallback_copy(dest_fd, src_fd, count);
        }
        if (n == 0) break;  // EOF

        // Step 2: splice from pipe to dest_fd
        ssize_t written = 0;
        while (written < n) {
            ssize_t w = splice(pipefd[0], NULL, dest_fd, NULL,
                              n - written, SPLICE_F_MOVE | SPLICE_F_NONBLOCK);

            if (w < 0) {
                if (errno == EINTR) continue;
                if (errno == EAGAIN || errno == EWOULDBLOCK) break;
                close(pipefd[0]);
                close(pipefd[1]);
                return total > 0 ? total : -1;
            }
            written += w;
        }

        total += written;
        count -= written;

        if (written < n) break;
    }

    close(pipefd[0]);
    close(pipefd[1]);

    g_zero_copy_stats.splice_calls++;
    g_zero_copy_stats.splice_bytes += total;

    return total;

#else
    // No splice available - use fallback
    return fallback_copy(dest_fd, src_fd, count);
#endif
}
