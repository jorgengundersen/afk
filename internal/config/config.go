// Package config defines the configuration types and validation for afk.
package config

import (
	"fmt"
	"time"
)

// Mode represents the runtime mode of afk.
type Mode int

const (
	// MaxIterationsMode runs the loop up to a fixed number of times.
	MaxIterationsMode Mode = iota
	// DaemonMode runs the loop indefinitely.
	DaemonMode
)

// KnownHarnesses lists the harness names afk supports.
var KnownHarnesses = map[string]bool{
	"claude":   true,
	"opencode": true,
	"codex":    true,
	"copilot":  true,
}

// Config holds all configuration for an afk session.
type Config struct {
	Mode            Mode
	MaxIterations   int
	SleepInterval   time.Duration
	Harness         string
	Model           string
	AgentFlags      string
	RawCommand      string
	Prompt          string
	BeadsEnabled    bool
	BeadsLabels     []string
	BeadsInstruct   string
	LogDir          string
	Stderr          bool
	Verbose         bool
	Quiet           bool
	Workdir         string
	PassthroughArgs []string
}

// Validate checks all fail-fast conditions from the product spec.
// It is a pure method and does not touch the file system or PATH.
func (c Config) Validate() error {
	if c.RawCommand != "" && (c.Harness != "" || c.Model != "" || c.AgentFlags != "") {
		return fmt.Errorf("--raw cannot be combined with --harness, --model, or --agent-flags")
	}

	if c.Quiet && (c.Stderr || c.Verbose) {
		return fmt.Errorf("--quiet cannot be combined with --stderr or --verbose")
	}

	if c.Prompt == "" && !c.BeadsEnabled {
		return fmt.Errorf("no prompt provided and beads integration is not active; nothing to do")
	}

	if c.SleepInterval > 0 && c.Mode != DaemonMode {
		return fmt.Errorf("--sleep requires --daemon mode")
	}

	if c.RawCommand == "" {
		if c.Harness == "" {
			return fmt.Errorf("harness is required when --raw is not set")
		}
		if !KnownHarnesses[c.Harness] {
			return fmt.Errorf("unknown harness %q", c.Harness)
		}
	}

	return nil
}
