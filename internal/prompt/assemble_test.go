package prompt

import (
	"strings"
	"testing"
)

func TestAssemble_PromptOnly(t *testing.T) {
	got, err := Assemble("do stuff", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "do stuff" {
		t.Errorf("Assemble() = %q, want %q", got, "do stuff")
	}
}

func TestAssemble_NeitherPromptNorIssue(t *testing.T) {
	_, err := Assemble("", "")
	if err == nil {
		t.Fatal("expected error for empty prompt and no issue")
	}
	want := "no prompt provided"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestAssemble_WhitespaceOnlyNoIssue(t *testing.T) {
	_, err := Assemble("   ", "")
	if err == nil {
		t.Fatal("expected error for whitespace-only prompt with no issue")
	}
	want := "no prompt provided"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestAssemble_IssueOnly(t *testing.T) {
	issueJSON := `{"id":"afk-1","title":"Fix bug"}`
	got, err := Assemble("", issueJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, issueJSON) {
		t.Error("result should contain issue JSON")
	}
	if !strings.Contains(strings.ToLower(got), "claim") {
		t.Error("result should contain claim instruction")
	}
	if !strings.Contains(strings.ToLower(got), "close") {
		t.Error("result should contain close instruction")
	}
}

func TestAssemble_BothPromptAndIssue(t *testing.T) {
	issueJSON := `{"id":"afk-1","title":"Fix bug"}`
	got, err := Assemble("focus on tests", issueJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, issueJSON) {
		t.Error("result should contain issue JSON")
	}
	if !strings.Contains(got, "focus on tests") {
		t.Error("result should contain user prompt")
	}
	if !strings.Contains(strings.ToLower(got), "claim") {
		t.Error("result should contain claim instruction")
	}
}

func TestAssemble_WhitespacePromptWithIssue(t *testing.T) {
	issueJSON := `{"id":"afk-1","title":"Fix bug"}`
	got, err := Assemble("   ", issueJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should behave like issue-only (whitespace prompt ignored)
	if !strings.Contains(got, issueJSON) {
		t.Error("result should contain issue JSON")
	}
	if strings.Contains(got, "   ") {
		t.Error("result should not contain whitespace-only prompt")
	}
}

func TestAssemble_IssueJSONInjectedAsIs(t *testing.T) {
	issueJSON := `{"id":"afk-1","title":"Fix bug","extra_field":true}`
	got, err := Assemble("", issueJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, issueJSON) {
		t.Error("issue JSON should be injected verbatim")
	}
}
