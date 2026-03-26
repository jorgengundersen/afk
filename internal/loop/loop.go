package loop

import (
	"context"
	"fmt"
	"time"
)

// Config holds the parameters the loop needs to run.
type Config struct {
	MaxIter int
	Daemon  bool
	Sleep   time.Duration
	Prompt  string
}

// Logger is the interface the loop uses to record events.
type Logger interface {
	Log(event string, fields map[string]any) error
}

// Runner executes an agent with the given prompt.
type Runner interface {
	Run(ctx context.Context, prompt string) (int, error)
}

// WorkSource provides work items for each iteration. When nil is passed
// to Run, the loop uses the static prompt from Config.
type WorkSource interface {
	Next() (prompt string, issueID string, issueTitle string, ok bool, err error)
}

// logErr wraps a log write failure for identification.
func logErr(err error) error {
	return fmt.Errorf("log failure: %w", err)
}

// Run orchestrates harness invocations in a loop.
func Run(ctx context.Context, cfg Config, runner Runner, logger Logger, ws WorkSource) (int, error) {
	if err := logger.Log("session-start", map[string]any{
		"mode":    mode(cfg),
		"maxIter": cfg.MaxIter,
	}); err != nil {
		return 1, logErr(err)
	}

	if cfg.Daemon {
		return runDaemon(ctx, cfg, runner, logger, ws)
	}
	return runMaxIter(ctx, cfg, runner, logger, ws)
}

func runMaxIter(ctx context.Context, cfg Config, runner Runner, logger Logger, ws WorkSource) (int, error) {
	allFailed := true
	iterations := 0

	for i := 1; i <= cfg.MaxIter; i++ {
		if ctx.Err() != nil {
			break
		}

		prompt, issueID, issueTitle, ok, wsErr := resolveWork(ws, cfg.Prompt)
		if ws != nil {
			if wsErr != nil {
				if err := logger.Log("work-source-error", map[string]any{"err": wsErr.Error()}); err != nil {
					return 1, logErr(err)
				}
				iterations++
				continue
			}
			if err := logger.Log("beads-check", map[string]any{"hasWork": ok}); err != nil {
				return 1, logErr(err)
			}
			if !ok {
				if err := logger.Log("session-end", map[string]any{"reason": "no-work"}); err != nil {
					return 1, logErr(err)
				}
				return 0, nil
			}
		}

		iterations++
		res := runIteration(ctx, i, prompt, issueID, issueTitle, runner, logger)
		if res.logErr != nil {
			return res.exitCode, res.logErr
		}

		if res.runErr == nil && res.exitCode == 0 {
			allFailed = false
		}
	}

	reason := "complete"
	if ctx.Err() != nil {
		reason = "signal"
	}
	if err := logger.Log("session-end", map[string]any{"reason": reason}); err != nil {
		return 1, logErr(err)
	}

	if iterations > 0 && allFailed {
		return 1, nil
	}
	return 0, nil
}

func runDaemon(ctx context.Context, cfg Config, runner Runner, logger Logger, ws WorkSource) (int, error) {
	iteration := 0
	allFailed := true
	for {
		if ctx.Err() != nil {
			break
		}

		prompt, issueID, issueTitle, ok, wsErr := resolveWork(ws, cfg.Prompt)
		if ws != nil {
			if wsErr != nil {
				if err := logger.Log("work-source-error", map[string]any{"err": wsErr.Error()}); err != nil {
					return 1, logErr(err)
				}
				if ctx.Err() != nil {
					break
				}
				if err := logger.Log("sleeping", map[string]any{"duration": cfg.Sleep}); err != nil {
					return 1, logErr(err)
				}
				select {
				case <-time.After(cfg.Sleep):
					if err := logger.Log("waking", nil); err != nil {
						return 1, logErr(err)
					}
				case <-ctx.Done():
				}
				continue
			}
			if err := logger.Log("beads-check", map[string]any{"hasWork": ok}); err != nil {
				return 1, logErr(err)
			}
			if !ok {
				// Daemon mode: sleep and retry when no work
				if ctx.Err() != nil {
					break
				}
				if err := logger.Log("sleeping", map[string]any{"duration": cfg.Sleep}); err != nil {
					return 1, logErr(err)
				}
				select {
				case <-time.After(cfg.Sleep):
					if err := logger.Log("waking", nil); err != nil {
						return 1, logErr(err)
					}
				case <-ctx.Done():
				}
				continue
			}
		}

		iteration++
		res := runIteration(ctx, iteration, prompt, issueID, issueTitle, runner, logger)
		if res.logErr != nil {
			return 1, res.logErr
		}

		if res.runErr == nil && res.exitCode == 0 {
			allFailed = false
		}

		if ctx.Err() != nil {
			break
		}

		if err := logger.Log("sleeping", map[string]any{"duration": cfg.Sleep}); err != nil {
			return 1, logErr(err)
		}
		select {
		case <-time.After(cfg.Sleep):
			if err := logger.Log("waking", nil); err != nil {
				return 1, logErr(err)
			}
		case <-ctx.Done():
		}
	}

	if err := logger.Log("session-end", map[string]any{"reason": "signal"}); err != nil {
		return 1, logErr(err)
	}
	if iteration > 0 && allFailed {
		return 1, nil
	}
	return 0, nil
}

func resolveWork(ws WorkSource, staticPrompt string) (prompt, issueID, issueTitle string, ok bool, err error) {
	if ws == nil {
		return staticPrompt, "", "", true, nil
	}
	return ws.Next()
}

// iterResult holds the outcome of a single iteration.
type iterResult struct {
	exitCode int
	runErr   error // error from the runner (nil = runner succeeded)
	logErr   error // error from logging (nil = logging succeeded)
}

func runIteration(ctx context.Context, iteration int, prompt, issueID, issueTitle string, runner Runner, logger Logger) iterResult {
	fields := map[string]any{"iteration": iteration}
	if issueID != "" {
		fields["issueID"] = issueID
		fields["issueTitle"] = issueTitle
	}
	if err := logger.Log("iteration-start", fields); err != nil {
		return iterResult{exitCode: 1, logErr: logErr(err)}
	}
	start := time.Now()

	exitCode, runErr := runner.Run(ctx, prompt)
	duration := time.Since(start)

	endFields := map[string]any{
		"iteration": iteration,
		"exitCode":  exitCode,
		"duration":  duration,
	}
	if issueID != "" {
		endFields["issueID"] = issueID
		endFields["issueTitle"] = issueTitle
	}
	if err := logger.Log("iteration-end", endFields); err != nil {
		return iterResult{exitCode: 1, logErr: logErr(err)}
	}

	if runErr != nil {
		if err := logger.Log("error", map[string]any{"iteration": iteration, "err": runErr.Error()}); err != nil {
			return iterResult{exitCode: 1, logErr: logErr(err)}
		}
	}

	return iterResult{exitCode: exitCode, runErr: runErr}
}

func mode(cfg Config) string {
	if cfg.Daemon {
		return "daemon"
	}
	return "max-iterations"
}
