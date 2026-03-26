package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jorgengundersen/afk/internal/config"
	"github.com/jorgengundersen/afk/internal/harness"
	"github.com/jorgengundersen/afk/internal/logger"
	"github.com/jorgengundersen/afk/internal/loop"
	"github.com/jorgengundersen/afk/internal/prompt"
	"github.com/jorgengundersen/afk/internal/signal"
)

func main() {
	cfg, err := config.ParseFlags(os.Args[1:])
	if err != nil {
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

	assembled, err := prompt.Assemble(cfg.Prompt, "")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}

	if err := harness.CheckBinary(cfg.Harness, cfg.Raw); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
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

	exitCode, err := loop.Run(ctx, loopCfg, runner, log, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
	}
	os.Exit(exitCode)
}
