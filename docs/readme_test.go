package docs_test

import (
	"os"
	"strings"
	"testing"
)

func TestREADMEExists(t *testing.T) {
	content, err := os.ReadFile("../README.md")
	if err != nil {
		t.Fatalf("cannot read README.md: %v", err)
	}
	doc := string(content)

	t.Run("has_title", func(t *testing.T) {
		if !strings.Contains(doc, "# afk") {
			t.Error("missing title '# afk'")
		}
	})

	t.Run("has_tagline", func(t *testing.T) {
		lower := strings.ToLower(doc)
		if !strings.Contains(lower, "agentic coding loop") || !strings.Contains(lower, "unattended") {
			t.Error("missing tagline about running agentic coding loops unattended")
		}
	})

	t.Run("has_install_section", func(t *testing.T) {
		if !strings.Contains(doc, "## Install") {
			t.Error("missing '## Install' section")
		}
		if !strings.Contains(doc, "go install") {
			t.Error("install section should contain 'go install' command")
		}
		if !strings.Contains(doc, "github.com/jorgengundersen/afk") {
			t.Error("install section should contain module path")
		}
	})

	t.Run("has_prerequisites", func(t *testing.T) {
		lower := strings.ToLower(doc)
		if !strings.Contains(lower, "prerequisite") && !strings.Contains(lower, "require") {
			t.Error("should mention prerequisites or requirements")
		}
		if !strings.Contains(lower, "go") {
			t.Error("should mention Go as a prerequisite")
		}
	})

	t.Run("has_quick_start_section", func(t *testing.T) {
		if !strings.Contains(doc, "## Quick Start") {
			t.Error("missing '## Quick Start' section")
		}
		// Should have at least 3 examples
		quickIdx := strings.Index(doc, "## Quick Start")
		howIdx := strings.Index(doc, "## How")
		if quickIdx < 0 || howIdx < 0 {
			t.Fatal("cannot find Quick Start or How It Works sections")
		}
		quickSection := doc[quickIdx:howIdx]
		codeBlocks := strings.Count(quickSection, "```")
		if codeBlocks < 6 { // 3 examples = 6 markers
			t.Errorf("Quick Start should have at least 3 code examples, found %d markers (need >= 6)", codeBlocks)
		}
	})

	t.Run("has_basic_prompt_example", func(t *testing.T) {
		if !strings.Contains(doc, "afk -p") || !strings.Contains(doc, "--prompt") {
			t.Error("Quick Start should include a basic prompt example")
		}
	})

	t.Run("has_daemon_example", func(t *testing.T) {
		if !strings.Contains(doc, "-d") {
			t.Error("Quick Start should include a daemon mode example")
		}
	})

	t.Run("has_beads_example", func(t *testing.T) {
		if !strings.Contains(doc, "--beads") {
			t.Error("Quick Start should include a beads example")
		}
	})

	t.Run("has_how_it_works_section", func(t *testing.T) {
		if !strings.Contains(doc, "## How It Works") {
			t.Error("missing '## How It Works' section")
		}
	})

	t.Run("how_it_works_mentions_loop", func(t *testing.T) {
		lower := strings.ToLower(doc)
		if !strings.Contains(lower, "loop") {
			t.Error("How It Works should mention the loop concept")
		}
	})

	t.Run("how_it_works_mentions_modes", func(t *testing.T) {
		lower := strings.ToLower(doc)
		if !strings.Contains(lower, "max-iterations") && !strings.Contains(lower, "max iterations") {
			t.Error("How It Works should mention max-iterations mode")
		}
		if !strings.Contains(lower, "daemon") {
			t.Error("How It Works should mention daemon mode")
		}
	})

	t.Run("how_it_works_mentions_beads", func(t *testing.T) {
		lower := strings.ToLower(doc)
		if !strings.Contains(lower, "bead") {
			t.Error("How It Works should mention beads")
		}
	})

	t.Run("has_supported_agents_section", func(t *testing.T) {
		if !strings.Contains(doc, "## Supported Agents") {
			t.Error("missing '## Supported Agents' section")
		}
	})

	t.Run("agents_table_has_implemented", func(t *testing.T) {
		lower := strings.ToLower(doc)
		if !strings.Contains(lower, "claude") {
			t.Error("should list claude agent")
		}
		if !strings.Contains(lower, "opencode") {
			t.Error("should list opencode agent")
		}
		if !strings.Contains(lower, "raw") {
			t.Error("should list raw command mode")
		}
	})

	t.Run("agents_table_has_planned", func(t *testing.T) {
		lower := strings.ToLower(doc)
		if !strings.Contains(lower, "codex") {
			t.Error("should list codex agent")
		}
		if !strings.Contains(lower, "copilot") {
			t.Error("should list copilot agent")
		}
		if !strings.Contains(lower, "planned") {
			t.Error("codex and copilot should be marked as planned")
		}
	})

	t.Run("has_documentation_section", func(t *testing.T) {
		if !strings.Contains(doc, "## Documentation") {
			t.Error("missing '## Documentation' section")
		}
	})

	t.Run("links_to_docs_files", func(t *testing.T) {
		docs := []string{
			"docs/cli-reference.md",
			"docs/user-guide.md",
			"docs/harnesses.md",
			"docs/logging.md",
		}
		for _, d := range docs {
			if !strings.Contains(doc, d) {
				t.Errorf("should link to %s", d)
			}
		}
	})

	t.Run("has_license_section", func(t *testing.T) {
		if !strings.Contains(doc, "## License") {
			t.Error("missing '## License' section")
		}
		if !strings.Contains(doc, "GPL-3.0") {
			t.Error("should mention GPL-3.0 license")
		}
	})

	t.Run("line_count_in_range", func(t *testing.T) {
		lines := strings.Count(doc, "\n") + 1
		if lines < 80 {
			t.Errorf("README too short: %d lines (expected ~100-120)", lines)
		}
		if lines > 150 {
			t.Errorf("README too long: %d lines (expected ~100-120)", lines)
		}
	})
}
