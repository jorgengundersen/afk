package afk

import (
	"bytes"
	"strings"
	"testing"
)

func TestMainHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Main([]string{"--help"}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("Main() exit code = %d, want 0", exitCode)
	}

	got := stdout.String()
	if got == "" {
		t.Fatal("Main() wrote empty help output")
	}
	if !strings.Contains(got, "Usage: afk [flags] -- command [args...]") {
		t.Fatalf("Main() help output missing usage line:\n%s", got)
	}

	if stderr.Len() != 0 {
		t.Fatalf("Main() wrote to stderr for help: %q", stderr.String())
	}
}
