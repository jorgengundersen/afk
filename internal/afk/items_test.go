package afk

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestParseStaticItems_JSONTopLevelArray(t *testing.T) {
	items, err := ParseStaticItems("[\"alpha\", {\"b\":2,\"a\":1}, [1, 2], true, null]")
	if err != nil {
		t.Fatalf("ParseStaticItems() error = %v", err)
	}

	if len(items) != 5 {
		t.Fatalf("len(items) = %d, want 5", len(items))
	}

	if items[0] != "alpha" {
		t.Fatalf("items[0] = %q, want %q", items[0], "alpha")
	}

	var gotObject map[string]any
	if err := json.Unmarshal([]byte(items[1]), &gotObject); err != nil {
		t.Fatalf("items[1] is not valid JSON object: %q (%v)", items[1], err)
	}
	wantObject := map[string]any{"a": float64(1), "b": float64(2)}
	if !reflect.DeepEqual(gotObject, wantObject) {
		t.Fatalf("items[1] object = %#v, want %#v", gotObject, wantObject)
	}

	if items[2] != "[1,2]" {
		t.Fatalf("items[2] = %q, want %q", items[2], "[1,2]")
	}
	if items[3] != "true" {
		t.Fatalf("items[3] = %q, want %q", items[3], "true")
	}
	if items[4] != "null" {
		t.Fatalf("items[4] = %q, want %q", items[4], "null")
	}
}

func TestParseStaticItems_JSONNumbersKeepOriginalValueWhenCompacted(t *testing.T) {
	items, err := ParseStaticItems("[9007199254740993, 1.0, 1e+21]")
	if err != nil {
		t.Fatalf("ParseStaticItems() error = %v", err)
	}

	want := []string{"9007199254740993", "1.0", "1e+21"}
	if !reflect.DeepEqual(items, want) {
		t.Fatalf("items = %#v, want %#v", items, want)
	}
}

func TestParseStaticItems_EmptyJSONArrayMeansNoItems(t *testing.T) {
	items, err := ParseStaticItems("[]")
	if err != nil {
		t.Fatalf("ParseStaticItems() error = %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("len(items) = %d, want 0", len(items))
	}
}

func TestParseStaticItems_ValidNonArrayJSONFallsBackToNewlineParsingOfOriginalInput(t *testing.T) {
	input := "{\n  \"id\": 7\n}\n"

	items, err := ParseStaticItems(input)
	if err != nil {
		t.Fatalf("ParseStaticItems() error = %v", err)
	}

	want := []string{"{", "  \"id\": 7", "}"}
	if !reflect.DeepEqual(items, want) {
		t.Fatalf("items = %#v, want %#v", items, want)
	}
}

func TestParseStaticItems_NewlineParsingPreservesStandaloneCarriageReturnWithoutLineFeed(t *testing.T) {
	items, err := ParseStaticItems("alpha\rbeta\r")
	if err != nil {
		t.Fatalf("ParseStaticItems() error = %v", err)
	}

	want := []string{"alpha\rbeta\r"}
	if !reflect.DeepEqual(items, want) {
		t.Fatalf("items = %#v, want %#v", items, want)
	}
}

func TestParseStaticItems_NewlineParsingRemovesOnlyTheCRFromCRLFLineEndings(t *testing.T) {
	items, err := ParseStaticItems("a\r\nb\r\r\nc\n")
	if err != nil {
		t.Fatalf("ParseStaticItems() error = %v", err)
	}

	want := []string{"a", "b\r", "c"}
	if !reflect.DeepEqual(items, want) {
		t.Fatalf("items = %#v, want %#v", items, want)
	}
}

func TestParseStaticItems_NewlineParsingIgnoresBlankUnicodeWhitespaceLines(t *testing.T) {
	items, err := ParseStaticItems("\n\t\n\u2003\n keep \n")
	if err != nil {
		t.Fatalf("ParseStaticItems() error = %v", err)
	}

	want := []string{" keep "}
	if !reflect.DeepEqual(items, want) {
		t.Fatalf("items = %#v, want %#v", items, want)
	}
}

func TestShouldParseAsNewlineWithoutJSONAttempt(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "newline-delimited items", input: "alpha\nbeta\n", want: true},
		{name: "whitespace-only with newline", input: "\n\t\u2003\n", want: true},
		{name: "json array candidate", input: "[\"alpha\",\"beta\"]", want: false},
		{name: "pretty json array candidate", input: " \n [\"alpha\",\n\"beta\"]\n", want: false},
		{name: "single-line scalar", input: "alpha", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldParseAsNewlineWithoutJSONAttempt(tc.input)
			if got != tc.want {
				t.Fatalf("shouldParseAsNewlineWithoutJSONAttempt(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}
