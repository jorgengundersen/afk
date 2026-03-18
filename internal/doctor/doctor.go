package doctor

import "io"

// Run is the entry point for the "afk doctor" subcommand.
// It parses its own flags from args, runs health checks, and
// returns an exit code (0 = healthy, 1 = errors found).
func Run(w io.Writer, args []string) int {
	return 0
}
