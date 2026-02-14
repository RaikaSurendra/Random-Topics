# Chapter 10: Hot Reload & Configuration

## Learning Objectives

1. Implement JSON configuration file parsing
2. Understand zero-downtime reload with SO_REUSEPORT
3. Implement graceful connection draining
4. Design production-ready configuration management

## Why Hot Reload?

```
WITHOUT HOT RELOAD
═══════════════════════════════════════════════════════════════

1. Change config file
2. Stop load balancer  ← DOWNTIME STARTS
3. All connections dropped!
4. Start load balancer ← DOWNTIME ENDS
5. Users angry about 503 errors


WITH HOT RELOAD
═══════════════════════════════════════════════════════════════

1. Change config file
2. Signal reload (SIGHUP)
3. New process starts with new config
4. Old process drains existing connections
5. Old process exits when done
6. ZERO DOWNTIME!
```

## SO_REUSEPORT Magic

```c
// Allow multiple processes to bind same port
int opt = 1;
setsockopt(fd, SOL_SOCKET, SO_REUSEPORT, &opt, sizeof(opt));

// Now both old and new process can accept on port 8080!
// Kernel load-balances between them during transition
```

## Reload Sequence

```
┌─────────────────────────────────────────────────────────────┐
│                    HOT RELOAD SEQUENCE                      │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  T=0: Admin sends SIGHUP to load balancer                  │
│        $ kill -HUP $(cat /var/run/lb.pid)                  │
│                                                             │
│  T=1: Old process forks new process                        │
│        ┌──────────┐        ┌──────────┐                    │
│        │   Old    │───────▶│   New    │                    │
│        │ (config  │        │ (config  │                    │
│        │    v1)   │        │    v2)   │                    │
│        └──────────┘        └──────────┘                    │
│                                                             │
│  T=2: Both processes accept connections (SO_REUSEPORT)     │
│        Old: Existing connections (draining)                │
│        New: All new connections                             │
│                                                             │
│  T=3: Old process finishes draining, exits                 │
│        ┌──────────┐                                        │
│        │   New    │ ← Now sole owner of port              │
│        │ (config  │                                        │
│        │    v2)   │                                        │
│        └──────────┘                                        │
│                                                             │
│  Total downtime: 0 seconds                                 │
│  Dropped connections: 0                                    │
└─────────────────────────────────────────────────────────────┘
```

## Configuration File Format

```json
{
  "listen_port": 8080,
  "algorithm": "weighted_round_robin",

  "backends": [
    {"host": "127.0.0.1", "port": 9001, "weight": 3},
    {"host": "127.0.0.1", "port": 9002, "weight": 2}
  ],

  "pool": {
    "max_size": 64,
    "ttl": 60
  },

  "rate_limit": {
    "per_ip": 100.0,
    "burst": 20
  },

  "drain_timeout": 30
}
```

## API

```c
// Load and parse config
config_t* config_load(const char *filename);

// Validate before applying
int config_validate(config_t *cfg);

// Check if file changed (for auto-reload)
int config_changed(config_t *cfg);

// Hot reload
config_t* config_reload(const char *filename);
```

## Connection Draining

```c
void reload_start_drain(reload_state_t *state, int timeout) {
    state->is_draining = 1;
    state->drain_start = time(NULL);
    state->drain_timeout = timeout;

    // Stop accepting new connections
    close(server_fd);

    // Wait for existing to complete
    while (state->active_connections > 0) {
        time_t elapsed = time(NULL) - state->drain_start;
        if (elapsed > timeout) break;  // Force exit

        // Process existing connections
        event_loop_run(loop, 100);
    }
}
```

## Signal Handling

```c
void signal_handler(int sig) {
    if (sig == SIGHUP) {
        // Reload config
        config_t *new_cfg = config_reload(config_file);
        if (config_validate(new_cfg)) {
            apply_config(new_cfg);
        }
    } else if (sig == SIGUSR2) {
        // Graceful shutdown (from new process)
        reload_start_drain(&state, 30);
    }
}
```

## Validation Before Apply

```c
config_t* safe_reload(const char *filename) {
    // 1. Parse new config
    config_t *new_cfg = config_load(filename);
    if (!new_cfg) return NULL;

    // 2. Validate
    if (!config_validate(new_cfg)) {
        fprintf(stderr, "Invalid config, keeping old\n");
        config_free(new_cfg);
        return NULL;
    }

    // 3. Test backend connectivity
    for (int i = 0; i < new_cfg->num_backends; i++) {
        if (!check_backend_health(&new_cfg->backends[i])) {
            fprintf(stderr, "Backend %d unreachable\n", i);
            // Warning only, don't fail
        }
    }

    return new_cfg;  // Safe to apply
}
```

## Exercises

1. Implement config file watching with `inotify` (Linux) or `kqueue` (macOS)
2. Add config diff logging (show what changed)
3. Implement config rollback on failure
4. Add YAML config format support

## Key Takeaways

1. **SO_REUSEPORT** enables zero-downtime reload
2. **Drain connections** before shutting down old process
3. **Validate config** before applying changes
4. **Use signals** (SIGHUP, SIGUSR2) for reload coordination
5. **PID files** track running instances
