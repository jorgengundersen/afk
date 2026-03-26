package beads

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

type Issue struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Status      string          `json:"status"`
	Priority    int             `json:"priority"`
	IssueType   string          `json:"issue_type"`
	Owner       string          `json:"owner"`
	Labels      []string        `json:"labels"`
	Deps        json.RawMessage `json:"dependencies"`
	RawJSON     json.RawMessage `json:"-"`
}

type Client struct {
	Labels    []string
	LabelsAny []string
	run       func(args []string) ([]byte, error)
}

func NewClient(labels, labelsAny []string) Client {
	return Client{
		Labels:    labels,
		LabelsAny: labelsAny,
		run:       runBD,
	}
}

func (c *Client) Ready() ([]Issue, error) {
	args := []string{"ready", "--json"}
	for _, l := range c.Labels {
		args = append(args, "--label", l)
	}
	for _, l := range c.LabelsAny {
		args = append(args, "--label-any", l)
	}

	out, err := c.run(args)
	if err != nil {
		return nil, fmt.Errorf("beads: bd ready failed: %w", err)
	}

	var rawItems []json.RawMessage
	if err := json.Unmarshal(out, &rawItems); err != nil {
		return nil, fmt.Errorf("beads: malformed JSON from bd ready: %w", err)
	}

	issues := make([]Issue, len(rawItems))
	for i, raw := range rawItems {
		if err := json.Unmarshal(raw, &issues[i]); err != nil {
			return nil, fmt.Errorf("beads: failed to parse issue %d: %w", i, err)
		}
		issues[i].RawJSON = raw
	}

	return issues, nil
}

func CheckBinary(lookup func(string) (string, error)) error {
	if _, err := lookup("bd"); err != nil {
		return fmt.Errorf("beads: binary %q not found in PATH", "bd")
	}
	return nil
}

func CheckBinaryInPath() error {
	return CheckBinary(exec.LookPath)
}

func runBD(args []string) ([]byte, error) {
	cmd := exec.Command("bd", args...)
	return cmd.Output()
}
