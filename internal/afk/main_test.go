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

func TestMainUsageErrorsExit2(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "missing separator", args: []string{"--loops", "1", "echo"}},
		{name: "unknown flag", args: []string{"--loops", "1", "--bogus", "--", "echo"}},
		{name: "missing child command", args: []string{"--loops", "1", "--"}},
		{name: "invalid flag combination", args: []string{"--loops", "1", "--daemon", "--", "echo"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			exitCode := Main(tc.args, nil, &stdout, &stderr)
			if exitCode != 2 {
				t.Fatalf("Main() exit code = %d, want 2", exitCode)
			}
			if stdout.Len() != 0 {
				t.Fatalf("Main() wrote to stdout on usage error: %q", stdout.String())
			}
			if stderr.Len() == 0 {
				t.Fatal("Main() wrote empty stderr on usage error")
			}
		})
	}
}
