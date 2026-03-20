// Package loop implements the core iteration logic for afk.
package loop

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

// EventLogger is the logging interface used by the loop.
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

// doOnce executes one iteration of the loop: claim an issue, assemble a
// prompt, run the harness, and log the result.
func doOnce(ctx context.Context, cfg config.Config, h Harness, bc BeadsClient, log EventLogger, iterNum int) (bool, error) {
	r := runOnce(ctx, cfg, h, bc, log, iterNum)
	return r.ran, r.err
}

func runOnce(ctx context.Context, cfg config.Config, h Harness, bc BeadsClient, log EventLogger, iterNum int) iterResult {
	if err := ctx.Err(); err != nil {
		return iterResult{err: err}
	}

	var issue *beads.Issue

	if cfg.BeadsEnabled && bc != nil {
		issues, err := bc.Ready(ctx)
		if errors.Is(err, beads.ErrNoWork) {
			log.Event("beads-check", Field{"count", "0"})
			if cfg.Prompt == "" {
				// No work and no prompt — skip this iteration entirely (Option C).
				return iterResult{}
			}
			// No issue but we have a user prompt, continue without issue.
		} else if err != nil {
			log.Event("error", Field{"message", fmt.Sprintf("beads ready: %s", err)})
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
	assembled, err := prompt.Assemble(cfg.Prompt, issue, instruction)
	if err != nil {
		return iterResult{err: err}
	}

	if err := ctx.Err(); err != nil {
		return iterResult{err: err}
	}

	var issueID, issueTitle string
	if issue != nil {
		issueID = issue.ID
		issueTitle = issue.Title
	}

	// Emit iteration-start with spec fields.
	startFields := []Field{{"iteration", fmt.Sprintf("%d", iterNum)}}
	if issue != nil {
		startFields = append(startFields, Field{"issue", issue.ID}, Field{"title", issue.Title})
	}
	log.Event("iteration-start", startFields...)

	iterStart := time.Now()

	exitCode, err := h.Run(ctx, assembled)
	if err != nil {
		log.Event("iteration-end",
			Field{"iteration", fmt.Sprintf("%d", iterNum)},
			Field{"exit-code", fmt.Sprintf("%d", exitCode)},
			Field{"duration", time.Since(iterStart).Round(time.Second).String()},
		)
		return iterResult{ran: true, issueID: issueID, issueTitle: issueTitle, err: fmt.Errorf("harness run: %w", err)}
	}

	log.Event("iteration-end",
		Field{"iteration", fmt.Sprintf("%d", iterNum)},
		Field{"exit-code", "0"},
		Field{"duration", time.Since(iterStart).Round(time.Second).String()},
	)
	return iterResult{ran: true, issueID: issueID, issueTitle: issueTitle}
}

// Run is the main entry point for the loop package. It dispatches to
// RunMaxIter or RunDaemon based on cfg.Mode.
func Run(ctx context.Context, cfg config.Config, h Harness, bc BeadsClient, log EventLogger, pr Printer) error {
	switch cfg.Mode {
	case config.DaemonMode:
		return RunDaemon(ctx, cfg, h, bc, log, pr)
	default:
		return RunMaxIter(ctx, cfg, h, bc, log, pr)
	}
}

// modeString returns the human-readable name for a config.Mode.
func modeString(m config.Mode) string {
	switch m {
	case config.DaemonMode:
		return "daemon"
	default:
		return "max-iterations"
	}
}

// sessionStartFields builds the fields for the session-start event.
func sessionStartFields(cfg config.Config) []Field {
	fields := []Field{{"mode", modeString(cfg.Mode)}}
	if cfg.Mode == config.MaxIterationsMode {
		fields = append(fields, Field{"max", fmt.Sprintf("%d", cfg.MaxIterations)})
	}
	fields = append(fields,
		Field{"harness", cfg.Harness},
		Field{"beads", fmt.Sprintf("%t", cfg.BeadsEnabled)},
	)
	if len(cfg.BeadsLabels) > 0 {
		fields = append(fields, Field{"labels", strings.Join(cfg.BeadsLabels, ",")})
	}
	return fields
}

// RunDaemon runs the loop indefinitely until the context is cancelled.
// When no work is found, it sleeps for cfg.SleepInterval before re-checking.
// When work is done, it immediately checks for the next issue.
func RunDaemon(ctx context.Context, cfg config.Config, h Harness, bc BeadsClient, log EventLogger, pr Printer) error {
	sessionStart := time.Now()
	log.Event("session-start", sessionStartFields(cfg)...)
	pr.Starting("daemon", 0, cfg.Harness, cfg.BeadsEnabled)

	iterNum := 0
	iterCount := 0
	for {
		if ctx.Err() != nil {
			log.Event("session-end",
				Field{"reason", "signal"},
				Field{"total-iterations", fmt.Sprintf("%d", iterCount)},
				Field{"duration", time.Since(sessionStart).Round(time.Second).String()},
			)
			return nil
		}

		iterNum++
		r := runOnce(ctx, cfg, h, bc, log, iterNum)
		if r.ran {
			iterCount++
			pr.Iteration(iterNum, 0, r.issueID, r.issueTitle)
		}
		if r.err != nil {
			log.Event("error", Field{"message", r.err.Error()})
		}

		if ctx.Err() != nil {
			log.Event("session-end",
				Field{"reason", "signal"},
				Field{"total-iterations", fmt.Sprintf("%d", iterCount)},
				Field{"duration", time.Since(sessionStart).Round(time.Second).String()},
			)
			return nil
		}

		if !r.ran {
			log.Event("sleeping", Field{"duration", cfg.SleepInterval.String()})
			pr.Sleeping(cfg.SleepInterval)
			t := time.NewTimer(cfg.SleepInterval)
			select {
			case <-ctx.Done():
				t.Stop()
				log.Event("session-end",
					Field{"reason", "signal"},
					Field{"total-iterations", fmt.Sprintf("%d", iterCount)},
					Field{"duration", time.Since(sessionStart).Round(time.Second).String()},
				)
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

// RunMaxIter runs the loop up to cfg.MaxIterations times.
// Stops early if no work is available or context is cancelled.
// Returns ErrAllFailed if every iteration failed.
func RunMaxIter(ctx context.Context, cfg config.Config, h Harness, bc BeadsClient, log EventLogger, pr Printer) error {
	sessionStart := time.Now()
	log.Event("session-start", sessionStartFields(cfg)...)
	pr.Starting("max-iterations", cfg.MaxIterations, cfg.Harness, cfg.BeadsEnabled)

	var total, succeeded, failed int

	for i := 0; i < cfg.MaxIterations; i++ {
		r := runOnce(ctx, cfg, h, bc, log, i+1)
		if r.ran {
			pr.Iteration(i+1, cfg.MaxIterations, r.issueID, r.issueTitle)
		}
		if ctx.Err() != nil {
			log.Event("session-end",
				Field{"reason", "context-cancelled"},
				Field{"total-iterations", fmt.Sprintf("%d", total)},
				Field{"duration", time.Since(sessionStart).Round(time.Second).String()},
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

	reason := "max-iterations"
	if total < cfg.MaxIterations {
		reason = "no-work"
	}
	pr.Done(total, succeeded, failed, reason)

	log.Event("session-end",
		Field{"reason", reason},
		Field{"total-iterations", fmt.Sprintf("%d", total)},
		Field{"duration", time.Since(sessionStart).Round(time.Second).String()},
	)

	if total > 0 && failed == total {
		return ErrAllFailed
	}
	return nil
}
