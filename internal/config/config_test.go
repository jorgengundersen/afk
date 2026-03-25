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

func TestValidate_ValidWithPrompt(t *testing.T) {
	cfg := Config{Prompt: "x"}
	if err := Validate(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_SleepWithoutDaemon(t *testing.T) {
	cfg := Config{Prompt: "x", SleepSet: true}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for --sleep without -d")
	}
	want := "--sleep requires daemon mode (-d)"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestValidate_RawWithModel(t *testing.T) {
	cfg := Config{Raw: "cmd", Model: "gpt-4", Prompt: "x"}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for --raw with --model")
	}
	want := "--raw cannot be combined with --harness or --model"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestValidate_RawWithHarness(t *testing.T) {
	cfg := Config{Raw: "cmd", HarnessSet: true, Prompt: "x"}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for --raw with --harness")
	}
	want := "--raw cannot be combined with --harness or --model"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestValidate_NoPromptNoBeads(t *testing.T) {
	cfg := Config{}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for no prompt and no beads")
	}
	want := "no prompt provided and beads not active; nothing to do"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestValidate_ValidWithBeads(t *testing.T) {
	cfg := Config{Beads: true}
	if err := Validate(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
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
