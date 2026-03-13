package prompt_test

import (
	"encoding/json"
	"testing"

	"github.com/jorgengundersen/afk/beads"
	"github.com/jorgengundersen/afk/prompt"
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
		name      string
		prompt    string
		issue     *beads.Issue
		wantEmpty bool
		contains  []string
		absent    []string
	}{
		{
			name:   "prompt and issue",
			prompt: "fix the bug",
			issue:  makeIssue("TST-1", "Fix login", "Login is broken", "- Users can log in"),
			contains: []string{
				"TST-1",
				"Fix login",
				"Login is broken",
				"Users can log in",
				"fix the bug",
			},
		},
		{
			name:   "prompt only",
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
			name:   "issue only",
			prompt: "",
			issue:  makeIssue("TST-2", "Add tests", "Need more tests", "- Coverage > 80%"),
			contains: []string{
				"TST-2",
				"Add tests",
				"Need more tests",
				"Coverage > 80%",
			},
		},
		{
			name:      "both empty",
			prompt:    "",
			issue:     nil,
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := prompt.Assemble(tt.prompt, tt.issue)

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
