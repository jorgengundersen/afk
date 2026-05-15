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

	SleepExplicit       bool
	EmptySleepsExplicit bool
	FailExplicit        bool

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
		case "sleep":
			cfg.SleepExplicit = true
		case "empty-sleeps":
			cfg.EmptySleepsExplicit = true
		case "fail":
			cfg.FailExplicit = true
		}
	})

	cfg.CommandArgv = after
	return cfg, nil
}
