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

func TestMainRunsFixedLoopMainChildAndPreservesIOContract(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		stdin    string
		setupEnv func(t *testing.T)
		wantOut  string
		wantErr  string
		wantCode int
	}{
		{
			name:     "sets AFK_INDEX for each fixed loop invocation",
			args:     []string{"-n", "3", "--", "sh", "-c", `printf "%s\n" "$AFK_INDEX"`},
			wantOut:  "1\n2\n3\n",
			wantCode: 0,
		},
		{
			name:     "executes main child argv directly without shell expansion",
			args:     []string{"-n", "1", "--", "printf", "%s\n", "$HOME"},
			wantOut:  "$HOME\n",
			wantCode: 0,
		},
		{
			name:     "inherits parent stdin for main child invocation",
			args:     []string{"-n", "1", "--", "sh", "-c", "cat"},
			stdin:    "hello\n",
			wantOut:  "hello\n",
			wantCode: 0,
		},
		{
			name:     "passes through main child stdout and stderr on original streams",
			args:     []string{"-n", "1", "--", "sh", "-c", `printf "out\n"; printf "err\n" >&2`},
			wantOut:  "out\n",
			wantErr:  "err\n",
			wantCode: 0,
		},
		{
			name: "removes inherited AFK_ITEM context in non-item loops",
			args: []string{"-n", "1", "--", "sh", "-c", `printf "%s|%s|%s\n" "${AFK_ITEM-unset}" "${AFK_ITEM_INDEX-unset}" "${AFK_ITEM_COUNT-unset}"`},
			setupEnv: func(t *testing.T) {
				t.Setenv("AFK_ITEM", "stale")
				t.Setenv("AFK_ITEM_INDEX", "9")
				t.Setenv("AFK_ITEM_COUNT", "10")
			},
			wantOut:  "unset|unset|unset\n",
			wantCode: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setupEnv != nil {
				tc.setupEnv(t)
			}

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			stdin := strings.NewReader(tc.stdin)

			exitCode := Main(tc.args, stdin, &stdout, &stderr)
			if exitCode != tc.wantCode {
				t.Fatalf("Main() exit code = %d, want %d", exitCode, tc.wantCode)
			}
			if stdout.String() != tc.wantOut {
				t.Fatalf("Main() stdout = %q, want %q", stdout.String(), tc.wantOut)
			}
			if stderr.String() != tc.wantErr {
				t.Fatalf("Main() stderr = %q, want %q", stderr.String(), tc.wantErr)
			}
		})
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
