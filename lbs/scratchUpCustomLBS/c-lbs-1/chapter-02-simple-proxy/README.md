# Chapter 02: Forward & Reverse Proxy

## Learning Objectives

1. Understand the difference between forward and reverse proxy
2. Build a forward proxy (client-side)
3. Build a reverse proxy (server-side)
4. Handle bidirectional data transfer

## Forward vs Reverse Proxy

### Forward Proxy (CONNECT method)

```
┌──────────┐         ┌──────────────┐         ┌──────────────┐
│  Client  │────────▶│   FORWARD    │────────▶│   Internet   │
│ Browser  │         │    PROXY     │         │   Server     │
└──────────┘         └──────────────┘         └──────────────┘
      │                     │                        │
      │  "Connect me to    │  "I'll connect for    │
      │   example.com"     │   you and relay"      │
      └────────────────────┴────────────────────────┘

CLIENT KNOWS about proxy (configured in browser/system)
SERVER DOESN'T KNOW client's real IP
```

### Reverse Proxy

```
┌──────────┐         ┌──────────────┐         ┌──────────────┐
│  Client  │────────▶│   REVERSE    │────────▶│   Backend    │
│          │         │    PROXY     │         │   Server     │
└──────────┘         └──────────────┘         └──────────────┘
      │                     │                        │
      │  "GET /page from   │  "I'll fetch from     │
      │   proxy.com"       │   hidden backend"     │
      └────────────────────┴────────────────────────┘

CLIENT DOESN'T KNOW about backend (thinks proxy IS the server)
SERVER KNOWS it's behind a proxy (sees proxy's IP)
```

## Data Flow in Reverse Proxy

```
┌────────────────────────────────────────────────────────────────┐
│                      REVERSE PROXY                              │
│                                                                 │
│   client_fd ◄────────────────────────────────► backend_fd      │
│                                                                 │
│   1. Accept client connection (client_fd)                      │
│   2. Connect to backend (backend_fd)                           │
│   3. Read from client_fd → Write to backend_fd                 │
│   4. Read from backend_fd → Write to client_fd                 │
│   5. Close both when done                                       │
│                                                                 │
└────────────────────────────────────────────────────────────────┘
```

## Files in This Chapter

| File | Description |
|------|-------------|
| `forward_proxy.c` | HTTP CONNECT tunnel proxy |
| `reverse_proxy.c` | Simple reverse proxy |
| `Makefile` | Build configuration |

## The Proxying Algorithm

```c
// Simplified pseudocode
void proxy_data(int client_fd, int backend_fd) {
    // Read request from client
    bytes = read(client_fd, buffer, size);

    // Forward to backend
    write(backend_fd, buffer, bytes);

    // Read response from backend
    bytes = read(backend_fd, buffer, size);

    // Send back to client
    write(client_fd, buffer, bytes);
}
```

## Key Implementation Details

### 1. Connecting to Backend

```c
// Resolve backend hostname
struct addrinfo hints, *result;
memset(&hints, 0, sizeof(hints));
hints.ai_family = AF_INET;
hints.ai_socktype = SOCK_STREAM;

getaddrinfo(backend_host, backend_port, &hints, &result);

// Create socket and connect
int backend_fd = socket(result->ai_family, result->ai_socktype, result->ai_protocol);
connect(backend_fd, result->ai_addr, result->ai_addrlen);
```

### 2. Relaying Data

The challenge: data can come from either direction at any time.

**Simple approach (sequential):**
```c
// Read all from client, send to backend
// Read all from backend, send to client
// Works for HTTP/1.0 but not for streaming
```

**Better approach (using select/poll/epoll):**
```c
// Monitor both fds simultaneously
// When data available on either, relay it
// More complex but handles all cases
```

### 3. Handling HTTP Headers

For reverse proxy, we might want to add headers:

```c
// Add X-Forwarded-For header
char header[256];
snprintf(header, sizeof(header), "X-Forwarded-For: %s\r\n", client_ip);
```

## Building & Running

```bash
# Compile
make

# Terminal 1: Start a backend (our echo server from Chapter 1)
cd ../chapter-01-fundamentals && ./echo_server 9000

# Terminal 2: Start reverse proxy
./reverse_proxy 8080 127.0.0.1 9000

# Terminal 3: Test through proxy
echo "Hello via proxy" | nc localhost 8080
```

## Testing with Real HTTP

```bash
# Start a simple Python HTTP server as backend
python3 -m http.server 9000 &

# Start our reverse proxy
./reverse_proxy 8080 127.0.0.1 9000

# Test with curl
curl http://localhost:8080/
```

## Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| Connection hangs | Blocking read | Use timeouts or non-blocking I/O |
| Partial data | Single read not enough | Loop until complete |
| Broken pipe | Client disconnected | Handle SIGPIPE |
| Backend refused | Wrong host/port | Check backend is running |

## Exercises

1. **Add logging**: Log each request with timestamp, client IP, path
2. **Add headers**: Insert X-Forwarded-For and X-Real-IP headers
3. **Handle errors**: Gracefully handle backend failures (return 502)
4. **Support HTTPS**: Use SSL/TLS (openssl library)

## Comparison: Forward vs Reverse Proxy Code

| Aspect | Forward Proxy | Reverse Proxy |
|--------|---------------|---------------|
| Backend | Dynamic (from request) | Fixed (config) |
| Client config | Required | Not needed |
| Use case | Client anonymity | Server protection |
| Typical port | 8080, 3128 | 80, 443 |

## Next Chapter

Chapter 03 extends this to multiple backends with load balancing.
