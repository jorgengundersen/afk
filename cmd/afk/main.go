package main

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"github.com/jorgengundersen/afk/internal/beads"
	"github.com/jorgengundersen/afk/internal/cli"
	"github.com/jorgengundersen/afk/internal/config"
	"github.com/jorgengundersen/afk/internal/eventlog"
	"github.com/jorgengundersen/afk/internal/harness"
	"github.com/jorgengundersen/afk/internal/loop"
	"github.com/jorgengundersen/afk/internal/terminal"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	cfg, err := cli.ParseAndValidate(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "afk: %v\n", err)
		return 2
	}

	if cfg.Workdir != "" {
		if err := os.Chdir(cfg.Workdir); err != nil {
			fmt.Fprintf(os.Stderr, "afk: %v\n", err)
			return 1
		}
	}

	var log loop.EventLogger
	var logCloser func() error
	if cfg.LogDir != "" {
		l, err := eventlog.New(cfg.LogDir, cfg.Stderr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "afk: %v\n", err)
			return 1
		}
		log = &logAdapter{l}
		logCloser = l.Close
	} else {
		log = nopLogger{}
	}
	if logCloser != nil {
		defer logCloser()
	}

	ctx := cli.SetupSignals([]os.Signal{syscall.SIGINT, syscall.SIGTERM},
		cli.WithOnSignal(func(sig string) {
			log.Event("signal-received", loop.Field{Key: "signal", Value: sig})
		}),
	)

	h, err := harness.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "afk: %v\n", err)
		return 1
	}

	var bc loop.BeadsClient
	if cfg.BeadsEnabled {
		bc = beads.NewClient(cfg.BeadsLabels)
	}

	pr := terminal.New(os.Stdout, cfg.Quiet, cfg.Verbose)

	switch cfg.Mode {
	case config.DaemonMode:
		err = loop.RunDaemon(ctx, cfg, h, bc, log, pr)
	default:
		err = loop.RunMaxIter(ctx, cfg, h, bc, log, pr)
	}

	if err != nil {
		if errors.Is(err, loop.ErrAllFailed) {
			return 1
		}
		fmt.Fprintf(os.Stderr, "afk: %v\n", err)
		return 1
	}
	return 0
}

// logAdapter wraps eventlog.Logger to satisfy loop.EventLogger,
// converting between the two Field types.
type logAdapter struct {
	l *eventlog.Logger
}

func (a *logAdapter) Event(name string, fields ...loop.Field) {
	ef := make([]eventlog.Field, len(fields))
	for i, f := range fields {
		ef[i] = eventlog.Field{Key: f.Key, Value: f.Value}
	}
	a.l.Event(name, ef...)
}

// nopLogger discards all events when no log directory is configured.
type nopLogger struct{}

func (nopLogger) Event(string, ...loop.Field) {}
