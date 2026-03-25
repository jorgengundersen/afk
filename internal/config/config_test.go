package config

import (
	"testing"
	"time"
)

func TestParseFlags_Defaults(t *testing.T) {
	cfg, err := ParseFlags([]string{"-p", "hi"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Prompt != "hi" {
		t.Errorf("Prompt = %q, want %q", cfg.Prompt, "hi")
	}
	if cfg.MaxIter != 20 {
		t.Errorf("MaxIter = %d, want 20", cfg.MaxIter)
	}
	if cfg.Sleep != 60*time.Second {
		t.Errorf("Sleep = %v, want 60s", cfg.Sleep)
	}
	if cfg.Harness != "claude" {
		t.Errorf("Harness = %q, want %q", cfg.Harness, "claude")
	}
	if cfg.Daemon {
		t.Error("Daemon should be false by default")
	}
	if cfg.Model != "" {
		t.Errorf("Model = %q, want empty", cfg.Model)
	}
	if cfg.Raw != "" {
		t.Errorf("Raw = %q, want empty", cfg.Raw)
	}
	if cfg.Beads {
		t.Error("Beads should be false by default")
	}
}

func TestParseFlags_AllFlags(t *testing.T) {
	args := []string{
		"-p", "do stuff",
		"-n", "5",
		"-d",
		"--sleep", "30s",
		"--harness", "aider",
		"--model", "gpt-4",
		"--raw", "my-agent {prompt}",
		"--beads",
	}
	cfg, err := ParseFlags(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Prompt != "do stuff" {
		t.Errorf("Prompt = %q, want %q", cfg.Prompt, "do stuff")
	}
	if cfg.MaxIter != 5 {
		t.Errorf("MaxIter = %d, want 5", cfg.MaxIter)
	}
	if !cfg.Daemon {
		t.Error("Daemon should be true")
	}
	if cfg.Sleep != 30*time.Second {
		t.Errorf("Sleep = %v, want 30s", cfg.Sleep)
	}
	if cfg.Harness != "aider" {
		t.Errorf("Harness = %q, want %q", cfg.Harness, "aider")
	}
	if cfg.Model != "gpt-4" {
		t.Errorf("Model = %q, want %q", cfg.Model, "gpt-4")
	}
	if cfg.Raw != "my-agent {prompt}" {
		t.Errorf("Raw = %q, want %q", cfg.Raw, "my-agent {prompt}")
	}
	if !cfg.Beads {
		t.Error("Beads should be true")
	}
	if !cfg.HarnessSet {
		t.Error("HarnessSet should be true when --harness is explicit")
	}
	if !cfg.SleepSet {
		t.Error("SleepSet should be true when --sleep is explicit")
	}
}

func TestParseFlags_UnknownFlag(t *testing.T) {
	_, err := ParseFlags([]string{"--nope"})
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
}

func TestParseFlags_BadType(t *testing.T) {
	_, err := ParseFlags([]string{"-n", "abc"})
	if err == nil {
		t.Fatal("expected error for bad type on -n")
	}
}

func TestParseFlags_SetFlags_NotSetByDefault(t *testing.T) {
	cfg, err := ParseFlags([]string{"-p", "hi"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.HarnessSet {
		t.Error("HarnessSet should be false when --harness not provided")
	}
	if cfg.SleepSet {
		t.Error("SleepSet should be false when --sleep not provided")
	}
}
