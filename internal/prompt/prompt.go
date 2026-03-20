package prompt

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jorgengundersen/afk/internal/beads"
)

// DefaultInstruction is the standard instruction text appended when a beads
// issue is present and no override is provided.
const DefaultInstruction = "Claim this issue and complete it. Follow AGENTS.md instructions. When complete, close the issue and exit."

// Assemble builds the final prompt string from a user prompt, an optional
// beads issue, and an instruction string. The instruction is only included
// when an issue is present. It is a pure function with no I/O or side effects.
func Assemble(userPrompt string, issue *beads.Issue, instruction string) (string, error) {
	var parts []string

	if issue != nil {
		formatted, err := formatIssue(issue)
		if err != nil {
			return "", fmt.Errorf("formatting issue %s: %w", issue.ID, err)
		}
		parts = append(parts, formatted)
		if instruction != "" {
			parts = append(parts, instruction)
		}
	}

	if userPrompt != "" {
		parts = append(parts, userPrompt)
	}

	return strings.Join(parts, "\n\n"), nil
}

func formatIssue(issue *beads.Issue) (string, error) {
	var fields struct {
		Description        string `json:"description"`
		AcceptanceCriteria string `json:"acceptance_criteria"`
	}
	if len(issue.Raw) > 0 {
		if err := json.Unmarshal(issue.Raw, &fields); err != nil {
			return "", err
		}
	}

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

	return b.String(), nil
}
