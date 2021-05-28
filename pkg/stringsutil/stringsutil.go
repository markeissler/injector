package stringsutil

import "strings"

// IsBlank returns true if a string has a non-zero length and doesn't contain only spaces.
func IsBlank(s string) bool {
	return strings.TrimSpace(s) == ""
}
