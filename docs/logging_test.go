package docs_test

import (
	"os"
	"strings"
	"testing"
)

func TestLoggingDocExists(t *testing.T) {
	content, err := os.ReadFile("logging.md")
	if err != nil {
		t.Fatalf("cannot read docs/logging.md: %v", err)
	}
	doc := string(content)

	t.Run("has_title", func(t *testing.T) {
		if !strings.Contains(doc, "# Logging") {
			t.Error("missing title '# Logging'")
		}
	})

	t.Run("has_log_file_location", func(t *testing.T) {
		if !strings.Contains(doc, "~/.local/share/afk/logs/") {
			t.Error("missing default log directory path")
		}
		if !strings.Contains(doc, "afk-") && !strings.Contains(doc, "YYYYMMDD") {
			t.Error("missing log file naming convention")
		}
	})

	t.Run("has_structured_format", func(t *testing.T) {
		if !strings.Contains(doc, "[afk]") {
			t.Error("missing [afk] tag in format description")
		}
		if !strings.Contains(doc, "key=value") {
			t.Error("missing key=value format description")
		}
	})

	t.Run("has_event_types_table", func(t *testing.T) {
		events := []string{
			"session-start",
			"session-end",
			"iteration-start",
			"iteration-end",
			"beads-check",
			"sleeping",
			"waking",
			"signal-received",
		}
		for _, ev := range events {
			if !strings.Contains(doc, ev) {
				t.Errorf("missing event type %q in documentation", ev)
			}
		}
	})

	t.Run("has_example_session_logs", func(t *testing.T) {
		// Should have example logs for both modes
		lower := strings.ToLower(doc)
		if !strings.Contains(lower, "max-iterations") && !strings.Contains(lower, "max iterations") {
			t.Error("missing max-iterations mode example")
		}
		if !strings.Contains(lower, "daemon") {
			t.Error("missing daemon mode example")
		}
		// Should contain actual log line examples
		count := strings.Count(doc, "[afk]")
		if count < 5 {
			t.Errorf("expected at least 5 example log lines containing '[afk]', found %d", count)
		}
	})

	t.Run("has_terminal_output_modes", func(t *testing.T) {
		modes := []string{"--quiet", "-v", "--stderr"}
		for _, mode := range modes {
			if !strings.Contains(doc, mode) {
				t.Errorf("missing terminal output mode %q", mode)
			}
		}
	})

	t.Run("has_no_levels_explanation", func(t *testing.T) {
		lower := strings.ToLower(doc)
		if !strings.Contains(lower, "no") || !strings.Contains(lower, "level") {
			t.Error("should explain that there are no log levels")
		}
	})
}
