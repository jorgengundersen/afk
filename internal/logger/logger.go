package logger

import (
	"os"
	"path/filepath"
	"time"
)

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
