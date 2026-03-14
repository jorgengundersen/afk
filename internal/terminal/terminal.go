// Package terminal provides human-facing terminal output for afk.
package terminal

import (
	"fmt"
	"io"
	"time"
)

// Printer writes human-facing progress messages to a writer.
type Printer struct {
	w       io.Writer
	quiet   bool
	verbose bool
}

// New creates a Printer that writes to w, respecting quiet and verbose flags.
func New(w io.Writer, quiet, verbose bool) *Printer {
	return &Printer{w: w, quiet: quiet, verbose: verbose}
}

// Starting prints the session start message.
func (p *Printer) Starting(mode string, maxIter int, harness string, beads bool) {
	if p.quiet {
		return
	}
	beadsStr := "off"
	if beads {
		beadsStr = "on"
	}
	if mode == "max-iterations" {
		fmt.Fprintf(p.w, "afk: starting (max-iterations=%d, harness=%s, beads=%s)\n", maxIter, harness, beadsStr)
	} else {
		fmt.Fprintf(p.w, "afk: starting (daemon, harness=%s, beads=%s)\n", harness, beadsStr)
	}
}

// Iteration prints the iteration start message. maxIter=0 means daemon mode (no total shown).
func (p *Printer) Iteration(n, maxIter int, issueID, title string) {
	if p.quiet {
		return
	}
	var counter string
	if maxIter > 0 {
		counter = fmt.Sprintf("[%d/%d]", n, maxIter)
	} else {
		counter = fmt.Sprintf("[%d]", n)
	}
	if issueID != "" {
		fmt.Fprintf(p.w, "afk: %s working on %s %q\n", counter, issueID, title)
	} else {
		fmt.Fprintf(p.w, "afk: %s working\n", counter)
	}
}

// Sleeping prints a message when entering sleep in daemon mode.
func (p *Printer) Sleeping(d time.Duration) {
	if p.quiet {
		return
	}
	fmt.Fprintf(p.w, "afk: sleeping (%s)\n", d)
}

// Waking prints a message when waking from sleep in daemon mode.
func (p *Printer) Waking() {
	if p.quiet {
		return
	}
	fmt.Fprintln(p.w, "afk: waking")
}

// Done prints the session completion summary.
func (p *Printer) Done(total, succeeded, failed int, reason string) {
	if p.quiet {
		return
	}
	noun := "iterations"
	if total == 1 {
		noun = "iteration"
	}
	if failed > 0 {
		fmt.Fprintf(p.w, "afk: done (%d %s, %d succeeded, %d failed, %s)\n", total, noun, succeeded, failed, reason)
	} else {
		fmt.Fprintf(p.w, "afk: done (%d %s, %s)\n", total, noun, reason)
	}
}

// VerboseDetail prints extra detail only in verbose mode.
func (p *Printer) VerboseDetail(msg string) {
	if !p.verbose {
		return
	}
	fmt.Fprintf(p.w, "afk:   %s\n", msg)
}

// IsVerbose reports whether verbose mode is enabled.
func (p *Printer) IsVerbose() bool {
	return p.verbose
}
