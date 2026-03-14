# Package Contracts

This document defines the public interface contract for each package. These are what tests verify and what other packages depend on.

## `config`

### Types

```go
type Config struct {
    Mode           Mode          // MaxIterations or Daemon
    MaxIterations  int           // only meaningful in MaxIterations mode
    SleepInterval  time.Duration // only meaningful in Daemon mode
    Harness        string        // "claude", "opencode", "codex", "copilot"
    Model          string        // optional model override
    AgentFlags     string        // extra flags for harness
    RawCommand     string        // mutually exclusive with Harness/Model/AgentFlags
    Prompt         string        // user prompt text (may be empty if beads active)
    BeadsEnabled   bool
    BeadsLabels    []string
    BeadsInstruct  string        // override default instruction
    LogDir         string
    Stderr         bool
    Verbose        bool
    Quiet          bool
}

type Mode int
const (
    MaxIterations Mode = iota
    Daemon
)
```

### Methods

```go
func (c Config) Validate() error
```

Pure method. Returns nil or a descriptive error for every fail-fast condition in the product spec. Does not touch the file system or PATH — binary existence checks happen in `internal/cli/` at wire-up time.

## `cli`

### Functions

```go
func ParseAndValidate(args []string, stdin io.Reader) (config.Config, error)
```

Parses CLI flags from args, reads stdin if needed, validates mutual exclusions and required inputs. Returns immutable config or error. This is the only place `flag` package is used.

```go
func SetupSignals(ctx context.Context) (context.Context, context.CancelFunc)
```

Returns a context that is cancelled on first SIGINT/SIGTERM. Second SIGINT triggers `os.Exit(1)`. The returned CancelFunc is for cleanup.

### Exit Code Constants

```go
const (
    ExitClean        = 0
    ExitRuntimeError = 1
    ExitUsageError   = 2
)
```

Exit codes live in `internal/cli/` because they are a CLI concern — they only matter at the boundary between the program and the shell.

## `harness`

### Interface

```go
type Harness interface {
    Run(ctx context.Context, prompt string) (exitCode int, err error)
}
```

Contract:
- `Run` invokes the external agent CLI with the given prompt.
- Returns the agent's exit code and any execution error (e.g., binary not found, signal).
- `exitCode` is meaningful only when `err == nil`.
- Context cancellation causes the agent process to be killed.
- `Run` is stateless between calls.

### Constructors

```go
func New(cfg config.Config) (Harness, error)
```

Factory that returns the appropriate harness implementation based on config. Checks binary existence in PATH. Returns error if binary not found. Returns the `Harness` interface because it selects between multiple concrete types — this is the accepted exception to "return concrete types."

## `prompt`

### Functions

```go
func Assemble(userPrompt string, issue *beads.Issue, instruction string) string
```

Pure function. Builds the final prompt string per the assembly rules:
1. User prompt (if non-empty)
2. Issue block with delimiter (if issue non-nil)
3. Instruction text (if issue non-nil)

Returns empty string only if all inputs are empty (which should be prevented by config validation).

## `beads`

### Types

```go
type Issue struct {
    ID    string
    Title string
    Raw   json.RawMessage // full JSON for prompt inclusion
}

type Client struct {
    labels []string
}
```

### Functions

```go
func NewClient(labels []string) *Client
func (c *Client) Ready(ctx context.Context) ([]Issue, error)
```

Contract:
- `Ready` executes `bd ready --json` (with optional label filter).
- Returns parsed issues sorted by bd's priority order.
- Returns `ErrNoWork` when bd returns valid JSON with zero issues.
- Returns error for invalid JSON, bd execution failures, or bd not in PATH.

### Sentinel Errors

```go
var (
    ErrNoWork     = errors.New("no work available")
    ErrBdNotFound = errors.New("bd not found in PATH")
)
```

## `loop`

### Interfaces

```go
type EventLogger interface {
    Event(name string, fields ...Field)
}

type BeadsClient interface {
    Ready(ctx context.Context) ([]beads.Issue, error)
}

type Harness interface {
    Run(ctx context.Context, prompt string) (exitCode int, err error)
}

type Printer interface {
    Starting(mode string, maxIter int, harness string, beads bool)
    Iteration(n, maxIter int, issueID, title string)
    Sleeping(d time.Duration)
    Waking()
    Done(total, succeeded, failed int, reason string)
    VerboseDetail(msg string)
}
```

### Functions

```go
func Run(ctx context.Context, cfg config.Config, h Harness, bc BeadsClient, log EventLogger, pr Printer) error
```

Contract:
- Dispatches to max-iterations or daemon mode based on `cfg.Mode`.
- Returns nil on clean completion (max reached, no work, signal shutdown).
- Returns error only for unrecoverable failures.
- Per-iteration agent failures are logged, not returned.
- Returns `ErrAllFailed` when every iteration failed (mapped to exit code 1 in main).
- Respects context cancellation for signal handling.

### Sentinel Errors

```go
var ErrAllFailed = errors.New("all iterations failed")
```

## `eventlog`

### Types

```go
type Logger struct { /* unexported fields */ }

type Field struct {
    Key   string
    Value string
}
```

### Functions

```go
func New(logDir string, stderr bool) (*Logger, error)
func (l *Logger) Event(name string, fields ...Field)
func (l *Logger) Close() error
func F(key, value string) Field
```

Contract:
- `New` creates log file in logDir. Returns error if dir not writable.
- `Event` writes one structured log line. Never returns error (logs write failure to stderr as last resort).
- `Close` flushes and closes the log file.
- `F` is a convenience constructor for Field.
