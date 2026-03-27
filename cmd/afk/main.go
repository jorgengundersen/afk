package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/jorgengundersen/afk/internal/beads"
	"github.com/jorgengundersen/afk/internal/config"
	"github.com/jorgengundersen/afk/internal/harness"
	"github.com/jorgengundersen/afk/internal/logger"
	"github.com/jorgengundersen/afk/internal/loop"
	"github.com/jorgengundersen/afk/internal/prompt"
	"github.com/jorgengundersen/afk/internal/quickstart"
	"github.com/jorgengundersen/afk/internal/signal"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "quickstart" {
		fmt.Print(quickstart.Text())
		os.Exit(0)
	}

	cfg, err := config.ParseFlags(os.Args[1:], os.Stdout)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}
	if err := config.Validate(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}

	logPath, err := logger.SessionPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	log := logger.New(logPath)
	defer log.Close()

	if err := harness.CheckBinary(cfg.Harness, cfg.Raw); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}

	if cfg.Beads {
		if err := beads.CheckBinaryInPath(); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(2)
		}
	}

	var ws loop.WorkSource
	if cfg.Beads {
		client := beads.NewClient(cfg.Labels, cfg.LabelsAny)
		ws = &beadsWorkSource{
			client:     &beadsClientAdapter{inner: &client},
			userPrompt: cfg.Prompt,
		}
	}

	// When not using beads, assemble the static prompt
	var assembled string
	if !cfg.Beads {
		assembled, err = prompt.Assemble(cfg.Prompt, "")
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(2)
		}
	}

	runner, err := harness.New(cfg.Harness, cfg.Model, cfg.Raw, cfg.HarnessArgs)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}

	ctx, cancel := signal.NotifyContext(context.Background())
	defer cancel()

	loopCfg := loop.Config{
		MaxIter: cfg.MaxIter,
		Daemon:  cfg.Daemon,
		Sleep:   cfg.Sleep,
		Prompt:  assembled,
	}

	exitCode, err := loop.Run(ctx, loopCfg, runner, log, ws)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
	}
	os.Exit(exitCode)
}

// issueResult is the interface-compatible issue type for the adapter.
type issueResult struct {
	ID      string
	Title   string
	RawJSON []byte
}

// readyClient abstracts the beads client for testability.
type readyClient interface {
	Ready() ([]issueResult, error)
}

// beadsClientAdapter wraps the real beads.Client to satisfy readyClient.
type beadsClientAdapter struct {
	inner *beads.Client
}

func (a *beadsClientAdapter) Ready() ([]issueResult, error) {
	issues, err := a.inner.Ready()
	if err != nil {
		return nil, err
	}
	results := make([]issueResult, len(issues))
	for i, iss := range issues {
		results[i] = issueResult{
			ID:      iss.ID,
			Title:   iss.Title,
			RawJSON: iss.RawJSON,
		}
	}
	return results, nil
}

// beadsWorkSource implements loop.WorkSource by fetching from beads and assembling prompts.
type beadsWorkSource struct {
	client     readyClient
	userPrompt string
}

func (b *beadsWorkSource) Next() (promptStr string, issueID string, issueTitle string, ok bool, err error) {
	issues, err := b.client.Ready()
	if err != nil {
		return "", "", "", false, fmt.Errorf("fetching work: %w", err)
	}
	if len(issues) == 0 {
		return "", "", "", false, nil
	}

	top := issues[0]
	assembled, err := prompt.Assemble(b.userPrompt, string(top.RawJSON))
	if err != nil {
		return "", "", "", false, fmt.Errorf("assembling prompt: %w", err)
	}

	return assembled, top.ID, top.Title, true, nil
}
