package mcli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubSlice(t *testing.T) {
	cases := []struct {
		description string
		args        []string
		expected    []string
		from        int
		to          int
		panic       bool
	}{
		{
			description: "Cuts front from slice",
			args:        []string{"a", "b", "c"},
			expected:    []string{"a"},
			from:        0,
			to:          1,
		},
		{
			description: "Cuts middle from slice",
			args:        []string{"a", "b", "c"},
			expected:    []string{"b"},
			from:        1,
			to:          2,
		},
		{
			description: "Cuts end from slice",
			args:        []string{"a", "b", "c"},
			expected:    []string{"c"},
			from:        2,
			to:          3,
		},
		{
			description: "Handles equal value",
			args:        []string{"a", "b", "c"},
			expected:    nil,
			from:        3,
			to:          3,
		},
		{
			description: "Handles from higher than end",
			args:        []string{"a", "b", "c"},
			expected:    nil,
			from:        4,
			to:          3,
		},
	}

	for _, tt := range cases {
		t.Run(tt.description, func(t *testing.T) {
			assert.Equal(t, subSlice(tt.args, tt.from, tt.to), tt.expected)
		})
	}
}

func TestRemoveCommandName(t *testing.T) {
	t.Run("match command", func(t *testing.T) {
		cmdName := "group1 cmd1 sub1"
		args := []string{"group1", "cmd1", "sub1", "-a", "-b", "12345"}
		got := removeCommandName(args, cmdName)
		assert.Equal(t, []string{"-a", "-b", "12345"}, got)
	})

	t.Run("partial match", func(t *testing.T) {
		cmdName := "group1 cmd1 sub2"
		args := []string{"group1", "cmd1", "sub1", "-a", "-b", "12345"}
		got := removeCommandName(args, cmdName)
		assert.Equal(t, []string{"sub1", "-a", "-b", "12345"}, got)
	})

	t.Run("not match", func(t *testing.T) {
		cmdName := "group2 cmd1"
		args := []string{"group1", "cmd1", "sub1", "-a", "-b", "12345"}
		got := removeCommandName(args, cmdName)
		assert.Equal(t, []string{"group1", "cmd1", "sub1", "-a", "-b", "12345"}, got)
	})
}
