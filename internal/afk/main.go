package afk

import (
	"fmt"
	"io"
)

const usageText = `Usage: afk [flags] -- command [args...]

Flags:
  -h, --help              Show help
  -n, --loops N           Run main child N times
  -d, --daemon            Run forever until interrupted
      --items TEXT        Newline-delimited or JSON-array item batch
      --items-cmd CMD     Shell command that outputs item batches (max captured stdout 8 MiB)
      --sleep DURATION    Sleep between empty --items-cmd batches (default 1m)
      --empty-sleeps N    Stop after N consecutive empty batches (default 0 = no limit)
      --fail MODE         Failure policy: continue or stop (default continue)
      --until-success     Retry until main child exits 0 (with --loops or --daemon)
      --timeout DURATION  Per-command timeout for main child and --items-cmd
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
