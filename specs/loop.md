# Iteration Loop

The core orchestrator. Takes a Config, a Runner, a Logger, and an optional
work source, then runs the harness in a loop. No business logic of its own —
it sequences the primitives built in prior specs.

## What this adds

```
$ afk -p "refactor auth"
# iteration 1: assemble prompt → invoke harness → log result
# iteration 2: same prompt → invoke harness → log result
# ... up to MaxIter (default 20)

$ afk -p "refactor auth" -n 1
# single iteration, then exit

$ afk -p "refactor auth" -d
# daemon mode: run iteration, sleep, repeat indefinitely

$ afk --beads
# iteration 1: fetch issue → compose prompt → invoke harness → log result
# iteration 2: fetch next issue → compose prompt → invoke harness → log result
# no issues ready → exit 0

$ afk --beads -d
# same, but sleeps and retries when no issues ready
```

## Structure

```
internal/loop/loop.go       # Run function
internal/loop/loop_test.go  # Tests
```

## Run

The loop accepts a Config, a Runner, a Logger, and an optional WorkSource.

- `ctx` carries cancellation (from signal handling, tested via context).
- `cfg` provides MaxIter, Daemon, Sleep, and Prompt.
- `runner` satisfies the local Runner interface — the loop calls Run(ctx, prompt).
- `logger` satisfies the local Logger interface — the loop calls Log(event, fields).
  Log returns an error. If logging fails, the loop exits immediately with
  exit code 1 — a broken logger means the session is unobservable, which
  violates the fail-fast principle. The loop does not attempt to continue
  without logging.
- `workSource` is an optional dependency. When nil, the loop uses the static
  prompt from Config. When present, the loop calls it before each iteration
  to get the prompt. The work source returns a prompt string, an issue ID,
  an issue title, and a boolean indicating whether work is available.
- Returns an exit code and an optional error.

The loop defines its own Config, Runner, Logger, and WorkSource types rather
than importing concrete types from other packages. This decouples the loop
from the config, harness, beads, and logger packages and enables testability
via dependency injection.

The loop does NOT call ParseFlags, Validate, Assemble, or New. Those are
called by main before the loop starts. The loop receives ready-to-use
dependencies.

## Behaviour

### Max-iterations mode (default)

- Runs exactly MaxIter iterations unless cancelled via context or work
  source is exhausted.
- Non-zero exit codes from the agent do NOT stop the loop. Log and continue.
- Launch failures (err != nil) do NOT stop the loop. Log and continue.
- If ALL iterations had launch failures (err != nil on every one), return
  exit code 1.
- When a work source is present and returns no work, the loop exits with
  reason "no-work" and exit code 0.
- The beads-check event is logged with the issue count each time the work
  source is consulted.

### Daemon mode

- Runs indefinitely until context is cancelled.
- All iterations use a fixed iteration number of 0 (no incrementing counter).
- Sleeps between iterations for `cfg.Sleep` duration.
- Sleep must be interruptible — if ctx is cancelled during sleep, wake
  immediately and exit.
- When a work source is present and returns no work, the loop sleeps and
  retries on the next cycle instead of exiting.
- If ALL iterations had launch failures or non-zero exit codes, return
  exit code 1 (same as max-iterations mode).

### Context cancellation

When ctx is cancelled (signal received):

- If the harness is running: the harness's Run method handles its own
  cancellation (it was given the same ctx). The loop does not kill subprocesses
  directly.
- If sleeping: wake immediately.
- After the current iteration completes (or sleep is interrupted), the loop
  exits cleanly.
- Log `session-end` with `reason=signal` before returning.

## Test cases

The loop is tested with a fake Runner (satisfies the interface) and a real
Logger writing to a temp file. No real subprocesses in loop tests.

### Unit tests

| Test                            | Setup                                  | Expected                                    |
|---------------------------------|----------------------------------------|---------------------------------------------|
| single iteration                | MaxIter=1, runner returns 0            | 1 iteration, exit 0, session-start/end logged |
| multiple iterations             | MaxIter=3, runner returns 0            | 3 iterations, exit 0                        |
| non-zero exit continues         | MaxIter=2, runner returns 1 then 0     | 2 iterations, exit 0                        |
| launch failure continues        | MaxIter=2, runner returns err          | 2 iterations, error logged each time        |
| all launch failures             | MaxIter=2, runner always returns err   | 2 iterations, exit 1                        |
| context cancellation mid-loop   | MaxIter=10, cancel after 2 iterations  | ~2 iterations, exit 0, reason=signal        |
| daemon sleeps between iters     | Daemon=true, cancel after 2 iterations | 2 iterations with sleep between, exit 0     |
| daemon sleep interrupted        | Daemon=true, cancel during sleep       | exits promptly, exit 0                      |

### Log verification

Tests should read the temp log file and verify:
- `session-start` is the first event with correct `mode` and `maxIter` fields.
- `iteration-start` and `iteration-end` bracket each iteration.
- `session-end` is the last event with correct reason.
- `error` events appear when runner returns err.

### Iteration logging with work source

When a work source provides an issue, the iteration-start and iteration-end
log events include the issue ID and title fields alongside the iteration
number. When no work source is present, these fields are omitted (existing
behaviour).

## Out of scope

- Signal handling setup (separate spec — the loop just respects ctx).
- Prompt assembly — the work source or caller handles this before the loop
  receives the prompt.
- Retry backoff or adaptive sleep.
- Progress output to terminal (the loop is silent; agents write to terminal
  directly).
- Parallel loop coordination or race condition prevention — label
  partitioning is a user convention documented in the beads client spec.

## Definition of done

- `Run` orchestrates iterations using Runner, Logger, and optional WorkSource.
- Max-iterations mode runs exactly N iterations or exits on no work.
- Daemon mode runs indefinitely with interruptible sleep, retries on no work.
- Context cancellation exits cleanly with reason=signal.
- Non-zero exit codes and launch failures do not stop the loop.
- All launch failures → exit code 1.
- Iteration logs include issue ID and title when work source provides them.
- All test cases pass.
