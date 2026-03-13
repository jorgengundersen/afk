# Testing Standards

## Philosophy

**Red/green TDD is the standard development practice.** Every piece of functionality is built one test at a time: write a failing test, make it pass, refactor, commit. No implementation without a failing test first. No batching multiple behaviors into one cycle.

Tests are the agent's primary self-verification mechanism. A test suite that doesn't catch real bugs is worse than no tests — it provides false confidence.

## Test Classification

### Unit Tests (majority)

Test individual primitives. Pure functions get table-driven tests. Dependency wrappers get tests against the real dependency where practical (e.g., file system operations use `t.TempDir()`), or against a fake where not.

Run with: `go test ./...`

### Integration Tests (selective)

Test that packages compose correctly. Primarily for `loop/` orchestration logic. Use build tag `//go:build integration` only if they require external binaries or are slow (>5s). Otherwise, they run with standard `go test`.

### End-to-End Tests (few)

Test the compiled binary as a black box. Build `afk`, invoke it with known inputs, assert on exit code and log output. Use build tag `//go:build e2e`. These require agent harness binaries to be available and are not part of the default test suite.

## Test Design

### Black-box by default

Tests import the package under test with the `_test` package suffix:

```go
package prompt_test

import "afk/prompt"
```

This forces tests to use only the exported API. If you can't test something without accessing internals, the exported API is incomplete — fix the API, don't break the test boundary.

**Exception:** Complex internal algorithms where testing exported behavior alone would require unreasonable setup. Document why with a comment at the top of the test file.

### Table-driven tests

Use table-driven tests for functions with multiple input/output cases:

```go
func TestAssemblePrompt(t *testing.T) {
    tests := []struct {
        name        string
        userPrompt  string
        issue       *beads.Issue
        instruction string
        want        string
    }{
        {
            name:       "user prompt only",
            userPrompt: "fix the bug",
            want:       "fix the bug",
        },
        {
            name:        "issue only",
            issue:       &beads.Issue{ID: "afk-1", Title: "Fix auth"},
            instruction: "Claim and complete.",
            want:        "--- issue ---\n{...}\nClaim and complete.",
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := prompt.Assemble(tt.userPrompt, tt.issue, tt.instruction)
            if got != tt.want {
                t.Errorf("Assemble() = %q, want %q", got, tt.want)
            }
        })
    }
}
```

### Subtests with `t.Run`

Always use `t.Run` with descriptive names. This enables running individual test cases and produces clear failure output.

### No test helpers that hide assertions

Helper functions may set up fixtures or build test data. Assertions live in the test function itself. A failing test should point directly at the assertion that failed, not deep inside a helper.

Helper functions that can fail call `t.Helper()` to fix stack traces.

### Test naming

```
TestFunctionName                          // single case
TestFunctionName/descriptive_case_name    // table-driven subtest
```

Name describes the scenario, not the expected outcome. `"empty_prompt_with_beads"` not `"should_return_issue_only"`.

## Fakes and Test Doubles

### Interfaces enable test doubles

The `Harness` interface exists partly to enable testing `loop/` without invoking real agent CLIs. Test doubles implement the same interface.

### Fakes over mocks

Write simple fake implementations, not mock frameworks. A fake harness that records calls and returns configured results:

```go
type fakeHarness struct {
    calls    []string
    exitCode int
    err      error
}

func (f *fakeHarness) Run(ctx context.Context, prompt string) (int, error) {
    f.calls = append(f.calls, prompt)
    return f.exitCode, f.err
}
```

No mock libraries. No reflection-based mocking. Fakes are plain Go structs.

### Real dependencies when cheap

`os/exec` tests: use a real subprocess where possible (a small test binary or `echo`). File system tests: use `t.TempDir()`. Only fake what is expensive, slow, or external.

## What to Test

### Always test

- All exported functions — they are contracts.
- Error paths — every `error` return has a test case that triggers it.
- Edge cases from the product spec — mutual exclusion of flags, empty inputs, missing binaries.
- Prompt assembly — exact output for all combinations (user prompt only, issue only, both, neither-should-error).
- Config validation — every fail-fast condition from the product spec.

### Don't test

- Unexported helpers that are only called from tested exported functions (tested transitively).
- Direct `os/exec.Command` construction details — test via the harness interface.
- Log formatting minutiae — test that events are logged, not exact string format.

## Test Execution

```bash
# All unit tests
go test ./...

# With race detector
go test -race ./...

# Verbose
go test -v ./...

# Specific package
go test ./prompt/

# Integration tests (if tagged)
go test -tags=integration ./...

# Coverage report
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

## CI Gate

All of the following must pass before merge:

1. `go vet ./...`
2. `gofmt -l .` (no output = all formatted)
3. `go test -race ./...`
4. `go build ./...`

These are the minimum quality gates. No merge with failures.

## TDD Workflow

Per PROMPT.md: red/green TDD. One test, one implementation.

1. Write a failing test for the next piece of behavior.
2. Run the test, confirm it fails for the right reason.
3. Write the minimum code to make it pass.
4. Run all tests, confirm green.
5. Refactor if needed (tests stay green).
6. Commit.

Each commit should have a corresponding test. No implementation without a test. No test without a clear behavior it verifies.
