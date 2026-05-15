package afk

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
)

func RunLoop(cfg Config, stdin io.Reader, stdout, stderr io.Writer) int {
	if cfg.LoopsExplicit {
		return runFixedNonItemLoops(cfg, stdin, stdout, stderr)
	}

	if cfg.Daemon && !cfg.ItemsExplicit && !cfg.ItemsCmdExplicit {
		return runDaemonNonItemLoop(cfg, stdin, stdout, stderr)
	}

	return 0
}

func runFixedNonItemLoops(cfg Config, stdin io.Reader, stdout, stderr io.Writer) int {
	for i := 1; i <= cfg.Loops; i++ {
		exitCode, err := runMainChild(cfg.CommandArgv, stdin, stdout, stderr, i)
		if err != nil {
			fmt.Fprintf(stderr, "execution error: %v\n", err)
			return 1
		}
		if exitCode != 0 && cfg.Fail == "stop" {
			return exitCode
		}
	}

	return 0
}

func runDaemonNonItemLoop(cfg Config, stdin io.Reader, stdout, stderr io.Writer) int {
	for i := 1; ; i++ {
		exitCode, err := runMainChild(cfg.CommandArgv, stdin, stdout, stderr, i)
		if err != nil {
			fmt.Fprintf(stderr, "execution error: %v\n", err)
			return 1
		}
		if exitCode != 0 && cfg.Fail == "stop" {
			return exitCode
		}
	}
}

func runMainChild(argv []string, stdin io.Reader, stdout, stderr io.Writer, iteration int) (int, error) {
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = EnvForNonItemInvocation(os.Environ(), iteration)

	err := cmd.Run()
	if err == nil {
		return 0, nil
	}

	exitErr, ok := err.(*exec.ExitError)
	if ok {
		if waitStatus, ok := exitErr.Sys().(syscall.WaitStatus); ok && waitStatus.Signaled() {
			return 128 + int(waitStatus.Signal()), nil
		}
		return exitErr.ExitCode(), nil
	}

	return 0, err
}
