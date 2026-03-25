package loop

import (
	"context"
	"time"

	"github.com/jorgengundersen/afk/internal/config"
)

// Logger is the interface the loop uses to record events.
type Logger interface {
	Log(event string, fields map[string]any)
}

// Runner executes an agent with the given prompt.
type Runner interface {
	Run(ctx context.Context, prompt string) (int, error)
}

// Run orchestrates harness invocations in a loop.
func Run(ctx context.Context, cfg config.Config, runner Runner, logger Logger) (int, error) {
	logger.Log("session-start", map[string]any{
		"mode":    mode(cfg),
		"maxIter": cfg.MaxIter,
	})

	if cfg.Daemon {
		return runDaemon(ctx, cfg, runner, logger)
	}
	return runMaxIter(ctx, cfg, runner, logger)
}

func runMaxIter(ctx context.Context, cfg config.Config, runner Runner, logger Logger) (int, error) {
	allFailed := true
	iterations := 0

	for i := 1; i <= cfg.MaxIter; i++ {
		if ctx.Err() != nil {
			break
		}

		iterations++
		exitCode, err := runIteration(ctx, i, cfg.Prompt, runner, logger)
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

func runDaemon(ctx context.Context, cfg config.Config, runner Runner, logger Logger) (int, error) {
	for {
		if ctx.Err() != nil {
			break
		}

		_, err := runIteration(ctx, 0, cfg.Prompt, runner, logger)
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

func runIteration(ctx context.Context, iteration int, prompt string, runner Runner, logger Logger) (int, error) {
	logger.Log("iteration-start", map[string]any{"iteration": iteration})
	start := time.Now()

	exitCode, err := runner.Run(ctx, prompt)
	duration := time.Since(start)

	logger.Log("iteration-end", map[string]any{
		"iteration": iteration,
		"exitCode":  exitCode,
		"duration":  duration,
	})

	return exitCode, err
}

func mode(cfg config.Config) string {
	if cfg.Daemon {
		return "daemon"
	}
	return "max-iterations"
}
