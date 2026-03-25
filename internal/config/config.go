package config

import (
	"flag"
	"io"
	"time"
)

type Config struct {
	Prompt  string
	MaxIter int
	Daemon  bool
	Sleep   time.Duration
	Harness string
	Model   string
	Raw     string
	Beads   bool

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
