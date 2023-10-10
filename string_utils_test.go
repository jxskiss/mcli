package mcli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrimPrefixWithPrefix(t *testing.T) {
	text := "trimMe"
	prefix := "trim"
	expected := "Me"

	assert.Equal(t, trimPrefix(text, prefix), expected)
}

func TestTrimPrefixWithoutPrefix(t *testing.T) {
	text := "trimMe"
	prefix := ""
	expected := "trimMe"

	assert.Equal(t, trimPrefix(text, prefix), expected)
}

func TestTrimPrefixWithInfix(t *testing.T) {
	text := "trimMe"
	prefix := "imMe"
	expected := "trimMe"

	assert.Equal(t, trimPrefix(text, prefix), expected)
}
