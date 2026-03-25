# Iteration Loop

The core orchestrator. Takes a Config, a Runner, and a Logger, then runs the
harness in a loop. No business logic of its own — it sequences the primitives
built in prior specs.

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
```

## Structure

```
internal/loop/loop.go       # Run function
internal/loop/loop_test.go  # Tests
```

## Run

```go
// Config holds the parameters the loop needs to run.
type Config struct {
    MaxIter int
    Daemon  bool
    Sleep   time.Duration
    Prompt  string
}

// Logger is the interface the loop uses to record events.
type Logger interface {
    Log(event string, fields map[string]any)
}

// Runner executes an agent with the given prompt.
type Runner interface {
    Run(ctx context.Context, prompt string) (int, error)
}

func Run(ctx context.Context, cfg Config, runner Runner, logger Logger) (int, error)
```

- `ctx` carries cancellation (from signal handling, tested via context).
- `cfg` provides MaxIter, Daemon, Sleep, and Prompt.
- `runner` satisfies the local `Runner` interface — the loop calls `Run(ctx, prompt)`.
- `logger` satisfies the local `Logger` interface — the loop calls `Log(event, fields)`.
- Returns an exit code and an optional error.

The loop defines its own `Config`, `Runner`, and `Logger` types rather than importing
concrete types from other packages. This decouples the loop from the config, harness,
and logger packages and enables testability via dependency injection.

The loop does NOT call ParseFlags, Validate, Assemble, or New. Those are
called by main before the loop starts. The loop receives ready-to-use
dependencies.

## Behaviour

### Max-iterations mode (default)

```
log session-start {mode: "max-iterations", maxIter: MaxIter}
for i := 1; i <= MaxIter; i++ {
    log iteration-start
    exitCode, err := runner.Run(ctx, prompt)
    log iteration-end (with exit code, duration)
    if err != nil {
        log error
        // launch failure — still continue
    }
}
log session-end reason=complete
return 0
```

- Runs exactly MaxIter iterations unless cancelled via context.
- Non-zero exit codes from the agent do NOT stop the loop. Log and continue.
- Launch failures (err != nil) do NOT stop the loop. Log and continue.
- If ALL iterations had launch failures (err != nil on every one), return
  exit code 1.

### Daemon mode

```
log session-start {mode: "daemon", maxIter: MaxIter}
for {
    log iteration-start
    exitCode, err := runner.Run(ctx, prompt)
    log iteration-end
    if err != nil {
        log error
    }
    log sleeping
    sleep(cfg.Sleep) // interruptible by ctx
    log waking
}
// only exits via context cancellation
log session-end reason=signal
return 0
```

- Runs indefinitely until context is cancelled.
- All iterations use a fixed iteration number of 0 (no incrementing counter).
- Sleeps between iterations for `cfg.Sleep` duration.
- Sleep must be interruptible — if ctx is cancelled during sleep, wake
  immediately and exit.

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

## Out of scope

- Beads issue fetching and prompt recomposition per iteration (future beads
  integration spec).
- Signal handling setup (separate spec — the loop just respects ctx).
- Prompt assembly — done before the loop, passed in as a string.
- Retry backoff or adaptive sleep.
- Progress output to terminal (the loop is silent; agents write to terminal
  directly).

## Definition of done

- `Run` orchestrates iterations using Runner and Logger.
- Max-iterations mode runs exactly N iterations.
- Daemon mode runs indefinitely with interruptible sleep.
- Context cancellation exits cleanly with reason=signal.
- Non-zero exit codes and launch failures do not stop the loop.
- All launch failures → exit code 1.
- All test cases pass.
- Tests passes.
