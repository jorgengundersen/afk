package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Logger writes structured events to a log file.
type Logger struct {
	path   string
	file   *os.File
	closed bool
}

// New creates a logger that writes to the given path.
// The file is created on first Log call (lazy open).
func New(path string) *Logger {
	return &Logger{path: path}
}

// Log writes a single event with the given name and fields.
// Returns an error if the file cannot be created or the write fails.
// Log after Close is a no-op and returns nil.
func (l *Logger) Log(event string, fields map[string]any) error {
	if l.closed {
		return nil
	}
	if l.file == nil {
		dir := filepath.Dir(l.path)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create log directory: %w", err)
		}
		f, err := os.OpenFile(l.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return fmt.Errorf("open log file: %w", err)
		}
		l.file = f
	}

	var b strings.Builder
	b.WriteString(time.Now().UTC().Format(time.RFC3339))
	b.WriteString(" [afk] ")
	b.WriteString(event)

	if len(fields) > 0 {
		keys := make([]string, 0, len(fields))
		for k := range fields {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v := fmt.Sprintf("%v", fields[k])
			if strings.Contains(v, " ") {
				v = fmt.Sprintf("%q", v)
			}
			b.WriteString(" ")
			b.WriteString(k)
			b.WriteString("=")
			b.WriteString(v)
		}
	}

	b.WriteString("\n")
	if _, err := l.file.WriteString(b.String()); err != nil {
		return fmt.Errorf("write log event: %w", err)
	}
	return nil
}

// Close flushes and closes the underlying file.
func (l *Logger) Close() error {
	if l.file == nil {
		return nil
	}
	err := l.file.Close()
	l.file = nil
	l.closed = true
	return err
}

// SessionPath returns the log file path for a new session.
// Creates the log directory if needed.
func SessionPath() (string, error) {
	base, err := dataHome()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(base, "afk", "logs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	filename := "afk-" + time.Now().Format("20060102-150405") + ".log"
	return filepath.Join(dir, filename), nil
}

func dataHome() (string, error) {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return xdg, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share"), nil
}
