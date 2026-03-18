package doctor

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"

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

// PrintHuman writes a human-readable health report to w.
func PrintHuman(w io.Writer, r Report) {
	fmt.Fprintln(w, "afk doctor")
	fmt.Fprintln(w)

	// Harnesses section — sorted for consistent order.
	fmt.Fprintln(w, "Harnesses")
	names := make([]string, 0, len(r.Harnesses))
	for name := range r.Harnesses {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		h := r.Harnesses[name]
		if h.Status == "found" {
			fmt.Fprintf(w, "  %-10s found (%s)\n", name, h.Path)
		} else {
			fmt.Fprintf(w, "  %-10s not found\n", name)
		}
	}

	fmt.Fprintln(w)

	// Beads section.
	fmt.Fprintln(w, "Beads")

	// Binary.
	if r.Beads.Binary.Status == "found" {
		fmt.Fprintf(w, "  %-18s found (%s)\n", "bd binary", r.Beads.Binary.Path)
	} else {
		fmt.Fprintf(w, "  %-18s not found\n", "bd binary")
	}

	// Project.
	if r.Beads.Project.Status == "found" {
		fmt.Fprintf(w, "  %-18s found\n", "project (.beads/)")
	} else {
		fmt.Fprintf(w, "  %-18s not found\n", "project (.beads/)")
	}

	// Health.
	switch r.Beads.Doctor.Status {
	case "ok":
		fmt.Fprintf(w, "  %-18s ok (v%s, %d checks passed)\n",
			"health", r.Beads.Doctor.CLIVersion, len(r.Beads.Doctor.Checks))
	case "error":
		fmt.Fprintf(w, "  %-18s error (%s)\n", "health", r.Beads.Doctor.Error)
	case "skipped":
		fmt.Fprintf(w, "  %-18s skipped (%s)\n", "health", r.Beads.Doctor.Reason)
	}

	// Warnings and errors — only shown when doctor ran (not skipped).
	if r.Beads.Doctor.Status == "ok" || r.Beads.Doctor.Status == "error" {
		fmt.Fprintln(w)

		// Warnings.
		var warnings []BdDoctorCheck
		for _, c := range r.Beads.Doctor.Checks {
			if c.Status == "warning" {
				warnings = append(warnings, c)
			}
		}
		fmt.Fprintln(w, "  warnings:")
		if len(warnings) == 0 {
			fmt.Fprintln(w, "    (none)")
		} else {
			for _, c := range warnings {
				fmt.Fprintf(w, "    %s: %s (category: %s)\n", c.Name, c.Message, c.Category)
			}
		}

		fmt.Fprintln(w)

		// Errors.
		var errors []BdDoctorCheck
		for _, c := range r.Beads.Doctor.Checks {
			if c.Status == "error" {
				errors = append(errors, c)
			}
		}
		fmt.Fprintln(w, "  errors:")
		if len(errors) == 0 {
			fmt.Fprintln(w, "    (none)")
		} else {
			for _, c := range errors {
				fmt.Fprintf(w, "    %s: %s (category: %s)\n", c.Name, c.Message, c.Category)
			}
		}
	}
}

// PrintJSON marshals the Report as indented JSON and writes it to w.
func PrintJSON(w io.Writer, r Report) {
	data, _ := json.MarshalIndent(r, "", "  ")
	data = append(data, '\n')
	w.Write(data)
}

// Run is the entry point for the "afk doctor" subcommand.
// It parses its own flags from args, runs health checks, and
// returns an exit code (0 = healthy, 1 = errors found).
func Run(w io.Writer, args []string) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	jsonFlag := fs.Bool("json", false, "output as JSON")
	fs.SetOutput(w)
	if err := fs.Parse(args); err != nil {
		return 1
	}

	r := Collect()

	if *jsonFlag {
		PrintJSON(w, r)
	} else {
		PrintHuman(w, r)
	}

	// Exit 1 if any beads check has error status.
	for _, c := range r.Beads.Doctor.Checks {
		if c.Status == "error" {
			return 1
		}
	}
	return 0
}
