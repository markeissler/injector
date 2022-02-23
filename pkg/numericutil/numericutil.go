package numericutil

import "github.com/markeissler/injector/pkg/stringutil"

// BoolToInt returns `1` for a `true` boolean value and `0` for a false boolean value.
func BoolToInt(b bool) int {
	if b {
		return 1
	}

	return 0
}

// StringToBool returns `true` for a non-blank string and `false` for a blank string.
func StringToBool(s string) bool {
	return !stringutil.IsBlank(s)
}

// StringToBoolInt returns `1` for a non-blank string and `0` for a blank string.
func StringToBoolInt(s string) int {
	return BoolToInt(StringToBool(s))
}
