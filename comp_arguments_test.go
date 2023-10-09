package mcli

import (
	"bytes"
	"flag"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArgCompletionContext(t *testing.T) {
	resetDefaultApp()
	addTestCompletionCommands()

	type globalFlags struct {
		G1 bool   `cli:"-g, --global-1"`
		G2 string `cli:"-G, --global-2"`
	}

	type testArgs struct {
		A  []string `cli:"-a, --a-flag, description a flag"`
		A1 bool     `cli:"-1, --1-flag"`
		S1 bool     `cli:"-s1, --s1-flag"`
		S2 bool     `cli:"-s2, --s2-flag"`
		V1 string   `cli:"value1"`
		V2 int64    `cli:"value2"`
	}

	argCompFuncs := map[string]ArgCompletionFunc{
		"-a": func(ctx ArgCompletionContext) []CompletionItem {
			args := ctx.CommandArgs().(*testArgs)
			if args.S1 {
				return nil
			}
			return []CompletionItem{
				{"abc", "abc description"},
				{"def", "def description"},
			}
		},
		"value1": func(ctx ArgCompletionContext) []CompletionItem {
			gf := ctx.GlobalFlags().(*globalFlags)
			if gf.G1 {
				return []CompletionItem{
					{"Tom", "Tom who"},
					{"John", "John Smith"},
				}
			}
			if gf.G2 == "hello" {
				return []CompletionItem{
					{"world", "Hello world!"},
					{"there", "Hello there!"},
				}
			}
			return []CompletionItem{
				{"dummy-value-1", ""},
			}
		},
		"value2": func(ctx ArgCompletionContext) []CompletionItem {
			fs := ctx.FlagSet()
			prefix := ctx.ArgPrefix()
			compItems := []CompletionItem{
				{"abc-1", "abc 1 description"},
				{"abc-2", "abc 2 description"},
				{"def-3", "def 3 description"},
				{"def-4", "def 4 description"},
			}
			if fs.Lookup("s2").Value.(flag.Getter).Get() == true {
				var result []CompletionItem
				for _, x := range compItems {
					if strings.HasPrefix(x.Value, prefix) {
						result = append(result, x)
					}
				}
				return result
			}
			return compItems
		},
	}
	cmdFunc := func(ctx *Context, args *testArgs) {
		// pass
	}
	testCmd := NewCommand(cmdFunc, WithArgCompFuncs(argCompFuncs))

	defaultApp.SetGlobalFlags(&globalFlags{})
	Add("group1 cmdv", testCmd, "A group1 cmdv description")

	var buf bytes.Buffer
	defaultApp.completionCtx.out = &buf

	reset := func() {
		buf.Reset()
		defaultApp.resetParsingContext()
		defaultApp.resetCompletionCtx()
		defaultApp.SetGlobalFlags(&globalFlags{})
	}

	t.Run("suggest flags", func(t *testing.T) {
		reset()
		Run("group1", "cmdv", "-", completionFlag, "zsh")
		got1 := buf.String()
		assert.Contains(t, got1, "-a:description a flag\n")
		assert.Contains(t, got1, "-1\n")
		assert.Contains(t, got1, "--s1:--s1-flag\n")
		assert.Contains(t, got1, "--s2:--s2-flag\n")
	})

	t.Run("suggest flag value / s1 true", func(t *testing.T) {
		reset()
		Run("group1", "cmdv", "-s1", "-a", "", completionFlag, "zsh")
		got1 := buf.String()
		assert.Equal(t, got1, "")
	})

	t.Run("suggest flag value / s1 false", func(t *testing.T) {
		reset()
		Run("group1", "cmdv", "-a", "", completionFlag, "zsh")
		got1 := buf.String()
		assert.Contains(t, got1, "abc:abc description\n")
		assert.Contains(t, got1, "def:def description\n")
	})

	t.Run("suggest value1 / g1 true", func(t *testing.T) {
		reset()
		Run("group1", "cmdv", "-g", "", completionFlag, "zsh")
		got1 := buf.String()
		assert.Contains(t, got1, "Tom:Tom who\n")
		assert.Contains(t, got1, "John:John Smith\n")
	})

	t.Run("suggest value1 / g2 hello", func(t *testing.T) {
		reset()
		Run("group1", "cmdv", "-G", "hello", "", completionFlag, "zsh")
		got1 := buf.String()
		assert.Contains(t, got1, "world:Hello world!\n")
		assert.Contains(t, got1, "there:Hello there!\n")
	})

	t.Run("suggest value1 / default", func(t *testing.T) {
		reset()
		Run("group1", "cmdv", "-g=0", "-G", "there", "", completionFlag, "zsh")
		got1 := buf.String()
		assert.Contains(t, got1, "dummy-value-1\n")
	})

	t.Run("suggest value2 / s2 true", func(t *testing.T) {
		reset()
		Run("group1", "cmdv", "-s2=1", "value1 value", "", completionFlag, "zsh")
		got1 := buf.String()
		assert.Contains(t, got1, "abc-1:abc 1 description\n")
		assert.Contains(t, got1, "abc-2:abc 2 description\n")
		assert.Contains(t, got1, "def-3:def 3 description\n")
		assert.Contains(t, got1, "def-4:def 4 description\n")

		reset()
		Run("group1", "cmdv", "-s2", "value1 value", "abc", completionFlag, "zsh")
		got2 := buf.String()
		assert.Contains(t, got2, "abc-1:abc 1 description\n")
		assert.Contains(t, got2, "abc-2:abc 2 description\n")
		assert.NotContains(t, got2, "def-3")
		assert.NotContains(t, got2, "def-4")

		reset()
		Run("group1", "cmdv", "-s2", "value1 value", "abd", completionFlag, "zsh")
		got3 := buf.String()
		assert.Equal(t, got3, "")
	})

	t.Run("suggest value2 / s2 false", func(t *testing.T) {
		reset()
		Run("group1", "cmdv", "value1 value", "", completionFlag, "zsh")
		got1 := buf.String()
		assert.Contains(t, got1, "abc-1:abc 1 description\n")
		assert.Contains(t, got1, "abc-2:abc 2 description\n")
		assert.Contains(t, got1, "def-3:def 3 description\n")
		assert.Contains(t, got1, "def-4:def 4 description\n")

		reset()
		Run("group1", "cmdv", "value1 value", "abc", completionFlag, "zsh")
		got2 := buf.String()
		assert.Contains(t, got2, "abc-1:abc 1 description\n")
		assert.Contains(t, got2, "abc-2:abc 2 description\n")
		assert.Contains(t, got2, "def-3:def 3 description\n")
		assert.Contains(t, got2, "def-4:def 4 description\n")
	})

}
