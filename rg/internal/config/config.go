package config

import (
	"fmt"
	"os"
	"strconv"
)

// NOTE: Config fields use yaml tags for future YAML unmarshaling support
// TODO: Add validation for required fields after loading

// Config holds all application configuration.
type Config struct {
	AppName     string         `yaml:"app_name" json:"appName"`
	Environment string         `yaml:"environment" json:"environment"`
	Port        int            `yaml:"port" json:"port"`
	Debug       bool           `yaml:"debug" json:"debug"`
	Database    DatabaseConfig `yaml:"database" json:"database"`
	Auth        AuthConfig     `yaml:"auth" json:"auth"`
	CORS        CORSConfig     `yaml:"cors" json:"cors"`
}

// DatabaseConfig holds database connection settings.
type DatabaseConfig struct {
	Host            string `yaml:"host" json:"host"`
	Port            int    `yaml:"port" json:"port"`
	User            string `yaml:"user" json:"user"`
	Password        string `yaml:"password" json:"password"`
	DBName          string `yaml:"dbname" json:"dbName"`
	SSLMode         string `yaml:"sslmode" json:"sslMode"`
	MaxOpenConns    int    `yaml:"max_open_conns" json:"maxOpenConns"`
	MaxIdleConns    int    `yaml:"max_idle_conns" json:"maxIdleConns"`
	ConnMaxLifetime int    `yaml:"conn_max_lifetime" json:"connMaxLifetime"` // in seconds
}

// AuthConfig holds authentication settings.
type AuthConfig struct {
	JWTSecret       string `yaml:"jwt_secret" json:"jwtSecret"`
	TokenExpiry     int    `yaml:"token_expiry" json:"tokenExpiry"` // in minutes
	BcryptCost      int    `yaml:"bcrypt_cost" json:"bcryptCost"`
	SessionTimeout  int    `yaml:"session_timeout" json:"sessionTimeout"` // FIXME: not implemented yet
}

// CORSConfig holds CORS settings.
type CORSConfig struct {
	AllowedOrigins []string `yaml:"allowed_origins" json:"allowedOrigins"`
	AllowedMethods []string `yaml:"allowed_methods" json:"allowedMethods"`
}

// DefaultConfig returns sensible defaults for development.
// HACK: Hardcoded values should be loaded from a config file in production
func DefaultConfig() *Config {
	return &Config{
		AppName:     "webshop-api",
		Environment: "development",
		Port:        8080,
		Debug:       true,
		Database: DatabaseConfig{
			Host:            "localhost",       // FIXME: Should be configurable
			Port:            5432,
			User:            "webshop_user",
			Password:        "webshop_pass123", // BUG: Hardcoded password in source
			DBName:          "webshop_dev",
			SSLMode:         "disable",         // NOTE: Enable in production!
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: 300,
		},
		Auth: AuthConfig{
			JWTSecret:      "super-secret-key-change-me", // TODO: Load from env var
			TokenExpiry:    60,
			BcryptCost:     10,
			SessionTimeout: 3600,
		},
		CORS: CORSConfig{
			AllowedOrigins: []string{"http://localhost:3000", "http://localhost:5173"},
			AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		},
	}
}

// Load reads configuration from a YAML file with env var overrides.
// TODO: Actually implement YAML parsing - currently only uses env vars and defaults
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	// Check if config file exists
	if _, err := os.Stat(path); err != nil {
		fmt.Printf("[WARN] Config file not found at %s, using defaults\n", path)
	}

	// Override from environment variables
	if port := os.Getenv("APP_PORT"); port != "" {
		p, err := strconv.Atoi(port)
		if err != nil {
			return nil, fmt.Errorf("invalid APP_PORT value %q: %w", port, err)
		}
		cfg.Port = p
	}
	if dbHost := os.Getenv("DB_HOST"); dbHost != "" {
		cfg.Database.Host = dbHost
	}
	if dbPort := os.Getenv("DB_PORT"); dbPort != "" {
		p, err := strconv.Atoi(dbPort)
		if err != nil {
			return nil, fmt.Errorf("invalid DB_PORT value %q: %w", dbPort, err)
		}
		cfg.Database.Port = p
	}
	if dbUser := os.Getenv("DB_USER"); dbUser != "" {
		cfg.Database.User = dbUser
	}
	if dbPass := os.Getenv("DB_PASSWORD"); dbPass != "" {
		cfg.Database.Password = dbPass
	}
	if dbName := os.Getenv("DB_NAME"); dbName != "" {
		cfg.Database.DBName = dbName
	}
	if dbSSL := os.Getenv("DB_SSLMODE"); dbSSL != "" {
		cfg.Database.SSLMode = dbSSL
	}
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		cfg.Auth.JWTSecret = secret
	}
	if env := os.Getenv("APP_ENV"); env != "" {
		cfg.Environment = env
		if env == "production" {
			cfg.Debug = false
		}
	}

	fmt.Printf("[INFO] Configuration loaded for environment: %s\n", cfg.Environment)
	return cfg, nil
}
