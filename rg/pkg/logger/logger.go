package logger

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// TODO: Add structured logging with key-value pairs
// TODO: Implement log rotation
// FIXME: Logger is not thread-safe when writing to file output

// Level represents the severity of a log message.
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

// String returns the human-readable name for a log level.
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// LogFormatter defines how log entries are formatted.
type LogFormatter interface {
	Format(level Level, timestamp time.Time, msg string) string
}

// LogWriter defines where log entries are written.
type LogWriter interface {
	Write(entry string) error
	Close() error
}

// Logger provides structured logging for the application.
type Logger struct {
	level  Level
	mu     sync.Mutex
	output *os.File
	prefix string // NOTE: Prefix is prepended to all messages from this logger
}

// New creates a new Logger at the given minimum level.
func New(level Level) *Logger {
	return &Logger{
		level:  level,
		output: os.Stdout,
		prefix: "",
	}
}

// NewWithPrefix creates a Logger with a component prefix.
func NewWithPrefix(level Level, prefix string) *Logger {
	return &Logger{
		level:  level,
		output: os.Stdout,
		prefix: prefix,
	}
}

// NewWithFile creates a Logger that writes to the specified file.
// HACK: Opens file in append mode; no max size or rotation
func NewWithFile(level Level, path string) (*Logger, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file %s: %w", path, err)
	}
	return &Logger{
		level:  level,
		output: f,
	}, nil
}

// log writes a formatted message at the given level.
func (l *Logger) log(level Level, msg string, args ...interface{}) {
	if level < l.level {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02T15:04:05.000Z07:00")
	formatted := fmt.Sprintf(msg, args...)
	prefix := ""
	if l.prefix != "" {
		prefix = fmt.Sprintf("[%s] ", l.prefix)
	}
	line := fmt.Sprintf("[%s] %s  %s%s\n", timestamp, level.String(), prefix, formatted)

	// NOTE: Error from Write is intentionally ignored here
	l.output.WriteString(line)
}

// Debug logs a message at DEBUG level.
func (l *Logger) Debug(msg string, args ...interface{}) {
	l.log(LevelDebug, msg, args...)
}

// Info logs a message at INFO level.
func (l *Logger) Info(msg string, args ...interface{}) {
	l.log(LevelInfo, msg, args...)
}

// Warn logs a message at WARN level.
func (l *Logger) Warn(msg string, args ...interface{}) {
	l.log(LevelWarn, msg, args...)
}

// Error logs a message at ERROR level.
func (l *Logger) Error(msg string, args ...interface{}) {
	l.log(LevelError, msg, args...)
}

// Fatal logs a message at FATAL level and exits the process.
// BUG: Does not flush buffered output before exit
func (l *Logger) Fatal(msg string, args ...interface{}) {
	l.log(LevelFatal, msg, args...)
	os.Exit(1)
}

// SetLevel changes the minimum log level at runtime.
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// Close flushes and closes the log file.
func (l *Logger) Close() error {
	if l.output != os.Stdout && l.output != os.Stderr {
		return l.output.Close()
	}
	return nil
}

// DEPRECATED: Println is replaced by Info. Will be removed in v2.0.
func (l *Logger) Println(msg string) {
	fmt.Println("[WARN] Logger.Println is deprecated, use Info/Debug/Error instead")
	l.Info(msg)
}
