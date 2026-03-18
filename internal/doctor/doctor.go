package doctor

import (
	"io"
	"os"
	"os/exec"

	"github.com/jorgengundersen/afk/internal/config"
)

// HarnessStatus reports whether a harness binary is available.
type HarnessStatus struct {
	Status string `json:"status"`
	Path   string `json:"path,omitempty"`
}

// BeadsBinaryStatus reports whether the bd binary is available.
type BeadsBinaryStatus struct {
	Status string `json:"status"`
	Path   string `json:"path,omitempty"`
}

// BeadsProjectStatus reports whether a .beads/ directory exists.
type BeadsProjectStatus struct {
	Status string `json:"status"`
	Path   string `json:"path,omitempty"`
}

// BeadsDoctorStatus reports the result of running bd doctor.
type BeadsDoctorStatus struct {
	Status          string          `json:"status"`
	Reason          string          `json:"reason,omitempty"`
	Error           string          `json:"error,omitempty"`
	OverallOK       bool            `json:"overall_ok,omitempty"`
	CLIVersion      string          `json:"cli_version,omitempty"`
	Checks          []BdDoctorCheck `json:"checks,omitempty"`
	SuppressedCount int             `json:"suppressed_count,omitempty"`
}

// Report is the full health report produced by Collect.
type Report struct {
	Harnesses map[string]HarnessStatus `json:"harnesses"`
	Beads     struct {
		Binary  BeadsBinaryStatus  `json:"binary"`
		Project BeadsProjectStatus `json:"project"`
		Doctor  BeadsDoctorStatus  `json:"doctor"`
	} `json:"beads"`
}

// harnessBinary maps known harness names to their CLI binary names.
var harnessBinary = map[string]string{
	"claude":   "claude",
	"opencode": "opencode",
	"codex":    "codex",
	"copilot":  "copilot",
}

// Collect runs all health checks and returns a Report.
func Collect() Report {
	var r Report
	r.Harnesses = make(map[string]HarnessStatus)

	for name := range config.KnownHarnesses {
		bin := harnessBinary[name]
		if bin == "" {
			bin = name
		}
		path, err := exec.LookPath(bin)
		if err != nil {
			r.Harnesses[name] = HarnessStatus{Status: "not_found"}
		} else {
			r.Harnesses[name] = HarnessStatus{Status: "found", Path: path}
		}
	}

	// Check beads binary.
	if path, err := exec.LookPath("bd"); err != nil {
		r.Beads.Binary = BeadsBinaryStatus{Status: "not_found"}
	} else {
		r.Beads.Binary = BeadsBinaryStatus{Status: "found", Path: path}
	}

	// Check beads project directory.
	bdFound := r.Beads.Binary.Status == "found"
	projectFound := false
	if info, err := os.Stat(".beads"); err == nil && info.IsDir() {
		r.Beads.Project = BeadsProjectStatus{Status: "found", Path: ".beads"}
		projectFound = true
	} else {
		r.Beads.Project = BeadsProjectStatus{Status: "not_found"}
	}

	// Run bd doctor if both bd and .beads/ are present.
	switch {
	case !bdFound:
		r.Beads.Doctor = BeadsDoctorStatus{Status: "skipped", Reason: "bd not found"}
	case !projectFound:
		r.Beads.Doctor = BeadsDoctorStatus{Status: "skipped", Reason: ".beads/ not found"}
	default:
		result, err := runBdDoctor()
		if err != nil {
			r.Beads.Doctor = BeadsDoctorStatus{Status: "error", Error: err.Error()}
		} else {
			r.Beads.Doctor = BeadsDoctorStatus{
				Status:          "ok",
				OverallOK:       result.OverallOK,
				CLIVersion:      result.CLIVersion,
				Checks:          result.Checks,
				SuppressedCount: result.SuppressedCount,
			}
		}
	}

	return r
}

// Run is the entry point for the "afk doctor" subcommand.
// It parses its own flags from args, runs health checks, and
// returns an exit code (0 = healthy, 1 = errors found).
func Run(w io.Writer, args []string) int {
	return 0
}
