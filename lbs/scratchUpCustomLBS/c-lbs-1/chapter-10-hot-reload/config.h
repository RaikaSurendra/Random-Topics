/**
 * Chapter 10: Hot Reload & Configuration
 * ======================================
 *
 * Zero-downtime configuration changes:
 * - JSON configuration file support
 * - SIGHUP-triggered reload
 * - SO_REUSEPORT for seamless restart
 * - Connection draining
 *
 * Usage:
 *   config_t *cfg = config_load("lb.json");
 *   if (config_validate(cfg)) {
 *       // Apply configuration
 *   }
 *   // On SIGHUP:
 *   config_reload("lb.json");
 */

#ifndef CONFIG_H
#define CONFIG_H

#include <time.h>

#define MAX_CONFIG_BACKENDS 32
#define CONFIG_PATH_MAX 512

// Backend configuration
typedef struct {
    char host[256];
    char port[6];
    int weight;
    int max_connections;
} ConfigBackend;

// Rate limiting configuration
typedef struct {
    int enabled;
    double per_ip_rate;      // Requests per second per IP
    double global_rate;      // Global requests per second
    int burst_size;          // Token bucket burst
} ConfigRateLimit;

// Connection pool configuration
typedef struct {
    int enabled;
    int max_size;            // Max pooled connections
    int ttl_seconds;         // Connection TTL
    int max_requests;        // Max requests per connection
} ConfigPool;

// Full configuration
typedef struct {
    // Server settings
    int listen_port;
    char bind_address[64];
    int backlog;

    // Backends
    ConfigBackend backends[MAX_CONFIG_BACKENDS];
    int num_backends;

    // Load balancing
    char algorithm[32];      // "round_robin", "weighted", etc.

    // Features
    ConfigRateLimit rate_limit;
    ConfigPool pool;

    // Timeouts
    int connect_timeout_ms;
    int read_timeout_ms;
    int write_timeout_ms;
    int idle_timeout_ms;

    // Graceful shutdown
    int drain_timeout_seconds;

    // Metadata
    char config_file[CONFIG_PATH_MAX];
    time_t loaded_at;
    time_t file_mtime;
} config_t;

/**
 * Load configuration from JSON file
 * @param filename Path to config file
 * @return Config struct or NULL on error
 */
config_t* config_load(const char *filename);

/**
 * Validate configuration
 * @param cfg Configuration to validate
 * @return 1 if valid, 0 if invalid
 */
int config_validate(config_t *cfg);

/**
 * Check if config file has changed
 * @param cfg Current configuration
 * @return 1 if file changed, 0 otherwise
 */
int config_changed(config_t *cfg);

/**
 * Reload configuration
 * @param filename Config file path
 * @return New config or NULL on error
 */
config_t* config_reload(const char *filename);

/**
 * Compare two configurations
 * @return 1 if equal, 0 if different
 */
int config_equal(config_t *a, config_t *b);

/**
 * Free configuration
 */
void config_free(config_t *cfg);

/**
 * Print configuration (for debugging)
 */
void config_print(config_t *cfg);

// Reload management
typedef struct {
    int is_draining;         // Currently draining connections
    int active_connections;  // Connections still active
    time_t drain_start;      // When drain started
    int drain_timeout;       // Max drain time
    char pid_file[256];      // PID file path
} reload_state_t;

/**
 * Initialize reload state
 * @param pid_file Path to PID file
 * @return Old process PID if exists, 0 otherwise
 */
int reload_init(reload_state_t *state, const char *pid_file);

/**
 * Start draining connections for reload
 */
void reload_start_drain(reload_state_t *state, int timeout_seconds);

/**
 * Check if drain is complete
 */
int reload_drain_complete(reload_state_t *state);

/**
 * Signal old process to shutdown
 */
int reload_signal_old(reload_state_t *state);

#endif // CONFIG_H
