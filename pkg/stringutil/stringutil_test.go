package stringutil_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/alphaflow/injector/pkg/stringutil"
)

func TestStringUtil_IsBlank_False(t *testing.T) {
	assert.False(t, stringutil.IsBlank(strings.Repeat("a", 5)), "string of chars")
}

func TestStringUtil_IsBlank(t *testing.T) {
	assert.True(t, stringutil.IsBlank(strings.Repeat(" ", 5)), "string of spaces")
	assert.True(t, stringutil.IsBlank(""), "zero length string")
}
