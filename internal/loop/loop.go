package loop

import (
	"context"
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
	Log(event string, fields map[string]any)
}

// Runner executes an agent with the given prompt.
type Runner interface {
	Run(ctx context.Context, prompt string) (int, error)
}

// WorkSource provides work items for each iteration. When nil is passed
// to Run, the loop uses the static prompt from Config.
type WorkSource interface {
	Next() (prompt string, issueID string, issueTitle string, ok bool)
}

// Run orchestrates harness invocations in a loop.
func Run(ctx context.Context, cfg Config, runner Runner, logger Logger, ws WorkSource) (int, error) {
	logger.Log("session-start", map[string]any{
		"mode":    mode(cfg),
		"maxIter": cfg.MaxIter,
	})

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

		prompt, issueID, issueTitle, ok := resolveWork(ws, cfg.Prompt)
		if ws != nil {
			logger.Log("beads-check", map[string]any{"hasWork": ok})
			if !ok {
				logger.Log("session-end", map[string]any{"reason": "no-work"})
				return 0, nil
			}
		}

		iterations++
		exitCode, err := runIteration(ctx, i, prompt, issueID, issueTitle, runner, logger)
		_ = exitCode

		if err != nil {
			logger.Log("error", map[string]any{"iteration": i, "err": err.Error()})
		} else {
			allFailed = false
		}
	}

	reason := "complete"
	if ctx.Err() != nil {
		reason = "signal"
	}
	logger.Log("session-end", map[string]any{"reason": reason})

	if iterations > 0 && allFailed {
		return 1, nil
	}
	return 0, nil
}

func runDaemon(ctx context.Context, cfg Config, runner Runner, logger Logger, ws WorkSource) (int, error) {
	for {
		if ctx.Err() != nil {
			break
		}

		prompt, issueID, issueTitle, ok := resolveWork(ws, cfg.Prompt)
		if ws != nil {
			logger.Log("beads-check", map[string]any{"hasWork": ok})
			if !ok {
				// Daemon mode: sleep and retry when no work
				if ctx.Err() != nil {
					break
				}
				logger.Log("sleeping", map[string]any{"duration": cfg.Sleep})
				select {
				case <-time.After(cfg.Sleep):
					logger.Log("waking", nil)
				case <-ctx.Done():
				}
				continue
			}
		}

		_, err := runIteration(ctx, 0, prompt, issueID, issueTitle, runner, logger)
		if err != nil {
			logger.Log("error", map[string]any{"err": err.Error()})
		}

		if ctx.Err() != nil {
			break
		}

		logger.Log("sleeping", map[string]any{"duration": cfg.Sleep})
		select {
		case <-time.After(cfg.Sleep):
			logger.Log("waking", nil)
		case <-ctx.Done():
		}
	}

	logger.Log("session-end", map[string]any{"reason": "signal"})
	return 0, nil
}

func resolveWork(ws WorkSource, staticPrompt string) (prompt, issueID, issueTitle string, ok bool) {
	if ws == nil {
		return staticPrompt, "", "", true
	}
	return ws.Next()
}

func runIteration(ctx context.Context, iteration int, prompt, issueID, issueTitle string, runner Runner, logger Logger) (int, error) {
	fields := map[string]any{"iteration": iteration}
	if issueID != "" {
		fields["issueID"] = issueID
		fields["issueTitle"] = issueTitle
	}
	logger.Log("iteration-start", fields)
	start := time.Now()

	exitCode, err := runner.Run(ctx, prompt)
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
	logger.Log("iteration-end", endFields)

	return exitCode, err
}

func mode(cfg Config) string {
	if cfg.Daemon {
		return "daemon"
	}
	return "max-iterations"
}
