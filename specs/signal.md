# Signal Handling

Graceful shutdown on SIGINT and SIGTERM. Wraps the loop's context so that a
signal cancels it cleanly. No business logic — just plumbing between OS signals
and Go's context cancellation.

## What this adds

```
$ afk -p "do work" -n 20
# user hits Ctrl+C during iteration 5
# current harness invocation finishes (or is cancelled by the harness)
# loop exits, logs session-end reason=signal
# afk exits 0

$ afk -p "do work" -d
# kill -TERM <pid>
# same behaviour: clean exit after current iteration
```

## Structure

```
internal/signal/signal.go       # Signal context setup
internal/signal/signal_test.go  # Tests
```

## NotifyContext

```go
func NotifyContext(parent context.Context) (context.Context, context.CancelFunc)
```

- Returns a context that is cancelled when SIGINT or SIGTERM is received.
- Returns a cancel func for cleanup (deferred in main).
- Uses `signal.NotifyContext` from the standard library under the hood.
- This is a thin wrapper. Its value is in being the single place where signal
  handling is configured, keeping main clean.

## Behaviour

- First SIGINT/SIGTERM: cancel context. The loop finishes its current
  iteration and exits cleanly with code 0.
- The harness Runner received the same ctx. How it handles cancellation is
  the harness's concern (it may forward the signal to the subprocess, or
  wait for it to finish).
- If the loop is sleeping (daemon mode), sleep is interrupted immediately.

## Test cases

| Test                          | Setup                                    | Expected                                  |
|-------------------------------|------------------------------------------|-------------------------------------------|
| context cancelled on signal   | Send SIGINT to self after delay          | ctx.Done() fires                          |
| cancel func stops listening   | Call cancel, then send SIGINT            | ctx not cancelled (signal ignored)        |

Note: testing actual signal delivery requires `syscall.Kill(syscall.Getpid(), ...)`.
These are integration-style tests.

## Out of scope

- Second signal forcing immediate exit (kill -9 is always available to the
  user; no need to handle double-signal).
- Timeout on graceful shutdown (if the harness hangs, the user sends another
  signal or kills the process).
- Signal forwarding to child processes (the harness owns its subprocess
  lifecycle — see "Process group management" in harness spec).

## Definition of done

- `NotifyContext` returns a context cancelled by SIGINT/SIGTERM.
- Tests verify context cancellation on signal delivery.
- Tests pass.
