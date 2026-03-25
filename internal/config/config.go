package config

import (
	"errors"
	"flag"
	"io"
	"time"
)

type Config struct {
	Prompt      string
	MaxIter     int
	Daemon      bool
	Sleep       time.Duration
	Harness     string
	Model       string
	Raw         string
	HarnessArgs string
	Beads       bool

	HarnessSet bool
	SleepSet   bool
}

func ParseFlags(args []string) (Config, error) {
	var cfg Config

	fs := flag.NewFlagSet("afk", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&cfg.Prompt, "p", "", "prompt")
	fs.IntVar(&cfg.MaxIter, "n", 20, "max iterations")
	fs.BoolVar(&cfg.Daemon, "d", false, "daemon mode")
	fs.DurationVar(&cfg.Sleep, "sleep", 60*time.Second, "sleep between iterations")
	fs.StringVar(&cfg.Harness, "harness", "claude", "harness to use")
	fs.StringVar(&cfg.Model, "model", "", "model override")
	fs.StringVar(&cfg.Raw, "raw", "", "raw command template")
	fs.StringVar(&cfg.HarnessArgs, "harness-args", "", "extra arguments for the harness subprocess")
	fs.BoolVar(&cfg.Beads, "beads", false, "use beads for issue tracking")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "harness":
			cfg.HarnessSet = true
		case "sleep":
			cfg.SleepSet = true
		}
	})

	return cfg, nil
}

func Validate(cfg Config) error {
	if cfg.Raw != "" && (cfg.HarnessSet || cfg.Model != "") {
		return errors.New("--raw cannot be combined with --harness or --model")
	}
	if cfg.SleepSet && !cfg.Daemon {
		return errors.New("--sleep requires daemon mode (-d)")
	}
	if cfg.Prompt == "" && !cfg.Beads {
		return errors.New("no prompt provided and beads not active; nothing to do")
	}
	return nil
}
