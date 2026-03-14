package docs_test

import (
	"os"
	"strings"
	"testing"
)

func TestCLIReferenceExists(t *testing.T) {
	content, err := os.ReadFile("cli-reference.md")
	if err != nil {
		t.Fatalf("cannot read docs/cli-reference.md: %v", err)
	}
	doc := string(content)

	t.Run("has_title", func(t *testing.T) {
		if !strings.Contains(doc, "# CLI Reference") {
			t.Error("missing title '# CLI Reference'")
		}
	})

	t.Run("has_synopsis", func(t *testing.T) {
		if !strings.Contains(doc, "## Synopsis") {
			t.Error("missing '## Synopsis' section")
		}
		if !strings.Contains(doc, "afk [flags]") {
			t.Error("synopsis should contain 'afk [flags]'")
		}
	})

	// Flag categories
	categories := []string{
		"Mode",
		"Harness",
		"Prompt",
		"Beads",
		"Logging",
	}
	for _, cat := range categories {
		t.Run("has_category_"+cat, func(t *testing.T) {
			if !strings.Contains(doc, cat) {
				t.Errorf("missing flag category %q", cat)
			}
		})
	}

	// All flags must be documented
	flags := []struct {
		name string
		text string
	}{
		{"n", "-n"},
		{"d", "-d"},
		{"sleep", "--sleep"},
		{"harness", "--harness"},
		{"model", "--model"},
		{"agent-flags", "--agent-flags"},
		{"raw", "--raw"},
		{"prompt", "--prompt"},
		{"beads", "--beads"},
		{"beads-labels", "--beads-labels"},
		{"beads-instruction", "--beads-instruction"},
		{"log", "--log"},
		{"stderr", "--stderr"},
		{"v", "-v"},
		{"quiet", "--quiet"},
		{"C", "-C"},
	}
	for _, f := range flags {
		t.Run("documents_flag_"+f.name, func(t *testing.T) {
			if !strings.Contains(doc, f.text) {
				t.Errorf("flag %s (%s) not documented", f.name, f.text)
			}
		})
	}

	t.Run("has_mutual_exclusion_rules", func(t *testing.T) {
		if !strings.Contains(doc, "Mutual") || !strings.Contains(doc, "xclusion") {
			t.Error("missing mutual exclusion rules section")
		}
	})

	t.Run("has_exit_codes", func(t *testing.T) {
		if !strings.Contains(doc, "Exit Code") || !strings.Contains(doc, "exit code") {
			// Accept either casing
			if !strings.Contains(strings.ToLower(doc), "exit code") {
				t.Error("missing exit codes section")
			}
		}
		for _, code := range []string{"0", "1", "2"} {
			if !strings.Contains(doc, code) {
				t.Errorf("exit code %s not documented", code)
			}
		}
	})

	t.Run("has_examples", func(t *testing.T) {
		if !strings.Contains(doc, "## Examples") {
			t.Error("missing '## Examples' section")
		}
		// Count code blocks as proxy for examples
		count := strings.Count(doc, "```")
		// At least 8 examples means at least 16 ``` markers
		if count < 16 {
			t.Errorf("expected at least 8 code block examples, found %d code block markers (need >= 16)", count)
		}
	})

	t.Run("codex_copilot_marked_planned", func(t *testing.T) {
		lower := strings.ToLower(doc)
		if !strings.Contains(lower, "codex") {
			t.Error("codex harness not mentioned")
		}
		if !strings.Contains(lower, "copilot") {
			t.Error("copilot harness not mentioned")
		}
		if !strings.Contains(lower, "planned") && !strings.Contains(lower, "coming soon") {
			t.Error("codex/copilot should be marked as planned or coming soon")
		}
	})
}
