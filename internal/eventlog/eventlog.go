package eventlog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Logger writes structured key=value log events to a file.
type Logger struct {
	mu     sync.Mutex
	file   *os.File
	stderr bool
}

// Field is a key-value pair for structured log events.
type Field struct {
	Key   string
	Value string
}

// New creates a Logger that writes to a new log file in logDir.
func New(logDir string, stderr bool) (*Logger, error) {
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}
	name := fmt.Sprintf("afk-%s.log", time.Now().UTC().Format("20060102-150405"))
	path := filepath.Join(logDir, name)
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create log file: %w", err)
	}
	return &Logger{file: f, stderr: stderr}, nil
}

// Event writes one structured log line.
func (l *Logger) Event(name string, fields ...Field) {
	var b strings.Builder
	b.WriteString(time.Now().UTC().Format(time.RFC3339))
	b.WriteString(" [afk] ")
	b.WriteString(name)
	for _, f := range fields {
		b.WriteByte(' ')
		b.WriteString(f.Key)
		b.WriteByte('=')
		if strings.ContainsAny(f.Value, " \t\"") {
			fmt.Fprintf(&b, "%q", f.Value)
		} else {
			b.WriteString(f.Value)
		}
	}
	b.WriteByte('\n')
	line := b.String()
	l.mu.Lock()
	fmt.Fprint(l.file, line)
	if l.stderr {
		fmt.Fprint(os.Stderr, line)
	}
	l.mu.Unlock()
}

// Close flushes and closes the log file.
func (l *Logger) Close() error {
	return l.file.Close()
}

// F is a convenience constructor for Field.
func F(key, value string) Field {
	return Field{Key: key, Value: value}
}
