package cli_test

import (
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/jorgengundersen/afk/internal/cli"
	"github.com/jorgengundersen/afk/internal/config"
)

func TestParseAndValidate_Defaults(t *testing.T) {
	cfg, err := cli.ParseAndValidate([]string{"-prompt", "do stuff"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Prompt != "do stuff" {
		t.Errorf("Prompt = %q, want %q", cfg.Prompt, "do stuff")
	}
	if cfg.Harness != "claude" {
		t.Errorf("Harness = %q, want %q", cfg.Harness, "claude")
	}
	if cfg.MaxIterations != 20 {
		t.Errorf("MaxIterations = %d, want 20", cfg.MaxIterations)
	}
	if cfg.Mode != config.MaxIterationsMode {
		t.Errorf("Mode = %d, want MaxIterationsMode", cfg.Mode)
	}
	if cfg.BeadsEnabled {
		t.Error("BeadsEnabled should be false by default")
	}
	if cfg.Verbose {
		t.Error("Verbose should be false by default")
	}
}

func TestParseAndValidate_AllFlags(t *testing.T) {
	args := []string{
		"-n", "5",
		"-d",
		"-sleep", "30s",
		"-harness", "opencode",
		"-model", "gpt-4o",
		"-agent-flags", "--yes",
		"-prompt", "fix it",
		"-beads",
		"-beads-labels", "backend,p0",
		"-beads-instruction", "do the thing",
		"-log", "/tmp/logs",
		"-stderr",
		"-v",
		"-C", "/tmp/work",
		"--", "extra1", "extra2",
	}
	cfg, err := cli.ParseAndValidate(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.MaxIterations != 5 {
		t.Errorf("MaxIterations = %d, want 5", cfg.MaxIterations)
	}
	if cfg.Mode != config.DaemonMode {
		t.Errorf("Mode = %d, want DaemonMode", cfg.Mode)
	}
	if cfg.SleepInterval != 30*time.Second {
		t.Errorf("SleepInterval = %v, want 30s", cfg.SleepInterval)
	}
	if cfg.Harness != "opencode" {
		t.Errorf("Harness = %q, want %q", cfg.Harness, "opencode")
	}
	if cfg.Model != "gpt-4o" {
		t.Errorf("Model = %q, want %q", cfg.Model, "gpt-4o")
	}
	if cfg.AgentFlags != "--yes" {
		t.Errorf("AgentFlags = %q, want %q", cfg.AgentFlags, "--yes")
	}
	if cfg.Prompt != "fix it" {
		t.Errorf("Prompt = %q, want %q", cfg.Prompt, "fix it")
	}
	if !cfg.BeadsEnabled {
		t.Error("BeadsEnabled should be true")
	}
	if len(cfg.BeadsLabels) != 2 || cfg.BeadsLabels[0] != "backend" || cfg.BeadsLabels[1] != "p0" {
		t.Errorf("BeadsLabels = %v, want [backend p0]", cfg.BeadsLabels)
	}
	if cfg.BeadsInstruct != "do the thing" {
		t.Errorf("BeadsInstruct = %q, want %q", cfg.BeadsInstruct, "do the thing")
	}
	if cfg.LogDir != "/tmp/logs" {
		t.Errorf("LogDir = %q, want %q", cfg.LogDir, "/tmp/logs")
	}
	if !cfg.Stderr {
		t.Error("Stderr should be true")
	}
	if !cfg.Verbose {
		t.Error("Verbose should be true")
	}
	if cfg.Workdir != "/tmp/work" {
		t.Errorf("Workdir = %q, want %q", cfg.Workdir, "/tmp/work")
	}
	if len(cfg.PassthroughArgs) != 2 || cfg.PassthroughArgs[0] != "extra1" || cfg.PassthroughArgs[1] != "extra2" {
		t.Errorf("PassthroughArgs = %v, want [extra1 extra2]", cfg.PassthroughArgs)
	}
}

func TestParseAndValidate_DaemonDefaultSleep(t *testing.T) {
	cfg, err := cli.ParseAndValidate([]string{"-d", "-prompt", "go"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.SleepInterval != 60*time.Second {
		t.Errorf("SleepInterval = %v, want 60s default in daemon mode", cfg.SleepInterval)
	}
}

func TestParseAndValidate_BeadsLabelsImpliesBeads(t *testing.T) {
	cfg, err := cli.ParseAndValidate([]string{"-beads-labels", "ui", "-prompt", "go"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.BeadsEnabled {
		t.Error("BeadsEnabled should be true when beads-labels is set")
	}
}

func TestParseAndValidate_ValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "raw with harness",
			args: []string{"-raw", "echo {prompt}", "-harness", "claude", "-prompt", "go"},
		},
		{
			name: "quiet with verbose",
			args: []string{"-quiet", "-v", "-prompt", "go"},
		},
		{
			name: "no prompt no beads",
			args: []string{},
		},
		{
			name: "sleep without daemon",
			args: []string{"-sleep", "5s", "-prompt", "go"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := cli.ParseAndValidate(tt.args)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestParseAndValidate_RawCommandSkipsHarness(t *testing.T) {
	cfg, err := cli.ParseAndValidate([]string{"-raw", "echo {prompt}", "-harness", "", "-prompt", "go"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.RawCommand != "echo {prompt}" {
		t.Errorf("RawCommand = %q, want %q", cfg.RawCommand, "echo {prompt}")
	}
}

func TestParseAndValidate_StdinPrompt(t *testing.T) {
	stdin := strings.NewReader("prompt from stdin\n")
	cfg, err := cli.ParseAndValidate([]string{}, cli.WithStdin(stdin))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Prompt != "prompt from stdin" {
		t.Errorf("Prompt = %q, want %q", cfg.Prompt, "prompt from stdin")
	}
}

func TestParseAndValidate_FlagPromptOverridesStdin(t *testing.T) {
	stdin := strings.NewReader("from stdin")
	cfg, err := cli.ParseAndValidate([]string{"-prompt", "from flag"}, cli.WithStdin(stdin))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Prompt != "from flag" {
		t.Errorf("Prompt = %q, want %q", cfg.Prompt, "from flag")
	}
}

func TestSetupSignals(t *testing.T) {
	ctx := cli.SetupSignals([]os.Signal{syscall.SIGUSR1})

	select {
	case <-ctx.Done():
		t.Fatal("context should not be cancelled yet")
	default:
	}

	// Send the signal to ourselves.
	syscall.Kill(syscall.Getpid(), syscall.SIGUSR1)

	select {
	case <-ctx.Done():
		// success
	case <-time.After(time.Second):
		t.Fatal("context was not cancelled after signal")
	}
}

func TestSetupSignals_OnSignalCallback(t *testing.T) {
	var received string
	ctx := cli.SetupSignals([]os.Signal{syscall.SIGUSR1}, cli.WithOnSignal(func(sig string) {
		received = sig
	}))

	syscall.Kill(syscall.Getpid(), syscall.SIGUSR1)

	select {
	case <-ctx.Done():
		// Give callback a moment to complete.
		time.Sleep(10 * time.Millisecond)
		if received != "user defined signal 1" {
			t.Errorf("OnSignal callback received = %q, want %q", received, "user defined signal 1")
		}
	case <-time.After(time.Second):
		t.Fatal("context was not cancelled after signal")
	}
}
