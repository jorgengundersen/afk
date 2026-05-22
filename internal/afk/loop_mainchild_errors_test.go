package afk

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
)

func TestMapMainChildExecutionError(t *testing.T) {
	tests := []struct {
		name           string
		command        string
		err            error
		wantExitCode   int
		wantDiagnostic string
	}{
		{
			name:           "interrupt exits 130 without writing execution error diagnostic",
			command:        "missing-bin",
			err:            errInterrupted,
			wantExitCode:   130,
			wantDiagnostic: "",
		},
		{
			name:           "missing command maps to 127 start failure",
			command:        "missing-bin",
			err:            exec.ErrNotFound,
			wantExitCode:   127,
			wantDiagnostic: "command not found: missing-bin",
		},
		{
			name:           "permission denied maps to 126 start failure",
			command:        "./script.sh",
			err:            &os.PathError{Op: "fork/exec", Path: "./script.sh", Err: syscall.EACCES},
			wantExitCode:   126,
			wantDiagnostic: "cannot execute: ./script.sh",
		},
		{
			name:           "generic execution error maps to exit 1",
			command:        "missing-bin",
			err:            errors.New("boom"),
			wantExitCode:   1,
			wantDiagnostic: "execution error: boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stderr bytes.Buffer

			exitCode := mapMainChildExecutionError(tt.command, tt.err, &stderr)
			if exitCode != tt.wantExitCode {
				t.Fatalf("mapMainChildExecutionError() exit code = %d, want %d", exitCode, tt.wantExitCode)
			}

			if tt.wantDiagnostic == "" {
				if strings.Contains(stderr.String(), "execution error") {
					t.Fatalf("stderr = %q, want no generic execution error diagnostic", stderr.String())
				}
				return
			}

			if !strings.Contains(stderr.String(), tt.wantDiagnostic) {
				t.Fatalf("stderr = %q, want diagnostic containing %q", stderr.String(), tt.wantDiagnostic)
			}
		})
	}
}
