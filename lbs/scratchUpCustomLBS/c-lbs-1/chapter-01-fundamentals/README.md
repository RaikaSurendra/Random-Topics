# Chapter 01: Socket Programming Fundamentals

## Learning Objectives

1. Understand file descriptors and sockets
2. Master the socket API lifecycle
3. Build a TCP echo server
4. Understand blocking vs non-blocking I/O

## The Socket Lifecycle

```
SERVER                                      CLIENT
──────                                      ──────

socket()   ← Create endpoint               socket()
   │                                           │
   ▼                                           │
bind()     ← Assign address                    │
   │                                           │
   ▼                                           │
listen()   ← Mark as passive                   │
   │                                           │
   ▼                                           ▼
accept()   ←─────── TCP Handshake ──────► connect()
   │              (SYN, SYN-ACK, ACK)          │
   ▼                                           ▼
read()  ◄──────────────────────────────── write()
   │                                           │
   ▼                                           ▼
write() ──────────────────────────────────► read()
   │                                           │
   ▼                                           ▼
close()                                    close()
```

## Key Concepts

### 1. File Descriptors

Everything in Unix is a file - including network sockets.

```c
int fd = socket(...);  // Returns integer (file descriptor)
// fd is an index into kernel's file table
// 0 = stdin, 1 = stdout, 2 = stderr
// Your socket will be 3, 4, 5, etc.
```

### 2. Socket System Calls

| Function | Purpose | Returns |
|----------|---------|---------|
| `socket()` | Create a socket | File descriptor |
| `bind()` | Assign local address | 0 on success |
| `listen()` | Mark as passive (server) | 0 on success |
| `accept()` | Wait for connection | New FD for client |
| `connect()` | Initiate connection (client) | 0 on success |
| `read()`/`recv()` | Receive data | Bytes read |
| `write()`/`send()` | Send data | Bytes written |
| `close()` | Close socket | 0 on success |

### 3. Address Structures

```c
// IPv4
struct sockaddr_in {
    sa_family_t    sin_family;  // AF_INET
    in_port_t      sin_port;    // Port (network byte order)
    struct in_addr sin_addr;    // IP address
};

// Generic (used in function signatures)
struct sockaddr {
    sa_family_t sa_family;
    char        sa_data[14];
};
```

### 4. Byte Order

Network uses **big-endian**. Your machine might use little-endian.

```c
htons()  // Host TO Network Short (16-bit, for ports)
htonl()  // Host TO Network Long (32-bit, for IPs)
ntohs()  // Network TO Host Short
ntohl()  // Network TO Host Long
```

## Files in This Chapter

| File | Description |
|------|-------------|
| `echo_server.c` | Simple TCP echo server |
| `echo_client.c` | Client to test the server |
| `getaddrinfo_demo.c` | Modern address resolution |

## Building & Running

```bash
# Compile
make

# Terminal 1: Start server
./echo_server 8080

# Terminal 2: Test with client
./echo_client 127.0.0.1 8080

# Or use netcat
echo "Hello" | nc localhost 8080
```

## Code Walkthrough

### Step 1: Create Socket

```c
int server_fd = socket(AF_INET, SOCK_STREAM, 0);
//                     │        │           │
//                     │        │           └─ Protocol (0 = auto)
//                     │        └─ Type (STREAM = TCP)
//                     └─ Family (INET = IPv4)
```

### Step 2: Set Socket Options

```c
int opt = 1;
setsockopt(server_fd, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt));
// Allows immediate rebind after server restart
// Without this: "Address already in use" error
```

### Step 3: Bind to Address

```c
struct sockaddr_in addr;
addr.sin_family = AF_INET;
addr.sin_addr.s_addr = INADDR_ANY;  // Bind to all interfaces
addr.sin_port = htons(8080);        // Port in network byte order

bind(server_fd, (struct sockaddr*)&addr, sizeof(addr));
```

### Step 4: Listen for Connections

```c
listen(server_fd, 10);  // Backlog queue of 10
// Marks socket as passive - ready to accept connections
```

### Step 5: Accept Connections

```c
struct sockaddr_in client_addr;
socklen_t client_len = sizeof(client_addr);

int client_fd = accept(server_fd, (struct sockaddr*)&client_addr, &client_len);
// Blocks until a client connects
// Returns NEW file descriptor for this specific client
// Original server_fd continues listening
```

### Step 6: Read/Write Data

```c
char buffer[1024];
ssize_t bytes_read = read(client_fd, buffer, sizeof(buffer));
// Returns: >0 (bytes read), 0 (connection closed), -1 (error)

write(client_fd, buffer, bytes_read);  // Echo back
```

### Step 7: Close Connection

```c
close(client_fd);  // Close client connection
close(server_fd);  // Close server (on shutdown)
```

## Common Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `Address already in use` | Port still bound | Use `SO_REUSEADDR` |
| `Connection refused` | No server listening | Start server first |
| `Bad file descriptor` | Using closed FD | Check FD validity |
| `Broken pipe` | Client disconnected | Handle `SIGPIPE` |

## Exercises

1. **Exercise 1:** Modify echo server to uppercase all received text
2. **Exercise 2:** Add logging to show client IP address
3. **Exercise 3:** Handle multiple clients (one at a time - iterative)
4. **Exercise 4:** Use `getaddrinfo()` instead of manual struct filling

## Key Takeaways

1. Sockets are file descriptors - use `read()`/`write()`
2. `accept()` returns a NEW socket for each client
3. Always use `htons()`/`ntohs()` for port numbers
4. `SO_REUSEADDR` prevents "address in use" errors
5. Check return values - network operations fail often

## Next Chapter

Chapter 02 builds on this to create a forward proxy, then a reverse proxy.
