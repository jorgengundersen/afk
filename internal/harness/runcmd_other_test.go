//go:build !unix

package harness

import (
	"context"
	"errors"
	"os/exec"
	"testing"
)

func TestRunCmdExitCode0(t *testing.T) {
	cmd := exec.Command("cmd", "/c", "exit 0")
	exitCode, err := runCmd(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
}

func TestRunCmdExitCode1(t *testing.T) {
	cmd := exec.Command("cmd", "/c", "exit 1")
	exitCode, err := runCmd(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
}

func TestRunCmdBinaryNotFound(t *testing.T) {
	cmd := exec.Command("nonexistent-binary-xyz-12345")
	_, err := runCmd(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected error for missing binary")
	}
}

func TestRunCmdContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	cmd := exec.CommandContext(ctx, "cmd", "/c", "timeout /t 60")
	_, err := runCmd(ctx, cmd)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
