package prompt_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jorgengundersen/afk/internal/beads"
	"github.com/jorgengundersen/afk/internal/prompt"
)

func makeIssue(id, title, desc, ac string) *beads.Issue {
	raw, _ := json.Marshal(map[string]string{
		"id":                  id,
		"title":               title,
		"description":         desc,
		"acceptance_criteria": ac,
	})
	return &beads.Issue{
		ID:    id,
		Title: title,
		Raw:   raw,
	}
}

func TestAssemble(t *testing.T) {
	tests := []struct {
		name        string
		prompt      string
		issue       *beads.Issue
		instruction string
		wantEmpty   bool
		contains    []string
		absent      []string
	}{
		{
			name:        "prompt and issue with instruction",
			prompt:      "fix the bug",
			issue:       makeIssue("TST-1", "Fix login", "Login is broken", "- Users can log in"),
			instruction: "do it now",
			contains: []string{
				"TST-1",
				"Fix login",
				"Login is broken",
				"Users can log in",
				"fix the bug",
				"do it now",
			},
		},
		{
			name:   "prompt only no instruction",
			prompt: "refactor auth",
			issue:  nil,
			contains: []string{
				"refactor auth",
			},
			absent: []string{
				"TST-1",
			},
		},
		{
			name:        "issue only with instruction",
			prompt:      "",
			issue:       makeIssue("TST-2", "Add tests", "Need more tests", "- Coverage > 80%"),
			instruction: "follow the plan",
			contains: []string{
				"TST-2",
				"Add tests",
				"Need more tests",
				"Coverage > 80%",
				"follow the plan",
			},
		},
		{
			name:      "all empty",
			prompt:    "",
			issue:     nil,
			wantEmpty: true,
		},
		{
			name:        "instruction without issue is ignored",
			prompt:      "just a prompt",
			issue:       nil,
			instruction: "should not appear",
			contains: []string{
				"just a prompt",
			},
			absent: []string{
				"should not appear",
			},
		},
		{
			name:   "issue with empty instruction omits instruction",
			prompt: "",
			issue:  makeIssue("TST-3", "Bug", "desc", "ac"),
			contains: []string{
				"TST-3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := prompt.Assemble(tt.prompt, tt.issue, tt.instruction)

			if tt.wantEmpty {
				if got != "" {
					t.Errorf("Assemble() = %q, want empty", got)
				}
				return
			}

			for _, s := range tt.contains {
				if !containsStr(got, s) {
					t.Errorf("Assemble() missing %q in:\n%s", s, got)
				}
			}
			for _, s := range tt.absent {
				if containsStr(got, s) {
					t.Errorf("Assemble() should not contain %q in:\n%s", s, got)
				}
			}
		})
	}
}

func TestAssemble_InstructionAfterIssue(t *testing.T) {
	issue := makeIssue("TST-1", "Fix", "broken", "criteria")
	got := prompt.Assemble("", issue, "do the work")

	issueIdx := strings.Index(got, "TST-1")
	instrIdx := strings.Index(got, "do the work")

	if issueIdx < 0 || instrIdx < 0 {
		t.Fatalf("expected both issue and instruction in output:\n%s", got)
	}
	if instrIdx < issueIdx {
		t.Errorf("instruction should appear after issue block, got issue at %d, instruction at %d", issueIdx, instrIdx)
	}
}

func containsStr(haystack, needle string) bool {
	return len(haystack) >= len(needle) && searchStr(haystack, needle)
}

func searchStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
