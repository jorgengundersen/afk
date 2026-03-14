package docs_test

import (
	"os"
	"strings"
	"testing"
)

func TestHarnessesDocExists(t *testing.T) {
	content, err := os.ReadFile("harnesses.md")
	if err != nil {
		t.Fatalf("cannot read docs/harnesses.md: %v", err)
	}
	doc := string(content)

	t.Run("has_title", func(t *testing.T) {
		if !strings.Contains(doc, "# Harnesses") {
			t.Error("missing title '# Harnesses'")
		}
	})

	t.Run("has_overview", func(t *testing.T) {
		if !strings.Contains(doc, "## Overview") {
			t.Error("missing '## Overview' section")
		}
		if !strings.Contains(doc, "harness") && !strings.Contains(doc, "Harness") {
			t.Error("overview should mention the harness abstraction")
		}
	})

	// Implemented harnesses
	t.Run("documents_claude", func(t *testing.T) {
		lower := strings.ToLower(doc)
		if !strings.Contains(lower, "claude") {
			t.Error("claude harness not documented")
		}
		if !strings.Contains(doc, "--dangerously-skip-permissions") {
			t.Error("claude section should mention --dangerously-skip-permissions flag")
		}
		if !strings.Contains(doc, "-p") {
			t.Error("claude section should mention -p flag for prompt")
		}
	})

	t.Run("documents_opencode", func(t *testing.T) {
		lower := strings.ToLower(doc)
		if !strings.Contains(lower, "opencode") {
			t.Error("opencode harness not documented")
		}
		if !strings.Contains(doc, "--yes") {
			t.Error("opencode section should mention --yes flag")
		}
	})

	t.Run("documents_model_flag", func(t *testing.T) {
		if !strings.Contains(doc, "--model") {
			t.Error("should document how --model applies")
		}
	})

	t.Run("documents_agent_flags", func(t *testing.T) {
		if !strings.Contains(doc, "--agent-flags") {
			t.Error("should document how --agent-flags applies")
		}
	})

	// Planned harnesses
	t.Run("codex_copilot_listed_as_planned", func(t *testing.T) {
		lower := strings.ToLower(doc)
		if !strings.Contains(lower, "codex") {
			t.Error("codex not mentioned")
		}
		if !strings.Contains(lower, "copilot") {
			t.Error("copilot not mentioned")
		}
		if !strings.Contains(lower, "planned") && !strings.Contains(lower, "coming soon") {
			t.Error("codex/copilot should be marked as planned or coming soon")
		}
	})

	// Raw command mode
	t.Run("documents_raw_command_mode", func(t *testing.T) {
		if !strings.Contains(doc, "--raw") {
			t.Error("should document --raw flag")
		}
		if !strings.Contains(doc, "{prompt}") {
			t.Error("should document {prompt} placeholder substitution")
		}
	})

	t.Run("raw_has_examples", func(t *testing.T) {
		lower := strings.ToLower(doc)
		if !strings.Contains(lower, "aider") {
			t.Error("should include aider as an example of raw command usage")
		}
	})

	// AGENTS.md mention
	t.Run("mentions_agents_md", func(t *testing.T) {
		if !strings.Contains(doc, "AGENTS.md") {
			t.Error("should mention AGENTS.md for steering agent behavior")
		}
	})

	// Should have code examples
	t.Run("has_code_examples", func(t *testing.T) {
		count := strings.Count(doc, "```")
		if count < 4 {
			t.Errorf("expected at least 2 code block examples, found %d markers (need >= 4)", count)
		}
	})
}
