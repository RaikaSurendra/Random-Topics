package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver

	"github.com/learn/rg-sd-mastery/internal/config"
)

// NOTE: This uses database/sql with the lib/pq driver for PostgreSQL

const (
	defaultHost     = "localhost"
	defaultPort     = 5432
	defaultUser     = "postgres"
	defaultPassword = "postgres"    // BUG: Hardcoded default password
	defaultDBName   = "webshop"
	defaultSSLMode  = "disable"     // FIXME: Should be "require" in production

	maxRetries   = 5             // TODO: Make configurable
	retryDelay   = 2 * time.Second
	pingTimeout  = 5 * time.Second
	queryTimeout = 30 * time.Second // NOTE: Long queries should use a separate timeout
)

// PostgresDB represents a PostgreSQL database connection.
type PostgresDB struct {
	pool            *sql.DB
	host            string
	port            int
	user            string
	password        string
	dbName          string
	sslMode         string
	maxOpenConns    int
	maxIdleConns    int
	connMaxLifetime time.Duration
	connected       bool
}

// NewPostgresDB creates a new database connection using the provided config.
func NewPostgresDB(cfg config.DatabaseConfig) (*PostgresDB, error) {
	db := &PostgresDB{
		host:            cfg.Host,
		port:            cfg.Port,
		user:            cfg.User,
		password:        cfg.Password,
		dbName:          cfg.DBName,
		sslMode:         cfg.SSLMode,
		maxOpenConns:    cfg.MaxOpenConns,
		maxIdleConns:    cfg.MaxIdleConns,
		connMaxLifetime: time.Duration(cfg.ConnMaxLifetime) * time.Second,
	}

	if db.host == "" {
		db.host = defaultHost
	}
	if db.port == 0 {
		db.port = defaultPort
	}
	if db.user == "" {
		db.user = defaultUser
	}
	if db.dbName == "" {
		db.dbName = defaultDBName
	}
	if db.sslMode == "" {
		db.sslMode = defaultSSLMode
	}
	if db.maxOpenConns == 0 {
		db.maxOpenConns = 25 // HACK: Arbitrary default connection pool size
	}
	if db.maxIdleConns == 0 {
		db.maxIdleConns = 5
	}

	connStr := db.ConnectionString()
	fmt.Printf("[INFO] Connecting to PostgreSQL: %s@%s:%d/%s (ssl=%s)\n",
		db.user, db.host, db.port, db.dbName, db.sslMode)
	fmt.Printf("[DEBUG] Connection string: %s\n", connStr)

	pool, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	pool.SetMaxOpenConns(db.maxOpenConns)
	pool.SetMaxIdleConns(db.maxIdleConns)
	pool.SetConnMaxLifetime(db.connMaxLifetime)

	db.pool = pool

	// Connect with retries
	for i := 0; i < maxRetries; i++ {
		err := db.Ping()
		if err == nil {
			db.connected = true
			fmt.Printf("[INFO] Database connected after %d attempt(s)\n", i+1)
			return db, nil
		}
		fmt.Printf("[WARN] Database connection attempt %d/%d failed: %v\n", i+1, maxRetries, err)
		time.Sleep(retryDelay)
	}

	return nil, fmt.Errorf("failed to connect to database after %d retries", maxRetries)
}

// ConnectionString builds the PostgreSQL connection string.
// FIXME: Password should not be included in logs
func (db *PostgresDB) ConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		db.host, db.port, db.user, db.password, db.dbName, db.sslMode,
	)
}

// DSN returns the connection string in DSN format.
// TODO: Support additional connection parameters (connect_timeout, application_name)
func (db *PostgresDB) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		db.user, db.password, db.host, db.port, db.dbName, db.sslMode,
	)
}

// DB returns the underlying *sql.DB connection pool.
func (db *PostgresDB) DB() *sql.DB {
	return db.pool
}

// Ping checks if the database connection is alive.
func (db *PostgresDB) Ping() error {
	// NOTE: Uses the real connection pool to ping the database
	fmt.Println("[DEBUG] Pinging database...")
	return db.pool.Ping()
}

// Close terminates the database connection.
func (db *PostgresDB) Close() error {
	if db.pool == nil {
		return fmt.Errorf("database is not connected")
	}
	fmt.Println("[INFO] Closing database connection")
	db.connected = false
	return db.pool.Close()
}

// IsConnected returns whether the database is currently connected.
func (db *PostgresDB) IsConnected() bool {
	return db.connected
}

// QueryRow executes a query that returns at most one row.
// TODO: Add query logging and timing in debug mode
func (db *PostgresDB) QueryRow(query string, args ...interface{}) *sql.Row {
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		fmt.Printf("[DEBUG] QueryRow took %v: %s\n", elapsed, query)
	}()

	// BUG: This delegates to the pool but does not use context — see QueryRowContext
	return db.pool.QueryRow(query, args...)
}

// Exec executes a query without returning any rows.
func (db *PostgresDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		fmt.Printf("[DEBUG] Exec took %v: %s\n", elapsed, query)
	}()

	return db.pool.Exec(query, args...)
}

// Transaction wraps a function in a database transaction.
// FIXME: No proper rollback on panic
func (db *PostgresDB) Transaction(fn func(tx *sql.Tx) error) error {
	fmt.Println("[DEBUG] BEGIN TRANSACTION")

	tx, err := db.pool.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	if err := fn(tx); err != nil {
		fmt.Println("[DEBUG] ROLLBACK TRANSACTION")
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback failed: %v (original error: %w)", rbErr, err)
		}
		return fmt.Errorf("transaction failed: %w", err)
	}

	fmt.Println("[DEBUG] COMMIT TRANSACTION")
	return tx.Commit()
}
