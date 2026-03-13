// Package loop implements the core iteration logic for afk.
package loop

import (
	"context"
	"errors"
	"fmt"

	"github.com/jorgengundersen/afk/beads"
	"github.com/jorgengundersen/afk/config"
	"github.com/jorgengundersen/afk/prompt"
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

// RunOnce executes one iteration of the loop: claim an issue, assemble a
// prompt, run the harness, and log the result.
func RunOnce(ctx context.Context, cfg config.Config, h Harness, bc BeadsClient, log EventLogger) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	log.Event("iteration_start")

	var issue *beads.Issue

	if cfg.BeadsEnabled && bc != nil {
		issues, err := bc.Ready(ctx)
		if errors.Is(err, beads.ErrNoWork) {
			if cfg.Prompt == "" {
				log.Event("iteration_end", Field{"status", "no_work"})
				return false, nil
			}
			// No issue but we have a user prompt, continue without issue.
		} else if err != nil {
			log.Event("iteration_end", Field{"status", "fail"}, Field{"error", err.Error()})
			return false, fmt.Errorf("beads ready: %w", err)
		} else if len(issues) > 0 {
			issue = &issues[0]
		}
	}

	assembled := prompt.Assemble(cfg.Prompt, issue)

	if err := ctx.Err(); err != nil {
		return false, err
	}

	_, err := h.Run(ctx, assembled)
	if err != nil {
		log.Event("iteration_end", Field{"status", "fail"}, Field{"error", err.Error()})
		return true, fmt.Errorf("harness run: %w", err)
	}

	log.Event("iteration_end", Field{"status", "ok"})
	return true, nil
}

// ErrAllFailed is returned when every iteration's harness returned non-zero.
var ErrAllFailed = errors.New("all iterations failed")

// RunMaxIter runs RunOnce up to cfg.MaxIterations times.
// Stops early if RunOnce returns no work or context is cancelled.
// Returns ErrAllFailed if every iteration failed.
func RunMaxIter(ctx context.Context, cfg config.Config, h Harness, bc BeadsClient, log EventLogger) error {
	log.Event("loop_start")

	var total, succeeded, failed int

	for i := 0; i < cfg.MaxIterations; i++ {
		ran, err := RunOnce(ctx, cfg, h, bc, log)
		if ctx.Err() != nil {
			log.Event("loop_end",
				Field{"total", fmt.Sprintf("%d", total)},
				Field{"succeeded", fmt.Sprintf("%d", succeeded)},
				Field{"failed", fmt.Sprintf("%d", failed)},
			)
			return nil
		}
		if !ran && err == nil {
			// No work available, stop early.
			break
		}
		total++
		if err != nil {
			failed++
		} else {
			succeeded++
		}
	}

	log.Event("loop_end",
		Field{"total", fmt.Sprintf("%d", total)},
		Field{"succeeded", fmt.Sprintf("%d", succeeded)},
		Field{"failed", fmt.Sprintf("%d", failed)},
	)

	if total > 0 && failed == total {
		return ErrAllFailed
	}
	return nil
}
