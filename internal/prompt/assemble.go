package prompt

import (
	"errors"
	"strings"
)

const issueInstruction = `You have been assigned the following issue. Claim it before starting work and close it when done.

Issue:
`

// Assemble composes the final prompt from a user prompt and optional issue JSON.
func Assemble(userPrompt, issueJSON string) (string, error) {
	userPrompt = strings.TrimSpace(userPrompt)
	hasPrompt := userPrompt != ""
	hasIssue := issueJSON != ""

	if !hasPrompt && !hasIssue {
		return "", errors.New("no prompt provided")
	}

	var b strings.Builder
	if hasIssue {
		b.WriteString(issueInstruction)
		b.WriteString(issueJSON)
	}
	if hasPrompt {
		if hasIssue {
			b.WriteString("\n\n")
		}
		b.WriteString(userPrompt)
	}

	return b.String(), nil
}
