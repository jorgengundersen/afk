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
