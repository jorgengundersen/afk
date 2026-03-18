package doctor

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// BdDoctorCheck represents a single health check from bd doctor.
type BdDoctorCheck struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Message  string `json:"message"`
	Detail   string `json:"detail,omitempty"`
	Fix      string `json:"fix,omitempty"`
	Category string `json:"category,omitempty"`
}

// BdDoctorResult represents the full output of bd doctor --json.
type BdDoctorResult struct {
	Path            string            `json:"path"`
	Checks          []BdDoctorCheck   `json:"checks"`
	OverallOK       bool              `json:"overall_ok"`
	CLIVersion      string            `json:"cli_version"`
	Timestamp       string            `json:"timestamp,omitempty"`
	Platform        map[string]string `json:"platform,omitempty"`
	SuppressedCount int               `json:"suppressed_count,omitempty"`
}

// runBdDoctor executes "bd doctor --json" and parses the output into a BdDoctorResult.
// It returns an error if bd exits non-zero or the output is not valid JSON.
func runBdDoctor() (*BdDoctorResult, error) {
	out, err := exec.Command("bd", "doctor", "--json").Output()
	if err != nil {
		return nil, fmt.Errorf("bd doctor: %w", err)
	}

	var result BdDoctorResult
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, fmt.Errorf("parsing bd doctor output: %w", err)
	}
	return &result, nil
}
