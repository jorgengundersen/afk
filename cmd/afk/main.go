package main

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"github.com/jorgengundersen/afk/beads"
	"github.com/jorgengundersen/afk/cli"
	"github.com/jorgengundersen/afk/config"
	"github.com/jorgengundersen/afk/eventlog"
	"github.com/jorgengundersen/afk/harness"
	"github.com/jorgengundersen/afk/loop"
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

	ctx := cli.SetupSignals(syscall.SIGINT, syscall.SIGTERM)

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

	h, err := harness.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "afk: %v\n", err)
		return 1
	}

	var bc loop.BeadsClient
	if cfg.BeadsEnabled {
		bc = beads.NewClient(cfg.BeadsLabels)
	}

	switch cfg.Mode {
	case config.DaemonMode:
		err = loop.RunDaemon(ctx, cfg, h, bc, log)
	default:
		err = loop.RunMaxIter(ctx, cfg, h, bc, log)
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
