package beads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
)

// ErrNoWork is returned when bd ready returns an empty list.
var ErrNoWork = errors.New("no work available")

// ErrBdNotFound is returned when the bd binary is not in PATH.
var ErrBdNotFound = errors.New("bd not found in PATH")

// Issue represents a single issue from bd.
type Issue struct {
	ID    string
	Title string
	Raw   json.RawMessage
}

// Client wraps the bd CLI for fetching issues.
type Client struct {
	labels []string
}

// NewClient creates a new beads client with optional label filters.
func NewClient(labels []string) *Client {
	return &Client{labels: labels}
}

// Ready executes `bd ready --json` and returns parsed issues.
func (c *Client) Ready(ctx context.Context) ([]Issue, error) {
	bdPath, err := exec.LookPath("bd")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBdNotFound, err)
	}

	args := []string{"ready", "--json"}
	for _, l := range c.labels {
		args = append(args, "--label", l)
	}

	cmd := exec.CommandContext(ctx, bdPath, args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bd ready: %w", err)
	}

	var raw []json.RawMessage
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("bd ready: invalid JSON: %w", err)
	}

	if len(raw) == 0 {
		return nil, ErrNoWork
	}

	issues := make([]Issue, len(raw))
	for i, r := range raw {
		var partial struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		}
		if err := json.Unmarshal(r, &partial); err != nil {
			return nil, fmt.Errorf("bd ready: parsing issue: %w", err)
		}
		issues[i] = Issue{
			ID:    partial.ID,
			Title: partial.Title,
			Raw:   r,
		}
	}

	return issues, nil
}
