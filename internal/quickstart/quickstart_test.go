package quickstart

import "testing"

func TestTextReturnsNonEmpty(t *testing.T) {
	got := Text()
	if got == "" {
		t.Fatal("Text() returned empty string")
	}
}

func TestTextEndsWithNewline(t *testing.T) {
	got := Text()
	if got[len(got)-1] != '\n' {
		t.Error("Text() should end with a newline")
	}
}
