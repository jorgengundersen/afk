// Package harness provides the agent harness abstraction and implementations.
package harness

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/jorgengundersen/afk/config"
)

// Harness runs an external agent CLI with a given prompt.
type Harness interface {
	Run(ctx context.Context, prompt string) (exitCode int, err error)
}

// binaryName maps harness names to their CLI binary names.
var binaryName = map[string]string{
	"claude":   "claude",
	"opencode": "opencode",
}

// New returns the appropriate harness implementation based on config.
// It checks that the required binary exists in PATH.
func New(cfg config.Config) (Harness, error) {
	bin, ok := binaryName[cfg.Harness]
	if !ok {
		return nil, fmt.Errorf("unknown harness %q", cfg.Harness)
	}

	if _, err := exec.LookPath(bin); err != nil {
		return nil, fmt.Errorf("harness %q: binary %q not found in PATH", cfg.Harness, bin)
	}

	switch cfg.Harness {
	case "claude":
		return newClaude(cfg), nil
	case "opencode":
		return newOpenCode(cfg), nil
	default:
		return nil, fmt.Errorf("unknown harness %q", cfg.Harness)
	}
}
