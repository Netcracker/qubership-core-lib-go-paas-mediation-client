package entity

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func callWriteContext() string {
	return nestedCall(5)
}

func nestedCall(n int) string {
	if n <= 0 {
		c, _ := WriteContext()
		return c
	}
	return nestedCall(n - 1)
}

func TestWriteContext(t *testing.T) {
	output := callWriteContext()

	assert.Contains(t, output, "entityWrite_test.go")
	re := regexp.MustCompile(`^[A-Za-z0-9_\-]+\.go:\d+ [A-Za-z0-9_]+\(.*\)$`)
	if !re.MatchString(output) {
		t.Errorf("output does not match expected format: %s", output)
	}
}
