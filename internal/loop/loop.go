// Package loop implements the core iteration logic for afk.
package loop

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jorgengundersen/afk/internal/beads"
	"github.com/jorgengundersen/afk/internal/config"
	"github.com/jorgengundersen/afk/internal/prompt"
)

// Field is a key-value pair for log events, matching eventlog.Field.
type Field struct {
	Key   string
	Value string
}

// EventLogger is the logging interface used by RunOnce.
type EventLogger interface {
	Event(name string, fields ...Field)
}

// BeadsClient is the interface for fetching issues.
type BeadsClient interface {
	Ready(ctx context.Context) ([]beads.Issue, error)
}

// Harness runs an external agent CLI with a given prompt.
type Harness interface {
	Run(ctx context.Context, prompt string) (exitCode int, err error)
}

// Printer is the terminal output interface for human-facing progress messages.
type Printer interface {
	Starting(mode string, maxIter int, harness string, beads bool)
	Iteration(n, maxIter int, issueID, title string)
	Sleeping(d time.Duration)
	Waking()
	Done(total, succeeded, failed int, reason string)
	VerboseDetail(msg string)
}

// iterResult holds the outcome of a single iteration.
type iterResult struct {
	ran        bool
	issueID    string
	issueTitle string
	err        error
}

// RunOnce executes one iteration of the loop: claim an issue, assemble a
// prompt, run the harness, and log the result.
func RunOnce(ctx context.Context, cfg config.Config, h Harness, bc BeadsClient, log EventLogger) (bool, error) {
	r := runOnce(ctx, cfg, h, bc, log)
	return r.ran, r.err
}

func runOnce(ctx context.Context, cfg config.Config, h Harness, bc BeadsClient, log EventLogger) iterResult {
	if err := ctx.Err(); err != nil {
		return iterResult{err: err}
	}

	log.Event("iteration-start")

	var issue *beads.Issue

	if cfg.BeadsEnabled && bc != nil {
		issues, err := bc.Ready(ctx)
		if errors.Is(err, beads.ErrNoWork) {
			log.Event("beads-check", Field{"count", "0"})
			if cfg.Prompt == "" {
				log.Event("iteration-end", Field{"status", "no_work"})
				return iterResult{}
			}
			// No issue but we have a user prompt, continue without issue.
		} else if err != nil {
			log.Event("iteration-end", Field{"status", "fail"}, Field{"error", err.Error()})
			return iterResult{err: fmt.Errorf("beads ready: %w", err)}
		} else {
			log.Event("beads-check", Field{"count", fmt.Sprintf("%d", len(issues))})
			if len(issues) > 0 {
				issue = &issues[0]
			}
		}
	}

	instruction := cfg.BeadsInstruct
	if instruction == "" {
		instruction = prompt.DefaultInstruction
	}
	assembled := prompt.Assemble(cfg.Prompt, issue, instruction)

	if err := ctx.Err(); err != nil {
		return iterResult{err: err}
	}

	var issueID, issueTitle string
	if issue != nil {
		issueID = issue.ID
		issueTitle = issue.Title
	}

	_, err := h.Run(ctx, assembled)
	if err != nil {
		log.Event("iteration-end", Field{"status", "fail"}, Field{"error", err.Error()})
		return iterResult{ran: true, issueID: issueID, issueTitle: issueTitle, err: fmt.Errorf("harness run: %w", err)}
	}

	log.Event("iteration-end", Field{"status", "ok"})
	return iterResult{ran: true, issueID: issueID, issueTitle: issueTitle}
}

// RunDaemon runs RunOnce in a loop indefinitely until the context is cancelled.
// When RunOnce returns no work, it sleeps for cfg.SleepInterval before re-checking.
// When RunOnce returns work done, it immediately checks for the next issue.
func RunDaemon(ctx context.Context, cfg config.Config, h Harness, bc BeadsClient, log EventLogger, pr Printer) error {
	log.Event("session-start")
	pr.Starting("daemon", 0, cfg.Harness, cfg.BeadsEnabled)

	iterNum := 0
	for {
		if ctx.Err() != nil {
			log.Event("session-end")
			return nil
		}

		iterNum++
		r := runOnce(ctx, cfg, h, bc, log)
		if r.ran {
			pr.Iteration(iterNum, 0, r.issueID, r.issueTitle)
		}
		if r.err != nil {
			log.Event("error", Field{"message", r.err.Error()})
		}

		if ctx.Err() != nil {
			log.Event("session-end")
			return nil
		}

		if !r.ran {
			log.Event("sleeping", Field{"duration", cfg.SleepInterval.String()})
			pr.Sleeping(cfg.SleepInterval)
			t := time.NewTimer(cfg.SleepInterval)
			select {
			case <-ctx.Done():
				t.Stop()
				log.Event("session-end")
				return nil
			case <-t.C:
			}
			log.Event("waking")
			pr.Waking()
		}
	}
}

// ErrAllFailed is returned when every iteration's harness returned non-zero.
var ErrAllFailed = errors.New("all iterations failed")

// RunMaxIter runs RunOnce up to cfg.MaxIterations times.
// Stops early if RunOnce returns no work or context is cancelled.
// Returns ErrAllFailed if every iteration failed.
func RunMaxIter(ctx context.Context, cfg config.Config, h Harness, bc BeadsClient, log EventLogger, pr Printer) error {
	log.Event("session-start")
	pr.Starting("max-iterations", cfg.MaxIterations, cfg.Harness, cfg.BeadsEnabled)

	var total, succeeded, failed int

	for i := 0; i < cfg.MaxIterations; i++ {
		r := runOnce(ctx, cfg, h, bc, log)
		if r.ran {
			pr.Iteration(i+1, cfg.MaxIterations, r.issueID, r.issueTitle)
		}
		if ctx.Err() != nil {
			log.Event("session-end",
				Field{"total", fmt.Sprintf("%d", total)},
				Field{"succeeded", fmt.Sprintf("%d", succeeded)},
				Field{"failed", fmt.Sprintf("%d", failed)},
			)
			return nil
		}
		if !r.ran && r.err == nil {
			// No work available, stop early.
			break
		}
		total++
		if r.err != nil {
			log.Event("error", Field{"message", r.err.Error()})
			failed++
		} else {
			succeeded++
		}
	}

	reason := "max iterations reached"
	if total < cfg.MaxIterations {
		reason = "no work remaining"
	}
	pr.Done(total, succeeded, failed, reason)

	log.Event("session-end",
		Field{"total", fmt.Sprintf("%d", total)},
		Field{"succeeded", fmt.Sprintf("%d", succeeded)},
		Field{"failed", fmt.Sprintf("%d", failed)},
	)

	if total > 0 && failed == total {
		return ErrAllFailed
	}
	return nil
}
