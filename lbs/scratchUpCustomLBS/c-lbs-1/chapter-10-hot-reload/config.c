/**
 * Chapter 10: Configuration & Hot Reload Implementation
 * ======================================================
 * JSON config parsing and zero-downtime reload
 */

#include "config.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/stat.h>
#include <signal.h>
#include <fcntl.h>
#include <errno.h>
#include <ctype.h>

// Simple JSON parser (no external dependencies)
// Handles basic JSON for config files

static void skip_whitespace(const char **p) {
    while (**p && isspace(**p)) (*p)++;
}

static int parse_string(const char **p, char *out, size_t max) {
    skip_whitespace(p);
    if (**p != '"') return -1;
    (*p)++;

    size_t i = 0;
    while (**p && **p != '"' && i < max - 1) {
        if (**p == '\\' && *(*p + 1)) {
            (*p)++;
            switch (**p) {
                case 'n': out[i++] = '\n'; break;
                case 't': out[i++] = '\t'; break;
                case '"': out[i++] = '"'; break;
                case '\\': out[i++] = '\\'; break;
                default: out[i++] = **p;
            }
        } else {
            out[i++] = **p;
        }
        (*p)++;
    }
    out[i] = '\0';

    if (**p == '"') (*p)++;
    return 0;
}

static int parse_int(const char **p) {
    skip_whitespace(p);
    int negative = 0;
    if (**p == '-') {
        negative = 1;
        (*p)++;
    }
    int value = 0;
    while (**p && isdigit(**p)) {
        value = value * 10 + (**p - '0');
        (*p)++;
    }
    return negative ? -value : value;
}

static double parse_double(const char **p) {
    skip_whitespace(p);
    char buf[64];
    int i = 0;
    while (**p && (isdigit(**p) || **p == '.' || **p == '-' || **p == 'e' || **p == 'E') && i < 63) {
        buf[i++] = **p;
        (*p)++;
    }
    buf[i] = '\0';
    return atof(buf);
}

static void skip_value(const char **p);

static void skip_object(const char **p) {
    skip_whitespace(p);
    if (**p != '{') return;
    (*p)++;
    int depth = 1;
    while (**p && depth > 0) {
        if (**p == '{') depth++;
        else if (**p == '}') depth--;
        else if (**p == '"') {
            (*p)++;
            while (**p && **p != '"') {
                if (**p == '\\' && *(*p + 1)) (*p)++;
                (*p)++;
            }
        }
        if (**p) (*p)++;
    }
}

static void skip_array(const char **p) {
    skip_whitespace(p);
    if (**p != '[') return;
    (*p)++;
    int depth = 1;
    while (**p && depth > 0) {
        if (**p == '[') depth++;
        else if (**p == ']') depth--;
        else if (**p == '"') {
            (*p)++;
            while (**p && **p != '"') {
                if (**p == '\\' && *(*p + 1)) (*p)++;
                (*p)++;
            }
        }
        if (**p) (*p)++;
    }
}

static void skip_value(const char **p) {
    skip_whitespace(p);
    if (**p == '"') {
        char tmp[1024];
        parse_string(p, tmp, sizeof(tmp));
    } else if (**p == '{') {
        skip_object(p);
    } else if (**p == '[') {
        skip_array(p);
    } else {
        while (**p && **p != ',' && **p != '}' && **p != ']') (*p)++;
    }
}

config_t* config_load(const char *filename) {
    FILE *f = fopen(filename, "r");
    if (!f) {
        perror("config_load");
        return NULL;
    }

    // Read entire file
    fseek(f, 0, SEEK_END);
    long size = ftell(f);
    fseek(f, 0, SEEK_SET);

    char *json = malloc(size + 1);
    if (!json) {
        fclose(f);
        return NULL;
    }

    fread(json, 1, size, f);
    json[size] = '\0';
    fclose(f);

    config_t *cfg = calloc(1, sizeof(config_t));
    if (!cfg) {
        free(json);
        return NULL;
    }

    // Set defaults
    cfg->listen_port = 8080;
    strcpy(cfg->bind_address, "0.0.0.0");
    cfg->backlog = 128;
    strcpy(cfg->algorithm, "round_robin");
    cfg->connect_timeout_ms = 5000;
    cfg->read_timeout_ms = 30000;
    cfg->write_timeout_ms = 30000;
    cfg->idle_timeout_ms = 60000;
    cfg->drain_timeout_seconds = 30;
    cfg->pool.max_size = 64;
    cfg->pool.ttl_seconds = 60;
    cfg->pool.max_requests = 1000;
    cfg->rate_limit.per_ip_rate = 100;
    cfg->rate_limit.burst_size = 10;

    strncpy(cfg->config_file, filename, sizeof(cfg->config_file) - 1);
    cfg->loaded_at = time(NULL);

    // Get file mtime
    struct stat st;
    if (stat(filename, &st) == 0) {
        cfg->file_mtime = st.st_mtime;
    }

    // Parse JSON
    const char *p = json;
    skip_whitespace(&p);
    if (*p != '{') goto done;
    p++;

    while (*p && *p != '}') {
        skip_whitespace(&p);
        if (*p == '}') break;

        char key[128];
        if (parse_string(&p, key, sizeof(key)) < 0) break;

        skip_whitespace(&p);
        if (*p == ':') p++;
        skip_whitespace(&p);

        if (strcmp(key, "listen_port") == 0) {
            cfg->listen_port = parse_int(&p);
        } else if (strcmp(key, "bind_address") == 0) {
            parse_string(&p, cfg->bind_address, sizeof(cfg->bind_address));
        } else if (strcmp(key, "backlog") == 0) {
            cfg->backlog = parse_int(&p);
        } else if (strcmp(key, "algorithm") == 0) {
            parse_string(&p, cfg->algorithm, sizeof(cfg->algorithm));
        } else if (strcmp(key, "backends") == 0) {
            skip_whitespace(&p);
            if (*p == '[') {
                p++;
                while (*p && *p != ']' && cfg->num_backends < MAX_CONFIG_BACKENDS) {
                    skip_whitespace(&p);
                    if (*p == '{') {
                        p++;
                        ConfigBackend *b = &cfg->backends[cfg->num_backends];
                        b->weight = 1;
                        b->max_connections = 100;

                        while (*p && *p != '}') {
                            skip_whitespace(&p);
                            char bkey[64];
                            if (parse_string(&p, bkey, sizeof(bkey)) < 0) break;
                            skip_whitespace(&p);
                            if (*p == ':') p++;
                            skip_whitespace(&p);

                            if (strcmp(bkey, "host") == 0) {
                                parse_string(&p, b->host, sizeof(b->host));
                            } else if (strcmp(bkey, "port") == 0) {
                                int port = parse_int(&p);
                                snprintf(b->port, sizeof(b->port), "%d", port);
                            } else if (strcmp(bkey, "weight") == 0) {
                                b->weight = parse_int(&p);
                            } else if (strcmp(bkey, "max_connections") == 0) {
                                b->max_connections = parse_int(&p);
                            } else {
                                skip_value(&p);
                            }

                            skip_whitespace(&p);
                            if (*p == ',') p++;
                        }
                        if (*p == '}') p++;
                        cfg->num_backends++;
                    }
                    skip_whitespace(&p);
                    if (*p == ',') p++;
                }
                if (*p == ']') p++;
            }
        } else if (strcmp(key, "pool") == 0) {
            skip_whitespace(&p);
            if (*p == '{') {
                p++;
                cfg->pool.enabled = 1;
                while (*p && *p != '}') {
                    skip_whitespace(&p);
                    char pkey[64];
                    if (parse_string(&p, pkey, sizeof(pkey)) < 0) break;
                    skip_whitespace(&p);
                    if (*p == ':') p++;

                    if (strcmp(pkey, "max_size") == 0) {
                        cfg->pool.max_size = parse_int(&p);
                    } else if (strcmp(pkey, "ttl") == 0) {
                        cfg->pool.ttl_seconds = parse_int(&p);
                    } else {
                        skip_value(&p);
                    }

                    skip_whitespace(&p);
                    if (*p == ',') p++;
                }
                if (*p == '}') p++;
            }
        } else if (strcmp(key, "rate_limit") == 0) {
            skip_whitespace(&p);
            if (*p == '{') {
                p++;
                cfg->rate_limit.enabled = 1;
                while (*p && *p != '}') {
                    skip_whitespace(&p);
                    char rkey[64];
                    if (parse_string(&p, rkey, sizeof(rkey)) < 0) break;
                    skip_whitespace(&p);
                    if (*p == ':') p++;

                    if (strcmp(rkey, "per_ip") == 0) {
                        cfg->rate_limit.per_ip_rate = parse_double(&p);
                    } else if (strcmp(rkey, "global") == 0) {
                        cfg->rate_limit.global_rate = parse_double(&p);
                    } else if (strcmp(rkey, "burst") == 0) {
                        cfg->rate_limit.burst_size = parse_int(&p);
                    } else {
                        skip_value(&p);
                    }

                    skip_whitespace(&p);
                    if (*p == ',') p++;
                }
                if (*p == '}') p++;
            }
        } else {
            skip_value(&p);
        }

        skip_whitespace(&p);
        if (*p == ',') p++;
    }

done:
    free(json);
    return cfg;
}

int config_validate(config_t *cfg) {
    if (!cfg) return 0;

    if (cfg->listen_port < 1 || cfg->listen_port > 65535) {
        fprintf(stderr, "Invalid port: %d\n", cfg->listen_port);
        return 0;
    }

    if (cfg->num_backends == 0) {
        fprintf(stderr, "No backends configured\n");
        return 0;
    }

    for (int i = 0; i < cfg->num_backends; i++) {
        ConfigBackend *b = &cfg->backends[i];
        if (b->host[0] == '\0' || b->port[0] == '\0') {
            fprintf(stderr, "Invalid backend %d\n", i);
            return 0;
        }
    }

    return 1;
}

int config_changed(config_t *cfg) {
    if (!cfg) return 0;

    struct stat st;
    if (stat(cfg->config_file, &st) != 0) {
        return 0;
    }

    return st.st_mtime != cfg->file_mtime;
}

config_t* config_reload(const char *filename) {
    config_t *new_cfg = config_load(filename);
    if (!new_cfg) return NULL;

    if (!config_validate(new_cfg)) {
        config_free(new_cfg);
        return NULL;
    }

    return new_cfg;
}

int config_equal(config_t *a, config_t *b) {
    if (!a || !b) return 0;

    if (a->listen_port != b->listen_port) return 0;
    if (strcmp(a->algorithm, b->algorithm) != 0) return 0;
    if (a->num_backends != b->num_backends) return 0;

    for (int i = 0; i < a->num_backends; i++) {
        if (strcmp(a->backends[i].host, b->backends[i].host) != 0) return 0;
        if (strcmp(a->backends[i].port, b->backends[i].port) != 0) return 0;
        if (a->backends[i].weight != b->backends[i].weight) return 0;
    }

    return 1;
}

void config_free(config_t *cfg) {
    free(cfg);
}

void config_print(config_t *cfg) {
    printf("Configuration:\n");
    printf("  Listen: %s:%d\n", cfg->bind_address, cfg->listen_port);
    printf("  Algorithm: %s\n", cfg->algorithm);
    printf("  Backends (%d):\n", cfg->num_backends);
    for (int i = 0; i < cfg->num_backends; i++) {
        printf("    [%d] %s:%s weight=%d\n", i,
               cfg->backends[i].host, cfg->backends[i].port,
               cfg->backends[i].weight);
    }
    if (cfg->pool.enabled) {
        printf("  Pool: size=%d ttl=%ds\n",
               cfg->pool.max_size, cfg->pool.ttl_seconds);
    }
    if (cfg->rate_limit.enabled) {
        printf("  Rate Limit: %.1f/s burst=%d\n",
               cfg->rate_limit.per_ip_rate, cfg->rate_limit.burst_size);
    }
}

// Reload state management

int reload_init(reload_state_t *state, const char *pid_file) {
    memset(state, 0, sizeof(reload_state_t));
    strncpy(state->pid_file, pid_file, sizeof(state->pid_file) - 1);

    // Check for existing process
    int fd = open(pid_file, O_RDONLY);
    if (fd >= 0) {
        char buf[32];
        ssize_t n = read(fd, buf, sizeof(buf) - 1);
        close(fd);

        if (n > 0) {
            buf[n] = '\0';
            pid_t old_pid = atoi(buf);

            // Check if still running
            if (old_pid > 0 && kill(old_pid, 0) == 0) {
                return old_pid;
            }
        }
    }

    // Write our PID
    fd = open(pid_file, O_WRONLY | O_CREAT | O_TRUNC, 0644);
    if (fd >= 0) {
        char buf[32];
        int n = snprintf(buf, sizeof(buf), "%d\n", getpid());
        write(fd, buf, n);
        close(fd);
    }

    return 0;
}

void reload_start_drain(reload_state_t *state, int timeout_seconds) {
    state->is_draining = 1;
    state->drain_start = time(NULL);
    state->drain_timeout = timeout_seconds;
}

int reload_drain_complete(reload_state_t *state) {
    if (!state->is_draining) return 1;

    if (state->active_connections == 0) return 1;

    time_t elapsed = time(NULL) - state->drain_start;
    if (elapsed >= state->drain_timeout) return 1;

    return 0;
}

int reload_signal_old(reload_state_t *state) {
    int fd = open(state->pid_file, O_RDONLY);
    if (fd < 0) return -1;

    char buf[32];
    ssize_t n = read(fd, buf, sizeof(buf) - 1);
    close(fd);

    if (n <= 0) return -1;

    buf[n] = '\0';
    pid_t old_pid = atoi(buf);

    if (old_pid > 0 && old_pid != getpid()) {
        return kill(old_pid, SIGUSR2);
    }

    return -1;
}

// ============================================================================
// Test/Demo Main Functions
// ============================================================================

#ifdef CONFIG_TEST
// Unit test for config parsing
int main(int argc, char *argv[]) {
    if (argc != 2) {
        fprintf(stderr, "Usage: %s <config.json>\n", argv[0]);
        return 1;
    }

    printf("=== Config Test ===\n\n");

    config_t *cfg = config_load(argv[1]);
    if (!cfg) {
        fprintf(stderr, "Failed to load config\n");
        return 1;
    }

    printf("Loaded config from: %s\n\n", argv[1]);
    config_print(cfg);

    printf("\nValidating...\n");
    if (config_validate(cfg)) {
        printf("Config is VALID\n");
    } else {
        printf("Config is INVALID\n");
    }

    config_free(cfg);
    return 0;
}
#endif

#ifdef HOT_RELOAD_LB
// Demo hot reload load balancer
#include <signal.h>

static volatile int g_running = 1;
static volatile int g_reload = 0;
static config_t *g_config = NULL;
static const char *g_config_file = NULL;

void handle_sighup(int sig) {
    (void)sig;
    g_reload = 1;
}

void handle_sigterm(int sig) {
    (void)sig;
    g_running = 0;
}

int main(int argc, char *argv[]) {
    if (argc != 2) {
        fprintf(stderr, "Usage: %s <config.json>\n", argv[0]);
        fprintf(stderr, "\nThis is a demo of the hot reload system.\n");
        fprintf(stderr, "Send SIGHUP to reload config, SIGTERM to stop.\n");
        return 1;
    }

    g_config_file = argv[1];

    // Load initial config
    g_config = config_load(g_config_file);
    if (!g_config) {
        fprintf(stderr, "Failed to load config: %s\n", g_config_file);
        return 1;
    }

    if (!config_validate(g_config)) {
        fprintf(stderr, "Invalid config\n");
        config_free(g_config);
        return 1;
    }

    // Setup signal handlers
    signal(SIGHUP, handle_sighup);
    signal(SIGTERM, handle_sigterm);
    signal(SIGINT, handle_sigterm);

    printf("=== Hot Reload Load Balancer Demo ===\n\n");
    printf("PID: %d\n", getpid());
    printf("Config: %s\n\n", g_config_file);
    config_print(g_config);
    printf("\nCommands:\n");
    printf("  kill -HUP %d   # Reload config\n", getpid());
    printf("  kill %d        # Stop\n", getpid());
    printf("\nWaiting for signals...\n\n");

    // Main loop
    while (g_running) {
        // Check for reload signal
        if (g_reload) {
            g_reload = 0;
            printf("\n[SIGHUP] Reloading config...\n");

            config_t *new_cfg = config_reload(g_config_file);
            if (new_cfg) {
                if (!config_equal(g_config, new_cfg)) {
                    printf("Config changed:\n");
                    config_print(new_cfg);

                    config_t *old = g_config;
                    g_config = new_cfg;
                    config_free(old);

                    printf("Reload complete!\n\n");
                } else {
                    printf("Config unchanged.\n\n");
                    config_free(new_cfg);
                }
            } else {
                printf("Reload FAILED - keeping old config\n\n");
            }
        }

        // Check for file changes (auto-reload)
        if (config_changed(g_config)) {
            printf("\n[FILE CHANGE] Config file modified, reloading...\n");
            g_reload = 1;
        }

        sleep(1);
    }

    printf("\n[SIGTERM] Shutting down...\n");
    config_free(g_config);
    return 0;
}
#endif
