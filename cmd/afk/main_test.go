package main

import (
	"os/exec"
	"strings"
	"testing"
)

func build(t *testing.T) string {
	t.Helper()
	bin := t.TempDir() + "/afk"
	out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %s\n%s", err, out)
	}
	return bin
}

func TestPrintFlag(t *testing.T) {
	bin := build(t)

	cmd := exec.Command(bin, "-p", "hello world")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("expected exit 0, got error: %v", err)
	}

	got := strings.TrimRight(string(out), "\n")
	if got != "hello world" {
		t.Errorf("stdout = %q, want %q", got, "hello world")
	}
}
