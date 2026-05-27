package afk

import (
	"bytes"
	"encoding/json"
	"strings"
	"unicode"
)

// ParseStaticItems parses static item input as a JSON array when possible,
// otherwise as newline-delimited text.
func ParseStaticItems(input string) ([]string, error) {
	if shouldParseAsNewlineWithoutJSONAttempt(input) {
		return parseNewlineItems(input), nil
	}

	var array []json.RawMessage
	if err := json.Unmarshal([]byte(input), &array); err == nil {
		return parseJSONArrayItems(array)
	}

	return parseNewlineItems(input), nil
}

func shouldParseAsNewlineWithoutJSONAttempt(input string) bool {
	if !strings.ContainsRune(input, '\n') {
		return false
	}

	for _, r := range input {
		if unicode.IsSpace(r) {
			continue
		}
		return r != '['
	}

	return true
}

func parseJSONArrayItems(array []json.RawMessage) ([]string, error) {
	items := make([]string, 0, len(array))
	for _, element := range array {
		trimmed := bytes.TrimSpace(element)
		if len(trimmed) > 0 && trimmed[0] == '"' {
			var s string
			if err := json.Unmarshal(trimmed, &s); err != nil {
				return nil, err
			}
			items = append(items, s)
			continue
		}

		var compacted bytes.Buffer
		if err := json.Compact(&compacted, element); err != nil {
			return nil, err
		}
		items = append(items, compacted.String())
	}

	return items, nil
}

func parseNewlineItems(input string) []string {
	lines := strings.Split(input, "\n")
	items := make([]string, 0, len(lines))
	for i, line := range lines {
		if i < len(lines)-1 && strings.HasSuffix(line, "\r") {
			line = strings.TrimSuffix(line, "\r")
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		items = append(items, line)
	}
	return items
}
