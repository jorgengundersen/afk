package afk

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"
)

func assertTimeoutDiagnostics(t *testing.T, stderr string, role string, duration string, wantCount int) {
	t.Helper()

	if got := strings.Count(strings.ToLower(stderr), "timeout"); got < wantCount {
		t.Fatalf("stderr = %q, want at least %d timeout diagnostic(s), got %d", stderr, wantCount, got)
	}
	if got := strings.Count(stderr, role); got < wantCount {
		t.Fatalf("stderr = %q, want at least %d %q timeout role mention(s), got %d", stderr, wantCount, role, got)
	}
	if got := strings.Count(stderr, duration); got < wantCount {
		t.Fatalf("stderr = %q, want at least %d configured timeout duration mention(s) %q, got %d", stderr, wantCount, duration, got)
	}
}

func assertInterruptDiagnostics(t *testing.T, stderr string, wantEscalation bool) {
	t.Helper()

	lower := strings.ToLower(stderr)
	if !strings.Contains(lower, "interrupt") {
		t.Fatalf("stderr = %q, want interrupt diagnostic", stderr)
	}
	if !strings.Contains(lower, "130") {
		t.Fatalf("stderr = %q, want interrupt exit semantics mention (130)", stderr)
	}

	hasEscalation := strings.Contains(lower, "second") && strings.Contains(lower, "forc")
	if wantEscalation && !hasEscalation {
		t.Fatalf("stderr = %q, want second-interrupt escalation diagnostic", stderr)
	}
	if !wantEscalation && hasEscalation {
		t.Fatalf("stderr = %q, got unexpected second-interrupt escalation diagnostic", stderr)
	}
}

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

	for _, want := range []string{
		"-h, --help",
		"-n, --loops",
		"-d, --daemon",
		"--items",
		"--items-cmd",
		"--sleep",
		"--empty-sleeps",
		"--fail",
		"--until-success",
		"--timeout",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("Main() help output missing %q:\n%s", want, got)
		}
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

func TestMainItemsCmdDaemonSleepsAfterEmptyBatchBeforeRepoll(t *testing.T) {
	tempDir := t.TempDir()
	callsFile := tempDir + "/items-cmd-calls"
	itemsCmd := fmt.Sprintf(
		`n=0; if [ -f %q ]; then n=$(cat %q); fi; n=$((n+1)); printf "%%s\n" "$n" > %q; if [ "$n" -eq 2 ]; then printf "c\n"; fi`,
		callsFile,
		callsFile,
		callsFile,
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	startedAt := time.Now()
	exitCode := Main(
		[]string{"--daemon", "--items-cmd", itemsCmd, "--sleep", "150ms", "--fail", "stop", "--", "sh", "-c", `printf "%s\n" "$AFK_INDEX"; exit 7`},
		nil,
		&stdout,
		&stderr,
	)
	elapsed := time.Since(startedAt)

	if exitCode != 7 {
		t.Fatalf("Main() exit code = %d, want 7", exitCode)
	}
	if stdout.String() != "1\n" {
		t.Fatalf("Main() stdout = %q, want %q", stdout.String(), "1\\n")
	}
	if stderr.Len() != 0 {
		t.Fatalf("Main() stderr = %q, want empty", stderr.String())
	}
	if elapsed < 120*time.Millisecond {
		t.Fatalf("Main() elapsed = %v, want >= 120ms due to empty-batch daemon sleep", elapsed)
	}

	calls, err := os.ReadFile(callsFile)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if string(calls) != "2\n" {
		t.Fatalf("items-cmd invocation count = %q, want %q", string(calls), "2\\n")
	}
}

func TestMainItemsCmdNonDaemonRetriesEmptyBatchesWithinEmptySleepsQuota(t *testing.T) {
	tempDir := t.TempDir()
	callsFile := tempDir + "/items-cmd-calls"
	itemsCmd := fmt.Sprintf(
		`n=0; if [ -f %q ]; then n=$(cat %q); fi; n=$((n+1)); printf "%%s\n" "$n" > %q; if [ "$n" -eq 2 ]; then printf "a\n"; fi`,
		callsFile,
		callsFile,
		callsFile,
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	startedAt := time.Now()
	exitCode := Main(
		[]string{"--items-cmd", itemsCmd, "--empty-sleeps", "1", "--sleep", "100ms", "--", "sh", "-c", `printf "%s %s\n" "$AFK_INDEX" "$AFK_ITEM"`},
		nil,
		&stdout,
		&stderr,
	)
	elapsed := time.Since(startedAt)

	if exitCode != 0 {
		t.Fatalf("Main() exit code = %d, want 0", exitCode)
	}
	if stdout.String() != "1 a\n" {
		t.Fatalf("Main() stdout = %q, want %q", stdout.String(), "1 a\\n")
	}
	if stderr.Len() != 0 {
		t.Fatalf("Main() stderr = %q, want empty", stderr.String())
	}
	if elapsed < 80*time.Millisecond {
		t.Fatalf("Main() elapsed = %v, want >= 80ms due to one empty-batch sleep", elapsed)
	}

	calls, err := os.ReadFile(callsFile)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if string(calls) != "2\n" {
		t.Fatalf("items-cmd invocation count = %q, want %q", string(calls), "2\\n")
	}
}

func TestMainItemsCmdNonDaemonExitsZeroWhenEmptySleepsQuotaIsExhausted(t *testing.T) {
	tempDir := t.TempDir()
	callsFile := tempDir + "/items-cmd-calls"
	itemsCmd := fmt.Sprintf(
		`n=0; if [ -f %q ]; then n=$(cat %q); fi; n=$((n+1)); printf "%%s\n" "$n" > %q`,
		callsFile,
		callsFile,
		callsFile,
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	startedAt := time.Now()
	exitCode := Main(
		[]string{"--items-cmd", itemsCmd, "--empty-sleeps", "2", "--sleep", "50ms", "--", "sh", "-c", `printf "%s\n" "$AFK_ITEM"`},
		nil,
		&stdout,
		&stderr,
	)
	elapsed := time.Since(startedAt)

	if exitCode != 0 {
		t.Fatalf("Main() exit code = %d, want 0", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("Main() stdout = %q, want empty", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("Main() stderr = %q, want empty", stderr.String())
	}
	if elapsed < 90*time.Millisecond {
		t.Fatalf("Main() elapsed = %v, want >= 90ms due to two empty-batch sleeps", elapsed)
	}

	calls, err := os.ReadFile(callsFile)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if string(calls) != "3\n" {
		t.Fatalf("items-cmd invocation count = %q, want %q", string(calls), "3\\n")
	}
}

func TestMainItemsCmdNonDaemonWithZeroEmptySleepsExitsImmediatelyAfterFirstEmptyBatch(t *testing.T) {
	tempDir := t.TempDir()
	callsFile := tempDir + "/items-cmd-calls"
	itemsCmd := fmt.Sprintf(
		`n=0; if [ -f %q ]; then n=$(cat %q); fi; n=$((n+1)); printf "%%s\n" "$n" > %q`,
		callsFile,
		callsFile,
		callsFile,
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	startedAt := time.Now()
	exitCode := Main(
		[]string{"--items-cmd", itemsCmd, "--empty-sleeps", "0", "--sleep", "200ms", "--", "sh", "-c", `printf "%s\n" "$AFK_ITEM"`},
		nil,
		&stdout,
		&stderr,
	)
	elapsed := time.Since(startedAt)

	if exitCode != 0 {
		t.Fatalf("Main() exit code = %d, want 0", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("Main() stdout = %q, want empty", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("Main() stderr = %q, want empty", stderr.String())
	}
	if elapsed >= 120*time.Millisecond {
		t.Fatalf("Main() elapsed = %v, want < 120ms because zero empty-sleeps must not sleep", elapsed)
	}

	calls, err := os.ReadFile(callsFile)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if string(calls) != "1\n" {
		t.Fatalf("items-cmd invocation count = %q, want %q", string(calls), "1\\n")
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

func TestMainItemsCmdStdoutLimitExits1AndDoesNotParsePartialBatch(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	oversizedBytes := itemsCommandStdoutLimitBytes + 1
	itemsCmd := fmt.Sprintf(`printf 'first-item\n'; head -c %d /dev/zero | tr '\000' a`, oversizedBytes)

	exitCode := Main(
		[]string{"--items-cmd", itemsCmd, "--", "sh", "-c", `printf "main:%s\n" "$AFK_ITEM"`},
		nil,
		&stdout,
		&stderr,
	)
	if exitCode != 1 {
		t.Fatalf("Main() exit code = %d, want 1", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("Main() stdout = %q, want empty because oversized items-cmd stdout must not be parsed", stdout.String())
	}
	lowerStderr := strings.ToLower(stderr.String())
	for _, want := range []string{"item source error", "--items-cmd", "stdout", "limit", "exceeded"} {
		if !strings.Contains(lowerStderr, want) {
			t.Fatalf("Main() stderr = %q, want oversized stdout diagnostic containing %q", stderr.String(), want)
		}
	}
}

func TestMainItemsCmdNonZeroExitDiscardsCapturedStdoutAndExits1(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Main(
		[]string{"--items-cmd", `printf "ghost-item\n"; printf "source-stderr\n" >&2; exit 3`, "--", "sh", "-c", `printf "main:%s\n" "$AFK_ITEM"`},
		nil,
		&stdout,
		&stderr,
	)
	if exitCode != 1 {
		t.Fatalf("Main() exit code = %d, want 1", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("Main() stdout = %q, want empty because non-zero items-cmd stdout must be discarded", stdout.String())
	}
	if !strings.Contains(stderr.String(), "source-stderr\n") {
		t.Fatalf("Main() stderr = %q, want source stderr passthrough", stderr.String())
	}
	if !strings.Contains(stderr.String(), "item source error: exit status 3\n") {
		t.Fatalf("Main() stderr = %q, want source error diagnostic", stderr.String())
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
	assertTimeoutDiagnostics(t, stderr.String(), "--items-cmd", "100ms", 1)
	if !strings.Contains(stderr.String(), "item source error: source command timed out") {
		t.Fatalf("Main() stderr = %q, want source timeout error diagnostic", stderr.String())
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

func TestMainDefaultFailContinueFixedLoopsCompletesAfterNonZeroExitsAndReturnsZero(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Main([]string{"-n", "3", "--", "sh", "-c", `printf "%s\n" "$AFK_INDEX"; exit 7`}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("Main() exit code = %d, want 0", exitCode)
	}
	if stdout.String() != "1\n2\n3\n" {
		t.Fatalf("Main() stdout = %q, want %q", stdout.String(), "1\\n2\\n3\\n")
	}
	if stderr.Len() != 0 {
		t.Fatalf("Main() stderr = %q, want empty", stderr.String())
	}
}

func TestMainDefaultFailContinueFixedLoopsTreatsSignaledMainChildAsFailureAndContinues(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Main([]string{"-n", "2", "--", "sh", "-c", `printf "%s\n" "$AFK_INDEX"; kill -INT $$`}, nil, &stdout, &stderr)
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

func TestRunLoopDefaultFailContinueDaemonKeepsRerunningSignaledMainChildUntilInterrupted(t *testing.T) {
	tempDir := t.TempDir()
	iterationsFile := tempDir + "/iterations.txt"
	cmd := fmt.Sprintf(`printf "%%s\\n" "$AFK_INDEX" >> "%s"; kill -INT $$`, iterationsFile)

	cfg, err := ParseArgs([]string{"--daemon", "--", "sh", "-c", cmd})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}
	if err := ValidateConfig(cfg); err != nil {
		t.Fatalf("ValidateConfig() error = %v", err)
	}

	interrupts := make(chan os.Signal, 1)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	done := make(chan int, 1)
	go func() {
		done <- runLoopWithInterrupts(cfg, nil, &stdout, &stderr, interrupts)
	}()

	deadline := time.Now().Add(2 * time.Second)
	for {
		data, readErr := os.ReadFile(iterationsFile)
		if readErr == nil && strings.HasPrefix(string(data), "1\n2\n") {
			break
		}
		if time.Now().After(deadline) {
			interrupts <- syscall.SIGINT
			select {
			case <-done:
			case <-time.After(1 * time.Second):
			}
			t.Fatalf("daemon did not schedule a second invocation after signaled failure; file contents: %q", string(data))
		}
		time.Sleep(10 * time.Millisecond)
	}

	interrupts <- syscall.SIGINT
	select {
	case exitCode := <-done:
		if exitCode != 130 {
			t.Fatalf("runLoopWithInterrupts() exit code = %d, want 130", exitCode)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("runLoopWithInterrupts() did not exit after interrupt")
	}

	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	assertInterruptDiagnostics(t, stderr.String(), false)
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
	assertTimeoutDiagnostics(t, stderr.String(), "main child", "100ms", 1)
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
	assertTimeoutDiagnostics(t, stderr.String(), "main child", "100ms", 1)
	if elapsed < 2*time.Second {
		t.Fatalf("Main() returned too quickly for SIGKILL timeout path: elapsed=%v, want >= 2s", elapsed)
	}
	if elapsed > 4*time.Second {
		t.Fatalf("Main() took too long for SIGKILL timeout path: elapsed=%v, want <= 4s", elapsed)
	}
}

func TestMainTimeoutCleansUpGrandchildrenInProcessGroup(t *testing.T) {
	tempDir := t.TempDir()
	terminatedFile := tempDir + "/timeout-term"

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Main(
		[]string{"-n", "1", "--fail", "stop", "--timeout", "100ms", "--", "sh", "-c", fmt.Sprintf(`trap '' TERM; (trap 'printf terminated > %s; exit 0' TERM; while :; do sleep 1; done) & while :; do sleep 1; done`, terminatedFile)},
		nil,
		&stdout,
		&stderr,
	)
	if exitCode != 124 {
		t.Fatalf("Main() exit code = %d, want 124", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("Main() stdout = %q, want empty", stdout.String())
	}
	assertTimeoutDiagnostics(t, stderr.String(), "main child", "100ms", 1)
	if _, err := os.Stat(terminatedFile); err != nil {
		t.Fatalf("expected timeout cleanup to terminate grandchild in spawned process group: %v", err)
	}
}

func TestMainTimeoutUnderDefaultContinueFixedLoopsCompletesAttemptsAndExitsZero(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Main(
		[]string{"-n", "3", "--timeout", "100ms", "--", "sh", "-c", `printf "%s\n" "$AFK_INDEX"; trap 'exit 0' TERM; while :; do sleep 1; done`},
		nil,
		&stdout,
		&stderr,
	)
	if exitCode != 0 {
		t.Fatalf("Main() exit code = %d, want 0", exitCode)
	}
	if stdout.String() != "1\n2\n3\n" {
		t.Fatalf("Main() stdout = %q, want %q", stdout.String(), "1\\n2\\n3\\n")
	}
	assertTimeoutDiagnostics(t, stderr.String(), "main child", "100ms", 3)
}

func TestMainTimeoutUnderUntilSuccessFixedLoopsExhaustionReturns124(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Main(
		[]string{"-n", "2", "--until-success", "--timeout", "100ms", "--", "sh", "-c", `printf "%s\n" "$AFK_INDEX"; trap 'exit 0' TERM; while :; do sleep 1; done`},
		nil,
		&stdout,
		&stderr,
	)
	if exitCode != 124 {
		t.Fatalf("Main() exit code = %d, want 124", exitCode)
	}
	if stdout.String() != "1\n2\n" {
		t.Fatalf("Main() stdout = %q, want %q", stdout.String(), "1\\n2\\n")
	}
	assertTimeoutDiagnostics(t, stderr.String(), "main child", "100ms", 2)
}

func TestMainTimeoutUnderFailStopDaemonExits124(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Main(
		[]string{"--daemon", "--fail", "stop", "--timeout", "100ms", "--", "sh", "-c", `printf "%s\n" "$AFK_INDEX"; trap 'exit 0' TERM; while :; do sleep 1; done`},
		nil,
		&stdout,
		&stderr,
	)
	if exitCode != 124 {
		t.Fatalf("Main() exit code = %d, want 124", exitCode)
	}
	if stdout.String() != "1\n" {
		t.Fatalf("Main() stdout = %q, want %q", stdout.String(), "1\\n")
	}
	assertTimeoutDiagnostics(t, stderr.String(), "main child", "100ms", 1)
}

func TestMainTimeoutUnderUntilSuccessDaemonRetriesAndStopsOnSuccess(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Main(
		[]string{"--daemon", "--until-success", "--timeout", "100ms", "--", "sh", "-c", `printf "%s\n" "$AFK_INDEX"; if [ "$AFK_INDEX" -eq 1 ]; then trap 'exit 0' TERM; while :; do sleep 1; done; fi; exit 0`},
		nil,
		&stdout,
		&stderr,
	)
	if exitCode != 0 {
		t.Fatalf("Main() exit code = %d, want 0", exitCode)
	}
	if stdout.String() != "1\n2\n" {
		t.Fatalf("Main() stdout = %q, want %q", stdout.String(), "1\\n2\\n")
	}
	assertTimeoutDiagnostics(t, stderr.String(), "main child", "100ms", 1)
}

func TestRunLoopInterruptCleansUpGrandchildrenInProcessGroup(t *testing.T) {
	tempDir := t.TempDir()
	terminatedFile := tempDir + "/interrupt-term"

	cfg := Config{
		Daemon:      true,
		CommandArgv: []string{"sh", "-c", fmt.Sprintf(`trap '' INT; trap '' TERM; (trap 'printf terminated > %s; exit 0' INT TERM; while :; do sleep 1; done) & while :; do sleep 1; done`, terminatedFile)},
	}

	interrupts := make(chan os.Signal, 2)
	resultCh := make(chan int, 1)
	stderr := new(bytes.Buffer)

	go func() {
		resultCh <- runLoopWithInterrupts(cfg, nil, new(bytes.Buffer), stderr, interrupts)
	}()

	time.Sleep(100 * time.Millisecond)
	interrupts <- syscall.SIGINT

	select {
	case exitCode := <-resultCh:
		if exitCode != 130 {
			t.Fatalf("runLoopWithInterrupts() exit code = %d, want 130", exitCode)
		}
	case <-time.After(8 * time.Second):
		t.Fatal("runLoopWithInterrupts() did not exit after SIGINT")
	}

	if _, err := os.Stat(terminatedFile); err != nil {
		t.Fatalf("expected grandchild in spawned process group to be interrupted/terminated: %v", err)
	}
	assertInterruptDiagnostics(t, stderr.String(), false)
}

func TestRunLoopInterruptDuringItemsCommandExits130AndDoesNotRunMainChild(t *testing.T) {
	tempDir := t.TempDir()
	terminatedFile := tempDir + "/items-cmd-interrupt-term"
	mainRunsFile := tempDir + "/main-runs"

	cfg := Config{
		Daemon:           true,
		ItemsCmdExplicit: true,
		ItemsCmd:         fmt.Sprintf(`trap '' INT; trap '' TERM; (trap 'printf terminated > %s; exit 0' INT TERM; while :; do sleep 1; done) & while :; do sleep 1; done`, terminatedFile),
		CommandArgv:      []string{"sh", "-c", fmt.Sprintf(`printf "ran\n" >> %s`, mainRunsFile)},
	}

	interrupts := make(chan os.Signal, 2)
	resultCh := make(chan int, 1)
	stderr := new(bytes.Buffer)

	go func() {
		resultCh <- runLoopWithInterrupts(cfg, nil, new(bytes.Buffer), stderr, interrupts)
	}()

	time.Sleep(100 * time.Millisecond)
	interrupts <- syscall.SIGINT

	select {
	case exitCode := <-resultCh:
		if exitCode != 130 {
			t.Fatalf("runLoopWithInterrupts() exit code = %d, want 130", exitCode)
		}
	case <-time.After(8 * time.Second):
		t.Fatal("runLoopWithInterrupts() did not exit after SIGINT during items-cmd execution")
	}

	if _, err := os.Stat(terminatedFile); err != nil {
		t.Fatalf("expected items-cmd process group to be interrupted/terminated: %v", err)
	}
	if _, err := os.Stat(mainRunsFile); !os.IsNotExist(err) {
		t.Fatalf("expected no main-child invocation after items-cmd interrupt, stat err=%v", err)
	}
	assertInterruptDiagnostics(t, stderr.String(), false)
}

func TestRunLoopFirstSigintStopsSchedulingNewDaemonIterations(t *testing.T) {
	tempDir := t.TempDir()
	runsFile := tempDir + "/runs"

	cfg := Config{
		Daemon:      true,
		CommandArgv: []string{"sh", "-c", fmt.Sprintf(`printf "run\n" >> %s; trap 'sleep 0.2; exit 0' INT; while :; do sleep 1; done`, runsFile)},
	}

	interrupts := make(chan os.Signal, 2)
	resultCh := make(chan int, 1)
	stderr := new(bytes.Buffer)

	go func() {
		resultCh <- runLoopWithInterrupts(cfg, nil, new(bytes.Buffer), stderr, interrupts)
	}()

	time.Sleep(100 * time.Millisecond)
	interrupts <- syscall.SIGINT

	select {
	case exitCode := <-resultCh:
		if exitCode != 130 {
			t.Fatalf("runLoopWithInterrupts() exit code = %d, want 130", exitCode)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("runLoopWithInterrupts() did not exit after SIGINT")
	}

	runs, err := os.ReadFile(runsFile)
	if err != nil {
		t.Fatalf("reading runs file: %v", err)
	}
	if got := strings.Count(string(runs), "run\n"); got != 1 {
		t.Fatalf("main-child invocation count = %d, want 1", got)
	}
	assertInterruptDiagnostics(t, stderr.String(), false)
}

func TestRunLoopSecondSigintHardKillsActiveProcessGroupAndExits130(t *testing.T) {
	cfg := Config{
		Daemon:      true,
		CommandArgv: []string{"sh", "-c", `trap '' INT TERM; while :; do sleep 1; done`},
	}

	interrupts := make(chan os.Signal, 2)
	resultCh := make(chan int, 1)
	stderr := new(bytes.Buffer)

	go func() {
		resultCh <- runLoopWithInterrupts(cfg, nil, new(bytes.Buffer), stderr, interrupts)
	}()

	time.Sleep(100 * time.Millisecond)
	started := time.Now()
	interrupts <- syscall.SIGINT
	time.Sleep(25 * time.Millisecond)
	interrupts <- syscall.SIGINT

	select {
	case exitCode := <-resultCh:
		elapsed := time.Since(started)
		if exitCode != 130 {
			t.Fatalf("runLoopWithInterrupts() exit code = %d, want 130", exitCode)
		}
		if elapsed > 1500*time.Millisecond {
			t.Fatalf("runLoopWithInterrupts() elapsed after second SIGINT = %v, want <= 1500ms for hard shutdown", elapsed)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("runLoopWithInterrupts() did not exit promptly after second SIGINT")
	}

	assertInterruptDiagnostics(t, stderr.String(), true)
}

func TestMainDefaultFailContinueStaticItemsCompletesAllItemsAfterNonZeroAndSignaledExits(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Main(
		[]string{"--items", "a\nb\nc", "--", "sh", "-c", `printf "%s\n" "$AFK_ITEM"; if [ "$AFK_INDEX" -eq 1 ]; then exit 7; fi; kill -TERM $$`},
		nil,
		&stdout,
		&stderr,
	)
	if exitCode != 0 {
		t.Fatalf("Main() exit code = %d, want 0", exitCode)
	}
	if stdout.String() != "a\nb\nc\n" {
		t.Fatalf("Main() stdout = %q, want %q", stdout.String(), "a\\nb\\nc\\n")
	}
	if stderr.Len() != 0 {
		t.Fatalf("Main() stderr = %q, want empty", stderr.String())
	}
}

func TestMainFailStopStaticItemsStopsOnFirstNonZeroAndReturnsExitCode(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Main(
		[]string{"--items", "a\nb", "--fail", "stop", "--", "sh", "-c", `printf "%s\n" "$AFK_ITEM"; if [ "$AFK_INDEX" -eq 1 ]; then kill -TERM $$; fi; exit 0`},
		nil,
		&stdout,
		&stderr,
	)
	if exitCode != 143 {
		t.Fatalf("Main() exit code = %d, want 143", exitCode)
	}
	if stdout.String() != "a\n" {
		t.Fatalf("Main() stdout = %q, want %q", stdout.String(), "a\\n")
	}
	if stderr.Len() != 0 {
		t.Fatalf("Main() stderr = %q, want empty", stderr.String())
	}
}

func TestMainDefaultFailContinueDynamicItemsCompletesBatchAfterNonZeroAndTimeout(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Main(
		[]string{"--items-cmd", "printf 'a\\nb\\n'", "--timeout", "100ms", "--", "sh", "-c", `printf "%s\n" "$AFK_ITEM"; if [ "$AFK_INDEX" -eq 1 ]; then exit 9; fi; trap 'exit 0' TERM; while :; do sleep 1; done`},
		nil,
		&stdout,
		&stderr,
	)
	if exitCode != 0 {
		t.Fatalf("Main() exit code = %d, want 0", exitCode)
	}
	if stdout.String() != "a\nb\n" {
		t.Fatalf("Main() stdout = %q, want %q", stdout.String(), "a\\nb\\n")
	}
	assertTimeoutDiagnostics(t, stderr.String(), "main child", "100ms", 1)
}

func TestMainFailStopDynamicItemsStopsOnTimeoutAndReturns124(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Main(
		[]string{"--items-cmd", "printf 'a\\nb\\n'", "--fail", "stop", "--timeout", "100ms", "--", "sh", "-c", `printf "%s\n" "$AFK_ITEM"; trap 'exit 0' TERM; while :; do sleep 1; done`},
		nil,
		&stdout,
		&stderr,
	)
	if exitCode != 124 {
		t.Fatalf("Main() exit code = %d, want 124", exitCode)
	}
	if stdout.String() != "a\n" {
		t.Fatalf("Main() stdout = %q, want %q", stdout.String(), "a\\n")
	}
	assertTimeoutDiagnostics(t, stderr.String(), "main child", "100ms", 1)
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
