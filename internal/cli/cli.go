package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/jorgengundersen/afk/internal/config"
)

// Option configures ParseAndValidate behavior.
type Option func(*options)

type options struct {
	stdin io.Reader
}

// WithStdin sets the reader used when -prompt is empty.
// When nil (default), os.Stdin is used and checked for pipe mode.
func WithStdin(r io.Reader) Option {
	return func(o *options) { o.stdin = r }
}

// ParseAndValidate parses CLI flags from args and returns a validated Config.
// It uses the stdlib flag package. The args slice should not include the
// program name (i.e., pass os.Args[1:]).
func ParseAndValidate(args []string, opts ...Option) (config.Config, error) {
	o := &options{}
	for _, fn := range opts {
		fn(o)
	}

	fs := flag.NewFlagSet("afk", flag.ContinueOnError)

	var cfg config.Config

	// Mode flags
	fs.IntVar(&cfg.MaxIterations, "n", 20, "Max loop iterations")
	daemon := fs.Bool("d", false, "Daemon mode (loop indefinitely)")
	sleep := fs.Duration("sleep", 0, "Sleep interval in daemon mode")

	// Harness flags
	fs.StringVar(&cfg.Harness, "harness", "claude", "Harness: claude, opencode, codex, copilot")
	fs.StringVar(&cfg.Model, "model", "", "Model to pass to the harness")
	fs.StringVar(&cfg.AgentFlags, "agent-flags", "", "Extra flags passed to the harness CLI")
	fs.StringVar(&cfg.RawCommand, "raw", "", "Raw command string")

	// Prompt
	fs.StringVar(&cfg.Prompt, "prompt", "", "Prompt text for the agent")

	// Beads
	fs.BoolVar(&cfg.BeadsEnabled, "beads", false, "Enable beads integration")
	beadsLabels := fs.String("beads-labels", "", "Comma-separated label filter (implies --beads)")
	fs.StringVar(&cfg.BeadsInstruct, "beads-instruction", "", "Override beads instruction text")

	// Logging
	fs.StringVar(&cfg.LogDir, "log", "", "Directory for session log files")
	fs.BoolVar(&cfg.Stderr, "stderr", false, "Mirror log output to stderr")
	fs.BoolVar(&cfg.Verbose, "v", false, "Increased verbosity")
	fs.BoolVar(&cfg.Quiet, "quiet", false, "Suppress all output except errors")

	// Workdir
	fs.StringVar(&cfg.Workdir, "C", "", "Change to directory before running")

	if err := fs.Parse(args); err != nil {
		return config.Config{}, fmt.Errorf("usage error: %w", err)
	}

	// Passthrough args after --
	cfg.PassthroughArgs = fs.Args()

	// Daemon mode
	if *daemon {
		cfg.Mode = config.DaemonMode
		if *sleep == 0 {
			cfg.SleepInterval = 60 * time.Second
		}
	}
	if *sleep > 0 {
		cfg.SleepInterval = *sleep
	}

	// Beads labels implies beads
	if *beadsLabels != "" {
		cfg.BeadsEnabled = true
		cfg.BeadsLabels = splitLabels(*beadsLabels)
	}

	// Read prompt from stdin when -prompt is empty
	if cfg.Prompt == "" {
		prompt, err := readStdin(o.stdin)
		if err != nil {
			return config.Config{}, fmt.Errorf("reading stdin: %w", err)
		}
		cfg.Prompt = prompt
	}

	if err := cfg.Validate(); err != nil {
		return config.Config{}, err
	}

	return cfg, nil
}

// readStdin reads all of r if non-nil, or reads os.Stdin if it is a pipe.
func readStdin(r io.Reader) (string, error) {
	if r != nil {
		data, err := io.ReadAll(r)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(data)), nil
	}
	// Check if os.Stdin is a pipe (not a terminal).
	info, err := os.Stdin.Stat()
	if err != nil {
		return "", nil
	}
	if info.Mode()&os.ModeCharDevice != 0 {
		// stdin is a terminal, not a pipe
		return "", nil
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// SignalOption configures SetupSignals behavior.
type SignalOption func(*signalOptions)

type signalOptions struct {
	onSignal func(string)
}

// WithOnSignal registers a callback that is called with the signal name
// when a signal is received, before the context is cancelled.
func WithOnSignal(fn func(string)) SignalOption {
	return func(o *signalOptions) { o.onSignal = fn }
}

// SetupSignals returns a context that is cancelled when any of the given
// signals are received.
func SetupSignals(signals []os.Signal, opts ...SignalOption) context.Context {
	o := &signalOptions{}
	for _, fn := range opts {
		fn(o)
	}

	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, signals...)
	go func() {
		sig := <-ch
		if o.onSignal != nil {
			o.onSignal(sig.String())
		}
		cancel()
	}()
	return ctx
}

func splitLabels(s string) []string {
	var labels []string
	for _, l := range splitComma(s) {
		if l != "" {
			labels = append(labels, l)
		}
	}
	return labels
}

func splitComma(s string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}
