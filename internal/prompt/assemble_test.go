package prompt

import "testing"

func TestAssemble_PromptProvided(t *testing.T) {
	got, err := Assemble("do stuff")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "do stuff" {
		t.Errorf("Assemble() = %q, want %q", got, "do stuff")
	}
}

func TestAssemble_EmptyPrompt(t *testing.T) {
	_, err := Assemble("")
	if err == nil {
		t.Fatal("expected error for empty prompt, got nil")
	}
	want := "no prompt provided"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestAssemble_WhitespaceOnly(t *testing.T) {
	got, err := Assemble("   ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "   " {
		t.Errorf("Assemble() = %q, want %q", got, "   ")
	}
}
