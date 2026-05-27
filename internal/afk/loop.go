package afk

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
)

var errInterrupted = errors.New("interrupted")

func RunLoop(cfg Config, stdin io.Reader, stdout, stderr io.Writer) int {
	return runLoopWithInterrupts(cfg, stdin, stdout, stderr, nil)
}

func runLoopWithInterrupts(cfg Config, stdin io.Reader, stdout, stderr io.Writer, interrupts <-chan os.Signal) int {
	if interrupts == nil {
		sigCh := make(chan os.Signal, 2)
		signal.Notify(sigCh, syscall.SIGINT)
		defer signal.Stop(sigCh)
		interrupts = sigCh
	}

	if cfg.LoopsExplicit {
		return runFixedNonItemLoops(cfg, stdin, stdout, stderr, interrupts)
	}

	if cfg.Daemon && !cfg.ItemsExplicit && !cfg.ItemsCmdExplicit {
		return runDaemonNonItemLoop(cfg, stdin, stdout, stderr, interrupts)
	}

	if cfg.ItemsExplicit {
		return runStaticItemLoop(cfg, stdin, stdout, stderr, interrupts)
	}

	if cfg.ItemsCmdExplicit {
		if cfg.Daemon {
			return runDaemonDynamicItemLoop(cfg, stdin, stdout, stderr, interrupts)
		}
		return runNonDaemonDynamicItemLoop(cfg, stdin, stdout, stderr, interrupts)
	}

	return 0
}

func runFixedNonItemLoops(cfg Config, stdin io.Reader, stdout, stderr io.Writer, interrupts <-chan os.Signal) int {
	lastNonZero := 0

	for i := 1; i <= cfg.Loops; i++ {
		if pendingSigint(interrupts) {
			return interruptExit(stderr)
		}

		env := EnvForNonItemInvocation(os.Environ(), i)
		exitCode, err := runMainChild(cfg.CommandArgv, stdin, stdout, stderr, env, cfg.Timeout, interrupts)
		if err != nil {
			return mapMainChildExecutionError(cfg.CommandArgv[0], err, stderr)
		}

		decision := decideMainChildExit(cfg.Fail, cfg.UntilSuccess, exitCode)
		if decision.recordLastNonZero {
			lastNonZero = exitCode
		}
		if decision.stop {
			return decision.exitCode
		}
	}

	return finalExitAfterCompletedWork(cfg.UntilSuccess, lastNonZero)
}

func runDaemonNonItemLoop(cfg Config, stdin io.Reader, stdout, stderr io.Writer, interrupts <-chan os.Signal) int {
	for i := 1; ; i++ {
		if pendingSigint(interrupts) {
			return interruptExit(stderr)
		}

		env := EnvForNonItemInvocation(os.Environ(), i)
		exitCode, err := runMainChild(cfg.CommandArgv, stdin, stdout, stderr, env, cfg.Timeout, interrupts)
		if err != nil {
			return mapMainChildExecutionError(cfg.CommandArgv[0], err, stderr)
		}

		decision := decideMainChildExit(cfg.Fail, cfg.UntilSuccess, exitCode)
		if decision.stop {
			return decision.exitCode
		}
	}
}

func runStaticItemLoop(cfg Config, stdin io.Reader, stdout, stderr io.Writer, interrupts <-chan os.Signal) int {
	items, err := ParseStaticItems(cfg.Items)
	if err != nil {
		fmt.Fprintf(stderr, "item parse error: %v\n", err)
		return 1
	}

	for itemIndex, item := range items {
		if pendingSigint(interrupts) {
			return interruptExit(stderr)
		}

		env := EnvForItemInvocation(os.Environ(), itemIndex+1, item, itemIndex, len(items))
		exitCode, err := runMainChild(cfg.CommandArgv, stdin, stdout, stderr, env, cfg.Timeout, interrupts)
		if err != nil {
			return mapMainChildExecutionError(cfg.CommandArgv[0], err, stderr)
		}

		decision := decideMainChildExit(cfg.Fail, cfg.UntilSuccess, exitCode)
		if decision.stop {
			return decision.exitCode
		}
	}

	return finalExitAfterCompletedWork(cfg.UntilSuccess, 0)
}

func runNonDaemonDynamicItemLoop(cfg Config, stdin io.Reader, stdout, stderr io.Writer, interrupts <-chan os.Signal) int {
	invocationIndex := 1
	remainingEmptySleeps := cfg.EmptySleeps

	for {
		if pendingSigint(interrupts) {
			return interruptExit(stderr)
		}

		items, err := runItemsCommand(cfg.ItemsCmd, stderr, cfg.Timeout, interrupts)
		if err != nil {
			if errors.Is(err, errInterrupted) {
				return 130
			}
			fmt.Fprintf(stderr, "item source error: %v\n", err)
			return 1
		}

		if len(items) == 0 {
			if remainingEmptySleeps <= 0 {
				return finalExitAfterCompletedWork(cfg.UntilSuccess, 0)
			}
			remainingEmptySleeps--
			if cfg.Sleep > 0 {
				if interruptibleSleep(cfg.Sleep, interrupts) {
					return interruptExit(stderr)
				}
			}
			continue
		}

		for itemIndex, item := range items {
			if pendingSigint(interrupts) {
				return interruptExit(stderr)
			}

			env := EnvForItemInvocation(os.Environ(), invocationIndex, item, itemIndex, len(items))
			exitCode, err := runMainChild(cfg.CommandArgv, stdin, stdout, stderr, env, cfg.Timeout, interrupts)
			if err != nil {
				return mapMainChildExecutionError(cfg.CommandArgv[0], err, stderr)
			}

			decision := decideMainChildExit(cfg.Fail, cfg.UntilSuccess, exitCode)
			if decision.stop {
				return decision.exitCode
			}
			invocationIndex++
		}

		return finalExitAfterCompletedWork(cfg.UntilSuccess, 0)
	}
}

func runDaemonDynamicItemLoop(cfg Config, stdin io.Reader, stdout, stderr io.Writer, interrupts <-chan os.Signal) int {
	invocationIndex := 1

	for {
		if pendingSigint(interrupts) {
			return interruptExit(stderr)
		}

		items, err := runItemsCommand(cfg.ItemsCmd, stderr, cfg.Timeout, interrupts)
		if err != nil {
			if errors.Is(err, errInterrupted) {
				return 130
			}
			fmt.Fprintf(stderr, "item source error: %v\n", err)
			return 1
		}

		if len(items) == 0 {
			if cfg.Sleep > 0 {
				if interruptibleSleep(cfg.Sleep, interrupts) {
					return interruptExit(stderr)
				}
			}
			continue
		}

		for itemIndex, item := range items {
			if pendingSigint(interrupts) {
				return interruptExit(stderr)
			}

			env := EnvForItemInvocation(os.Environ(), invocationIndex, item, itemIndex, len(items))
			exitCode, err := runMainChild(cfg.CommandArgv, stdin, stdout, stderr, env, cfg.Timeout, interrupts)
			if err != nil {
				return mapMainChildExecutionError(cfg.CommandArgv[0], err, stderr)
			}

			decision := decideMainChildExit(cfg.Fail, cfg.UntilSuccess, exitCode)
			if decision.stop {
				return decision.exitCode
			}
			invocationIndex++
		}
	}
}
