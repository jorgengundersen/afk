package logger

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestSessionPathCreatesDir(t *testing.T) {
	// Use a temp dir as the base to avoid polluting the real home dir.
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)

	path, err := SessionPath()
	if err != nil {
		t.Fatalf("SessionPath() error: %v", err)
	}

	// Path should match pattern: <base>/afk/logs/afk-<timestamp>.log
	re := regexp.MustCompile(`afk/logs/afk-\d{8}-\d{6}\.log$`)
	if !re.MatchString(path) {
		t.Fatalf("path %q does not match expected pattern", path)
	}

	// Directory should have been created.
	dir := filepath.Dir(path)
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("log directory was not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("expected %q to be a directory", dir)
	}
}
