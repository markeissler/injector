package stringsutil_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	stringsutil "github.com/alphaflow/injector/pkg/strings"
)

func TestStringsUtil_IsBlank_False(t *testing.T) {
	assert.False(t, stringsutil.IsBlank(strings.Repeat("a", 5)), "string of chars")
}

func TestStringsUtil_IsBlank(t *testing.T) {
	assert.True(t, stringsutil.IsBlank(strings.Repeat(" ", 5)), "string of spaces")
	assert.True(t, stringsutil.IsBlank(""), "zero length string")
}
