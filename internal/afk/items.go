package afk

import (
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

	var parsed any
	if err := json.Unmarshal([]byte(input), &parsed); err == nil {
		array, isArray := parsed.([]any)
		if isArray {
			return parseJSONArrayItems(array)
		}

		// Valid JSON that is not an array falls back to newline parsing using
		// the original input bytes.
		return parseNewlineItems(input), nil
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

func parseJSONArrayItems(array []any) ([]string, error) {
	items := make([]string, 0, len(array))
	for _, element := range array {
		s, isString := element.(string)
		if isString {
			items = append(items, s)
			continue
		}

		encoded, err := json.Marshal(element)
		if err != nil {
			return nil, err
		}
		items = append(items, string(encoded))
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
