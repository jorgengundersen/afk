package afk

import (
	"fmt"
	"io"
)

const usageText = `Usage: afk [flags] -- command [args...]

Flags:
  -h, --help  Show help
`

func Main(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	_ = stdin

	if wantsHelp(args) {
		_, _ = io.WriteString(stdout, usageText)
		return 0
	}

	fmt.Fprintln(stderr, "usage error: missing required -- separator")
	return 2
}

func wantsHelp(args []string) bool {
	limit := len(args)
	for i, arg := range args {
		if arg == "--" {
			limit = i
			break
		}
	}

	for _, arg := range args[:limit] {
		if arg == "-h" || arg == "--help" {
			return true
		}
	}

	return false
}
