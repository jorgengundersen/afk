package doctor

import (
	"bytes"
	"testing"
)

func TestRun_ReturnsZero(t *testing.T) {
	var buf bytes.Buffer
	code := Run(&buf, nil)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}
