package afk

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"
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

func TestMainFailStopFixedLoopsStopsAtFirstNonZeroExit(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Main([]string{"-n", "3", "--fail", "stop", "--", "sh", "-c", `printf "%s\n" "$AFK_INDEX"; exit 7`}, nil, &stdout, &stderr)
	if exitCode != 7 {
		t.Fatalf("Main() exit code = %d, want 7", exitCode)
	}
	if stdout.String() != "1\n" {
		t.Fatalf("Main() stdout = %q, want %q", stdout.String(), "1\\n")
	}
	if stderr.Len() != 0 {
		t.Fatalf("Main() stderr = %q, want empty", stderr.String())
	}
}

func TestMainFailStopDaemonStopsAtFirstNonZeroExit(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Main([]string{"--daemon", "--fail", "stop", "--", "sh", "-c", `printf "%s\n" "$AFK_INDEX"; exit 9`}, nil, &stdout, &stderr)
	if exitCode != 9 {
		t.Fatalf("Main() exit code = %d, want 9", exitCode)
	}
	if stdout.String() != "1\n" {
		t.Fatalf("Main() stdout = %q, want %q", stdout.String(), "1\\n")
	}
	if stderr.Len() != 0 {
		t.Fatalf("Main() stderr = %q, want empty", stderr.String())
	}
}

func TestMainDaemonRerunsAfterSuccessWithoutBuiltInDelay(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	start := time.Now()
	exitCode := Main([]string{"--daemon", "--fail", "stop", "--", "sh", "-c", `printf "%s\n" "$AFK_INDEX"; if [ "$AFK_INDEX" -eq 1 ]; then exit 0; fi; exit 7`}, nil, &stdout, &stderr)
	elapsed := time.Since(start)

	if exitCode != 7 {
		t.Fatalf("Main() exit code = %d, want 7", exitCode)
	}
	if stdout.String() != "1\n2\n" {
		t.Fatalf("Main() stdout = %q, want %q", stdout.String(), "1\\n2\\n")
	}
	if stderr.Len() != 0 {
		t.Fatalf("Main() stderr = %q, want empty", stderr.String())
	}
	if elapsed > 2*time.Second {
		t.Fatalf("Main() daemon non-item rerun took too long: elapsed=%v, want <= 2s (no built-in delay)", elapsed)
	}
}

func TestMainUntilSuccessFixedLoopsStopsAtFirstZeroExit(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Main([]string{"-n", "3", "--until-success", "--", "sh", "-c", `printf "%s\n" "$AFK_INDEX"; if [ "$AFK_INDEX" -lt 2 ]; then exit 7; fi; exit 0`}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("Main() exit code = %d, want 0", exitCode)
	}
	if stdout.String() != "1\n2\n" {
		t.Fatalf("Main() stdout = %q, want %q", stdout.String(), "1\\n2\\n")
	}
	if stderr.Len() != 0 {
		t.Fatalf("Main() stderr = %q, want empty", stderr.String())
	}
}

func TestMainUntilSuccessFixedLoopsReturnsLastNonZeroWhenAttemptsExhausted(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Main([]string{"-n", "3", "--until-success", "--", "sh", "-c", `printf "%s\n" "$AFK_INDEX"; exit $((40 + AFK_INDEX))`}, nil, &stdout, &stderr)
	if exitCode != 43 {
		t.Fatalf("Main() exit code = %d, want 43", exitCode)
	}
	if stdout.String() != "1\n2\n3\n" {
		t.Fatalf("Main() stdout = %q, want %q", stdout.String(), "1\\n2\\n3\\n")
	}
	if stderr.Len() != 0 {
		t.Fatalf("Main() stderr = %q, want empty", stderr.String())
	}
}

func TestMainUntilSuccessDaemonStopsAtFirstZeroExit(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	start := time.Now()
	exitCode := Main([]string{"--daemon", "--until-success", "--", "sh", "-c", `printf "%s\n" "$AFK_INDEX"; if [ "$AFK_INDEX" -lt 3 ]; then exit 9; fi; exit 0`}, nil, &stdout, &stderr)
	elapsed := time.Since(start)

	if exitCode != 0 {
		t.Fatalf("Main() exit code = %d, want 0", exitCode)
	}
	if stdout.String() != "1\n2\n3\n" {
		t.Fatalf("Main() stdout = %q, want %q", stdout.String(), "1\\n2\\n3\\n")
	}
	if stderr.Len() != 0 {
		t.Fatalf("Main() stderr = %q, want empty", stderr.String())
	}
	if elapsed > 2*time.Second {
		t.Fatalf("Main() daemon until-success took too long: elapsed=%v, want <= 2s", elapsed)
	}
}

func TestMainUntilSuccessTreatsSignaledMainChildAsFailureAndRetries(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Main([]string{"-n", "2", "--until-success", "--", "sh", "-c", `printf "%s\n" "$AFK_INDEX"; if [ "$AFK_INDEX" -eq 1 ]; then kill -TERM $$; fi; exit 0`}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("Main() exit code = %d, want 0", exitCode)
	}
	if stdout.String() != "1\n2\n" {
		t.Fatalf("Main() stdout = %q, want %q", stdout.String(), "1\\n2\\n")
	}
	if stderr.Len() != 0 {
		t.Fatalf("Main() stderr = %q, want empty", stderr.String())
	}
}

func TestMainUntilSuccessDaemonTreatsSignaledMainChildAsFailureAndRetries(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Main([]string{"--daemon", "--until-success", "--", "sh", "-c", `printf "%s\n" "$AFK_INDEX"; if [ "$AFK_INDEX" -eq 1 ]; then kill -TERM $$; fi; exit 0`}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("Main() exit code = %d, want 0", exitCode)
	}
	if stdout.String() != "1\n2\n" {
		t.Fatalf("Main() stdout = %q, want %q", stdout.String(), "1\\n2\\n")
	}
	if stderr.Len() != 0 {
		t.Fatalf("Main() stderr = %q, want empty", stderr.String())
	}
}

func TestMainFailStopMapsSignaledMainChildTo128PlusSignal(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Main([]string{"-n", "3", "--fail", "stop", "--", "sh", "-c", "kill -TERM $$"}, nil, &stdout, &stderr)
	if exitCode != 143 {
		t.Fatalf("Main() exit code = %d, want 143", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("Main() stdout = %q, want empty", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("Main() stderr = %q, want empty", stderr.String())
	}
}

func TestMainMissingMainChildCommandExits127Immediately(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Main([]string{"-n", "3", "--fail", "continue", "--", "definitely-not-a-real-command-afk"}, nil, &stdout, &stderr)
	if exitCode != 127 {
		t.Fatalf("Main() exit code = %d, want 127", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("Main() stdout = %q, want empty", stdout.String())
	}
	if stderr.Len() == 0 {
		t.Fatal("Main() wrote empty stderr for missing command")
	}
	if strings.Count(strings.TrimSpace(stderr.String()), "\n") != 0 {
		t.Fatalf("Main() stderr = %q, want single-line diagnostic", stderr.String())
	}
}

func TestMainNonExecutableMainChildExits126Immediately(t *testing.T) {
	tempDir := t.TempDir()
	path := tempDir + "/not-executable.sh"
	if err := os.WriteFile(path, []byte("#!/bin/sh\necho should-not-run\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Main([]string{"-n", "3", "--fail", "continue", "--", path}, nil, &stdout, &stderr)
	if exitCode != 126 {
		t.Fatalf("Main() exit code = %d, want 126", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("Main() stdout = %q, want empty", stdout.String())
	}
	if stderr.Len() == 0 {
		t.Fatal("Main() wrote empty stderr for non-executable command")
	}
	if strings.Count(strings.TrimSpace(stderr.String()), "\n") != 0 {
		t.Fatalf("Main() stderr = %q, want single-line diagnostic", stderr.String())
	}
}

func TestMainTimeoutMainChildReturnsSynthetic124AndWaitsForChildExit(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	start := time.Now()
	exitCode := Main([]string{"-n", "1", "--fail", "stop", "--timeout", "100ms", "--", "sh", "-c", `trap 'sleep 0.3; exit 0' TERM; while :; do sleep 1; done`}, nil, &stdout, &stderr)
	elapsed := time.Since(start)

	if exitCode != 124 {
		t.Fatalf("Main() exit code = %d, want 124", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("Main() stdout = %q, want empty", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("Main() stderr = %q, want empty", stderr.String())
	}
	if elapsed < 300*time.Millisecond {
		t.Fatalf("Main() returned too quickly after timeout: elapsed=%v, want >= 300ms to wait for child exit", elapsed)
	}
}

func TestMainTimeoutMainChildEscalatesToSigkillAfterGracePeriod(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	start := time.Now()
	exitCode := Main([]string{"-n", "1", "--fail", "stop", "--timeout", "100ms", "--", "sh", "-c", `trap '' TERM; while :; do sleep 1; done`}, nil, &stdout, &stderr)
	elapsed := time.Since(start)

	if exitCode != 124 {
		t.Fatalf("Main() exit code = %d, want 124", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("Main() stdout = %q, want empty", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("Main() stderr = %q, want empty", stderr.String())
	}
	if elapsed < 2*time.Second {
		t.Fatalf("Main() returned too quickly for SIGKILL timeout path: elapsed=%v, want >= 2s", elapsed)
	}
	if elapsed > 4*time.Second {
		t.Fatalf("Main() took too long for SIGKILL timeout path: elapsed=%v, want <= 4s", elapsed)
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
