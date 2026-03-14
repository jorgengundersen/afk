package terminal_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/jorgengundersen/afk/internal/terminal"
)

func TestStarting_NormalMode(t *testing.T) {
	var buf bytes.Buffer
	p := terminal.New(&buf, false, false)
	p.Starting("max-iterations", 20, "claude", true)

	got := buf.String()
	want := "afk: starting (max-iterations=20, harness=claude, beads=on)\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestStarting_QuietMode(t *testing.T) {
	var buf bytes.Buffer
	p := terminal.New(&buf, true, false)
	p.Starting("max-iterations", 20, "claude", true)

	if buf.Len() != 0 {
		t.Errorf("quiet mode should produce no output, got %q", buf.String())
	}
}

func TestStarting_DaemonMode(t *testing.T) {
	var buf bytes.Buffer
	p := terminal.New(&buf, false, false)
	p.Starting("daemon", 0, "opencode", false)

	got := buf.String()
	want := "afk: starting (daemon, harness=opencode, beads=off)\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestIteration_WithIssue(t *testing.T) {
	var buf bytes.Buffer
	p := terminal.New(&buf, false, false)
	p.Iteration(1, 20, "afk-42", "Fix auth bug")

	got := buf.String()
	want := "afk: [1/20] working on afk-42 \"Fix auth bug\"\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestIteration_WithoutIssue(t *testing.T) {
	var buf bytes.Buffer
	p := terminal.New(&buf, false, false)
	p.Iteration(3, 10, "", "")

	got := buf.String()
	want := "afk: [3/10] working\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestIteration_DaemonMode(t *testing.T) {
	var buf bytes.Buffer
	p := terminal.New(&buf, false, false)
	p.Iteration(5, 0, "afk-7", "Add tests")

	got := buf.String()
	want := "afk: [5] working on afk-7 \"Add tests\"\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestIteration_QuietMode(t *testing.T) {
	var buf bytes.Buffer
	p := terminal.New(&buf, true, false)
	p.Iteration(1, 20, "afk-42", "Fix auth bug")

	if buf.Len() != 0 {
		t.Errorf("quiet mode should produce no output, got %q", buf.String())
	}
}

func TestSleeping(t *testing.T) {
	var buf bytes.Buffer
	p := terminal.New(&buf, false, false)
	p.Sleeping(60 * time.Second)

	got := buf.String()
	want := "afk: sleeping (1m0s)\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSleeping_QuietMode(t *testing.T) {
	var buf bytes.Buffer
	p := terminal.New(&buf, true, false)
	p.Sleeping(60 * time.Second)

	if buf.Len() != 0 {
		t.Errorf("quiet mode should produce no output, got %q", buf.String())
	}
}

func TestWaking(t *testing.T) {
	var buf bytes.Buffer
	p := terminal.New(&buf, false, false)
	p.Waking()

	got := buf.String()
	want := "afk: waking\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDone_NoWorkRemaining(t *testing.T) {
	var buf bytes.Buffer
	p := terminal.New(&buf, false, false)
	p.Done(2, 2, 0, "no work remaining")

	got := buf.String()
	want := "afk: done (2 iterations, no work remaining)\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDone_WithFailures(t *testing.T) {
	var buf bytes.Buffer
	p := terminal.New(&buf, false, false)
	p.Done(3, 1, 2, "max iterations reached")

	got := buf.String()
	want := "afk: done (3 iterations, 1 succeeded, 2 failed, max iterations reached)\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDone_QuietMode(t *testing.T) {
	var buf bytes.Buffer
	p := terminal.New(&buf, true, false)
	p.Done(2, 2, 0, "no work remaining")

	if buf.Len() != 0 {
		t.Errorf("quiet mode should produce no output, got %q", buf.String())
	}
}

func TestDone_SingleIteration(t *testing.T) {
	var buf bytes.Buffer
	p := terminal.New(&buf, false, false)
	p.Done(1, 1, 0, "no work remaining")

	got := buf.String()
	want := "afk: done (1 iteration, no work remaining)\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestVerboseDetail_NormalMode(t *testing.T) {
	var buf bytes.Buffer
	p := terminal.New(&buf, false, false)
	p.VerboseDetail("exit-code=0 duration=12s")

	if buf.Len() != 0 {
		t.Errorf("normal mode should not show verbose detail, got %q", buf.String())
	}
}

func TestVerboseDetail_VerboseMode(t *testing.T) {
	var buf bytes.Buffer
	p := terminal.New(&buf, false, true)
	p.VerboseDetail("exit-code=0 duration=12s")

	got := buf.String()
	want := "afk:   exit-code=0 duration=12s\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestIsVerbose(t *testing.T) {
	p := terminal.New(nil, false, true)
	if !p.IsVerbose() {
		t.Error("expected IsVerbose() to be true")
	}
	p2 := terminal.New(nil, false, false)
	if p2.IsVerbose() {
		t.Error("expected IsVerbose() to be false")
	}
}
