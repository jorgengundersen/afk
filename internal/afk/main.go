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
	cfg, err := ParseArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "usage error: %v\n", err)
		return 2
	}

	if cfg.Help {
		_, _ = io.WriteString(stdout, usageText)
		return 0
	}

	if err := ValidateConfig(cfg); err != nil {
		fmt.Fprintf(stderr, "usage error: %v\n", err)
		return 2
	}

	return RunLoop(cfg, stdin, stdout, stderr)
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
