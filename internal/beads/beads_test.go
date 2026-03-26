package beads

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestReady_ParsesIssues(t *testing.T) {
	issues := []map[string]any{
		{
			"id":          "afk-1",
			"title":       "Fix bug",
			"description": "Something broke",
			"status":      "open",
			"priority":    1,
			"issue_type":  "bug",
			"owner":       "test@test.com",
		},
		{
			"id":          "afk-2",
			"title":       "Add feature",
			"description": "New thing",
			"status":      "open",
			"priority":    2,
			"issue_type":  "feature",
			"owner":       "test@test.com",
		},
	}
	raw, _ := json.Marshal(issues)

	c := Client{
		run: func(args []string) ([]byte, error) {
			return raw, nil
		},
	}

	result, err := c.Ready()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("got %d issues, want 2", len(result))
	}
	if result[0].ID != "afk-1" {
		t.Errorf("ID = %q, want %q", result[0].ID, "afk-1")
	}
	if result[0].Title != "Fix bug" {
		t.Errorf("Title = %q, want %q", result[0].Title, "Fix bug")
	}
	if result[0].Priority != 1 {
		t.Errorf("Priority = %d, want 1", result[0].Priority)
	}
	if result[1].ID != "afk-2" {
		t.Errorf("ID = %q, want %q", result[1].ID, "afk-2")
	}
}

func TestReady_PreservesRawJSON(t *testing.T) {
	issues := []map[string]any{
		{"id": "afk-1", "title": "Bug", "description": "", "status": "open", "priority": 1, "issue_type": "bug", "owner": ""},
	}
	raw, _ := json.Marshal(issues)

	c := Client{
		run: func(args []string) ([]byte, error) {
			return raw, nil
		},
	}

	result, err := c.Ready()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result[0].RawJSON) == 0 {
		t.Error("RawJSON should not be empty")
	}
	// Should be valid JSON
	var m map[string]any
	if err := json.Unmarshal(result[0].RawJSON, &m); err != nil {
		t.Errorf("RawJSON is not valid JSON: %v", err)
	}
}

func TestReady_EmptyArray(t *testing.T) {
	c := Client{
		run: func(args []string) ([]byte, error) {
			return []byte("[]"), nil
		},
	}

	result, err := c.Ready()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("got %d issues, want 0", len(result))
	}
}

func TestReady_SubprocessError(t *testing.T) {
	c := Client{
		run: func(args []string) ([]byte, error) {
			return nil, errors.New("exit status 1")
		},
	}

	_, err := c.Ready()
	if err == nil {
		t.Fatal("expected error for subprocess failure")
	}
}

func TestReady_MalformedJSON(t *testing.T) {
	c := Client{
		run: func(args []string) ([]byte, error) {
			return []byte("not json"), nil
		},
	}

	_, err := c.Ready()
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestReady_LabelFilters(t *testing.T) {
	var capturedArgs []string
	c := Client{
		Labels:    []string{"backend", "infra"},
		LabelsAny: []string{"bugfix"},
		run: func(args []string) ([]byte, error) {
			capturedArgs = args
			return []byte("[]"), nil
		},
	}

	_, err := c.Ready()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should contain: ready --json --label backend --label infra --label-any bugfix
	expected := []string{"ready", "--json", "--label", "backend", "--label", "infra", "--label-any", "bugfix"}
	if len(capturedArgs) != len(expected) {
		t.Fatalf("args = %v, want %v", capturedArgs, expected)
	}
	for i, want := range expected {
		if capturedArgs[i] != want {
			t.Errorf("args[%d] = %q, want %q", i, capturedArgs[i], want)
		}
	}
}

func TestReady_NoLabelFilters(t *testing.T) {
	var capturedArgs []string
	c := Client{
		run: func(args []string) ([]byte, error) {
			capturedArgs = args
			return []byte("[]"), nil
		},
	}

	_, _ = c.Ready()

	expected := []string{"ready", "--json"}
	if len(capturedArgs) != len(expected) {
		t.Fatalf("args = %v, want %v", capturedArgs, expected)
	}
}

func TestCheckBinary_NotInPath(t *testing.T) {
	err := CheckBinary(func(name string) (string, error) {
		return "", errors.New("not found")
	})
	if err == nil {
		t.Fatal("expected error when bd not in PATH")
	}
	want := `beads: binary "bd" not found in PATH`
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestCheckBinary_Found(t *testing.T) {
	err := CheckBinary(func(name string) (string, error) {
		return "/usr/bin/bd", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
