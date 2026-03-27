package quickstart

import _ "embed"

//go:embed quickstart.txt
var content string

// Text returns the quickstart cheatsheet content.
func Text() string {
	return content
}
