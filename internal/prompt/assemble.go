package prompt

import "errors"

// Assemble returns the prompt verbatim, or an error if empty.
func Assemble(prompt string) (string, error) {
	if prompt == "" {
		return "", errors.New("no prompt provided")
	}
	return prompt, nil
}
