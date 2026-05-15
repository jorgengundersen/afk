package afk

import (
	"bytes"
	"fmt"
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

func TestMainRunsStaticItemsInSourceOrderWithItemContextEnv(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Main([]string{"--items", "a\nb", "--", "sh", "-c", `echo "$AFK_INDEX $AFK_ITEM_INDEX/$AFK_ITEM_COUNT $AFK_ITEM"`}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("Main() exit code = %d, want 0", exitCode)
	}
	if stdout.String() != "1 0/2 a\n2 1/2 b\n" {
		t.Fatalf("Main() stdout = %q, want %q", stdout.String(), "1 0/2 a\\n2 1/2 b\\n")
	}
	if stderr.Len() != 0 {
		t.Fatalf("Main() stderr = %q, want empty", stderr.String())
	}
}

func TestMainStaticEmptyItemSourcesSkipMainChildAndExitZero(t *testing.T) {
	tests := []struct {
		name  string
		items string
	}{
		{name: "empty string", items: ""},
		{name: "whitespace-only newline text", items: "  \n\t"},
		{name: "empty JSON array", items: "[]"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			exitCode := Main([]string{"--items", tc.items, "--", "sh", "-c", "exit 77"}, nil, &stdout, &stderr)
			if exitCode != 0 {
				t.Fatalf("Main() exit code = %d, want 0", exitCode)
			}
			if stdout.Len() != 0 {
				t.Fatalf("Main() stdout = %q, want empty", stdout.String())
			}
			if stderr.Len() != 0 {
				t.Fatalf("Main() stderr = %q, want empty", stderr.String())
			}
		})
	}
}

func TestMainItemsCmdNonDaemonProcessesFirstNonEmptyBatchAndExitsZero(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Main([]string{"--items-cmd", "printf 'a\\nb\\n'", "--", "sh", "-c", `echo "$AFK_INDEX $AFK_ITEM_INDEX/$AFK_ITEM_COUNT $AFK_ITEM"`}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("Main() exit code = %d, want 0", exitCode)
	}
	if stdout.String() != "1 0/2 a\n2 1/2 b\n" {
		t.Fatalf("Main() stdout = %q, want %q", stdout.String(), "1 0/2 a\\n2 1/2 b\\n")
	}
	if stderr.Len() != 0 {
		t.Fatalf("Main() stderr = %q, want empty", stderr.String())
	}
}

func TestMainItemsCmdDaemonRepollsAfterExhaustedNonEmptyBatches(t *testing.T) {
	tempDir := t.TempDir()
	callsFile := tempDir + "/items-cmd-calls"
	itemsCmd := fmt.Sprintf(
		`n=0; if [ -f %q ]; then n=$(cat %q); fi; n=$((n+1)); printf "%%s\n" "$n" > %q; if [ "$n" -eq 1 ]; then printf "a\nb\n"; else printf "c\n"; fi`,
		callsFile,
		callsFile,
		callsFile,
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Main(
		[]string{"--daemon", "--items-cmd", itemsCmd, "--fail", "stop", "--", "sh", "-c", `printf "%s %s/%s %s\n" "$AFK_INDEX" "$AFK_ITEM_INDEX" "$AFK_ITEM_COUNT" "$AFK_ITEM"; if [ "$AFK_INDEX" -eq 3 ]; then exit 7; fi`},
		nil,
		&stdout,
		&stderr,
	)
	if exitCode != 7 {
		t.Fatalf("Main() exit code = %d, want 7", exitCode)
	}
	if stdout.String() != "1 0/2 a\n2 1/2 b\n3 0/1 c\n" {
		t.Fatalf("Main() stdout = %q, want %q", stdout.String(), "1 0/2 a\\n2 1/2 b\\n3 0/1 c\\n")
	}
	if stderr.Len() != 0 {
		t.Fatalf("Main() stderr = %q, want empty", stderr.String())
	}

	calls, err := os.ReadFile(callsFile)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if string(calls) != "2\n" {
		t.Fatalf("items-cmd invocation count = %q, want %q", string(calls), "2\\n")
	}
}

func TestMainItemsCmdStdoutIsCapturedAndStderrPassesThrough(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Main([]string{"--items-cmd", "printf 'item-from-source\\n'; printf 'source-stderr\\n' >&2", "--", "sh", "-c", `printf "main:%s\n" "$AFK_ITEM"`}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("Main() exit code = %d, want 0", exitCode)
	}
	if stdout.String() != "main:item-from-source\n" {
		t.Fatalf("Main() stdout = %q, want %q", stdout.String(), "main:item-from-source\\n")
	}
	if stderr.String() != "source-stderr\n" {
		t.Fatalf("Main() stderr = %q, want %q", stderr.String(), "source-stderr\\n")
	}
}

func TestMainItemsCmdDoesNotInheritLoopContextEnv(t *testing.T) {
	t.Setenv("AFK_INDEX", "99")
	t.Setenv("AFK_ITEM", "stale")
	t.Setenv("AFK_ITEM_INDEX", "8")
	t.Setenv("AFK_ITEM_COUNT", "9")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Main([]string{"--items-cmd", `printf "%s|%s|%s|%s\n" "${AFK_INDEX-unset}" "${AFK_ITEM-unset}" "${AFK_ITEM_INDEX-unset}" "${AFK_ITEM_COUNT-unset}"`, "--", "sh", "-c", `printf "%s\n" "$AFK_ITEM"`}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("Main() exit code = %d, want 0", exitCode)
	}
	if stdout.String() != "unset|unset|unset|unset\n" {
		t.Fatalf("Main() stdout = %q, want %q", stdout.String(), "unset|unset|unset|unset\\n")
	}
	if stderr.Len() != 0 {
		t.Fatalf("Main() stderr = %q, want empty", stderr.String())
	}
}

func TestMainItemsCmdDoesNotConsumeInheritedParentStdin(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Main(
		[]string{"--items-cmd", `if IFS= read -r line; then printf "saw:%s\n" "$line"; else printf "empty\n"; fi`, "--", "sh", "-c", `printf "%s\n" "$AFK_ITEM"`},
		strings.NewReader("from-parent-stdin\n"),
		&stdout,
		&stderr,
	)
	if exitCode != 0 {
		t.Fatalf("Main() exit code = %d, want 0", exitCode)
	}
	if stdout.String() != "empty\n" {
		t.Fatalf("Main() stdout = %q, want %q", stdout.String(), "empty\\n")
	}
	if stderr.Len() != 0 {
		t.Fatalf("Main() stderr = %q, want empty", stderr.String())
	}
}

func TestMainTimeoutItemsCmdKillsProcessGroupDiscardsCapturedStdoutAndExits1(t *testing.T) {
	tempDir := t.TempDir()
	terminatedFile := tempDir + "/items-cmd-term"

	itemsCmd := fmt.Sprintf(
		`printf "ghost-item\n"; trap '' TERM; (trap 'printf terminated > %s; exit 0' TERM; while :; do sleep 1; done) & while :; do sleep 1; done`,
		terminatedFile,
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	start := time.Now()
	exitCode := Main([]string{"--items-cmd", itemsCmd, "--timeout", "100ms", "--", "sh", "-c", `printf "main:%s\n" "$AFK_ITEM"`}, nil, &stdout, &stderr)
	elapsed := time.Since(start)

	if exitCode != 1 {
		t.Fatalf("Main() exit code = %d, want 1", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("Main() stdout = %q, want empty because timed-out items-cmd stdout must be discarded", stdout.String())
	}
	if _, err := os.Stat(terminatedFile); err != nil {
		t.Fatalf("expected timed-out items-cmd process group to receive SIGTERM: %v", err)
	}
	if elapsed < 2*time.Second {
		t.Fatalf("Main() returned too quickly for items-cmd SIGKILL timeout path: elapsed=%v, want >= 2s", elapsed)
	}
	if elapsed > 4*time.Second {
		t.Fatalf("Main() took too long for items-cmd timeout path: elapsed=%v, want <= 4s", elapsed)
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
