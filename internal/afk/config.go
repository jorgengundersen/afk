package afk

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"
)

var ErrMissingSeparator = errors.New("missing required -- separator")

// Config is the parsed, pre-validation CLI configuration.
type Config struct {
	Help bool

	Loops        int
	Daemon       bool
	Items        string
	ItemsCmd     string
	Sleep        time.Duration
	EmptySleeps  int
	Fail         string
	UntilSuccess bool
	Timeout      time.Duration

	LoopsExplicit       bool
	ItemsExplicit       bool
	ItemsCmdExplicit    bool
	SleepExplicit       bool
	EmptySleepsExplicit bool
	FailExplicit        bool
	TimeoutExplicit     bool

	CommandArgv []string
}

func ParseArgs(args []string) (Config, error) {
	if wantsHelp(args) {
		return Config{Help: true}, nil
	}

	separator := -1
	for i, arg := range args {
		if arg == "--" {
			separator = i
			break
		}
	}
	if separator == -1 {
		return Config{}, ErrMissingSeparator
	}

	before := args[:separator]
	after := append([]string(nil), args[separator+1:]...)

	fs := flag.NewFlagSet("afk", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	cfg := Config{
		Sleep:       60 * time.Second,
		EmptySleeps: 0,
		Fail:        "continue",
	}

	fs.IntVar(&cfg.Loops, "loops", 0, "")
	fs.IntVar(&cfg.Loops, "n", 0, "")
	fs.BoolVar(&cfg.Daemon, "daemon", false, "")
	fs.BoolVar(&cfg.Daemon, "d", false, "")
	fs.StringVar(&cfg.Items, "items", "", "")
	fs.StringVar(&cfg.ItemsCmd, "items-cmd", "", "")
	fs.DurationVar(&cfg.Sleep, "sleep", cfg.Sleep, "")
	fs.IntVar(&cfg.EmptySleeps, "empty-sleeps", cfg.EmptySleeps, "")
	fs.StringVar(&cfg.Fail, "fail", cfg.Fail, "")
	fs.BoolVar(&cfg.UntilSuccess, "until-success", false, "")
	fs.DurationVar(&cfg.Timeout, "timeout", 0, "")

	if err := fs.Parse(before); err != nil {
		return Config{}, err
	}
	if fs.NArg() > 0 {
		return Config{}, fmt.Errorf("unexpected positional afk arguments: %s", strings.Join(fs.Args(), " "))
	}

	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "loops", "n":
			cfg.LoopsExplicit = true
		case "items":
			cfg.ItemsExplicit = true
		case "items-cmd":
			cfg.ItemsCmdExplicit = true
		case "sleep":
			cfg.SleepExplicit = true
		case "empty-sleeps":
			cfg.EmptySleepsExplicit = true
		case "fail":
			cfg.FailExplicit = true
		case "timeout":
			cfg.TimeoutExplicit = true
		}
	})

	cfg.CommandArgv = after
	return cfg, nil
}

func ValidateConfig(cfg Config) error {
	if len(cfg.CommandArgv) == 0 {
		return errors.New("missing main child command after --")
	}

	hasItems := cfg.ItemsExplicit
	hasItemsCmd := cfg.ItemsCmdExplicit
	hasLoops := cfg.LoopsExplicit
	hasDriver := hasLoops || cfg.Daemon || hasItems || hasItemsCmd
	if !hasDriver {
		return errors.New("at least one loop driver is required: --loops, --daemon, --items, or --items-cmd")
	}

	if hasLoops && cfg.Loops <= 0 {
		return errors.New("--loops must be a positive integer")
	}

	if hasLoops && cfg.Daemon {
		return errors.New("--loops and --daemon are mutually exclusive")
	}
	if hasLoops && (hasItems || hasItemsCmd) {
		return errors.New("--loops and item sources are mutually exclusive")
	}
	if hasItems && hasItemsCmd {
		return errors.New("--items and --items-cmd are mutually exclusive")
	}
	if cfg.Daemon && hasItems {
		return errors.New("--daemon and --items are mutually exclusive")
	}

	if cfg.SleepExplicit {
		if !hasItemsCmd {
			return errors.New("explicit --sleep requires --items-cmd")
		}
		if cfg.Sleep < 0 {
			return errors.New("--sleep must be non-negative")
		}
	}

	if cfg.EmptySleepsExplicit {
		if !hasItemsCmd {
			return errors.New("explicit --empty-sleeps requires --items-cmd")
		}
		if cfg.EmptySleeps < 0 {
			return errors.New("--empty-sleeps must be non-negative")
		}
	}

	failValue := cfg.Fail
	if failValue == "" {
		failValue = "continue"
	}
	if failValue != "continue" && failValue != "stop" {
		return errors.New("--fail must be continue or stop")
	}

	if cfg.UntilSuccess {
		if cfg.FailExplicit {
			return errors.New("--until-success is invalid with explicit --fail")
		}
		if hasItems || hasItemsCmd {
			return errors.New("--until-success is invalid with item sources")
		}
		if !hasLoops && !cfg.Daemon {
			return errors.New("--until-success requires --loops or --daemon")
		}
	}

	if cfg.TimeoutExplicit && cfg.Timeout <= 0 {
		return errors.New("--timeout must be positive when set")
	}
	if cfg.Timeout < 0 {
		return errors.New("--timeout must be positive when set")
	}

	return nil
}
