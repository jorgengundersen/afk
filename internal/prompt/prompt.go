package prompt

import (
	"encoding/json"
	"strings"

	"github.com/jorgengundersen/afk/internal/beads"
)

// DefaultInstruction is the standard instruction text appended when a beads
// issue is present and no override is provided.
const DefaultInstruction = "Claim this issue and complete it. Follow AGENTS.md instructions. When complete, close the issue and exit."

// Assemble builds the final prompt string from a user prompt, an optional
// beads issue, and an instruction string. The instruction is only included
// when an issue is present. It is a pure function with no I/O or side effects.
func Assemble(userPrompt string, issue *beads.Issue, instruction string) string {
	var parts []string

	if issue != nil {
		parts = append(parts, formatIssue(issue))
		if instruction != "" {
			parts = append(parts, instruction)
		}
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
