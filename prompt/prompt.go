package prompt

import (
	"encoding/json"
	"strings"

	"github.com/jorgengundersen/afk/beads"
)

// Assemble builds the final prompt string from a user prompt and an optional
// beads issue. It is a pure function with no I/O or side effects.
func Assemble(userPrompt string, issue *beads.Issue) string {
	var parts []string

	if issue != nil {
		parts = append(parts, formatIssue(issue))
	}

	if userPrompt != "" {
		parts = append(parts, userPrompt)
	}

	return strings.Join(parts, "\n\n")
}

func formatIssue(issue *beads.Issue) string {
	var fields struct {
		Description        string `json:"description"`
		AcceptanceCriteria string `json:"acceptance_criteria"`
	}
	_ = json.Unmarshal(issue.Raw, &fields)

	var b strings.Builder
	b.WriteString("## Issue: ")
	b.WriteString(issue.ID)
	b.WriteString("\n\n**Title:** ")
	b.WriteString(issue.Title)

	if fields.Description != "" {
		b.WriteString("\n\n**Description:**\n")
		b.WriteString(fields.Description)
	}

	if fields.AcceptanceCriteria != "" {
		b.WriteString("\n\n**Acceptance Criteria:**\n")
		b.WriteString(fields.AcceptanceCriteria)
	}

	return b.String()
}
