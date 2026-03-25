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
	path string
	file *os.File
}

// New creates a logger that writes to the given path.
// The file is created on first Log call (lazy open).
func New(path string) *Logger {
	return &Logger{path: path}
}

// Log writes a single event with the given name and fields.
// Silent on failure — never returns error, never panics.
func (l *Logger) Log(event string, fields map[string]string) {
	if l.file == nil {
		dir := filepath.Dir(l.path)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return
		}
		f, err := os.OpenFile(l.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return
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
			v := fields[k]
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
	_, _ = l.file.WriteString(b.String())
}

// Close flushes and closes the underlying file.
func (l *Logger) Close() error {
	if l.file == nil {
		return nil
	}
	return l.file.Close()
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
