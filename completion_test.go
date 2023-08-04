package mcli

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func addTestCompletionCommands() {
	Add("cmd1", dummyCmd, "A cmd1 description")
	AddHidden("cmd2", dummyCmd, "A hidden cmd2 description")
	AddGroup("group1", "A group1 description")
	Add("group1 cmd1", dummyCmd, "A group1 cmd1 description")
	Add("group1 cmd2", dummyCmd, "A group1 cmd2 description")
	Add("group1 cmd3 sub1", dummyCmd, "A group1 cmd3 sub1 description")
	AddHelp()
	AddCompletion()
}

func TestCompletionCommand(t *testing.T) {
	resetDefaultApp()
	addTestCompletionCommands()

	Run("completion", "bash")
	Run("completion", "zsh")
	Run("completion", "powershell")
	Run("completion", "fish")
}

func TestCompletionUsage(t *testing.T) {
	resetDefaultApp()
	addTestCompletionCommands()

	bashUsage := defaultApp.completionUsage("bash")()
	assert.Contains(t, bashUsage, "USAGE:\n  mcli.test completion bash")

	zshUsage := defaultApp.completionUsage("zsh")()
	assert.Contains(t, zshUsage, "USAGE:\n  mcli.test completion zsh")

	powershellUsage := defaultApp.completionUsage("powershell")
	assert.Contains(t, powershellUsage(), "USAGE:\n  mcli.test completion powershell")

	fishUsage := defaultApp.completionUsage("fish")()
	assert.Contains(t, fishUsage, "USAGE:\n  mcli.test completion fish")
}

func TestSuggestCommands(t *testing.T) {
	resetDefaultApp()
	addTestCompletionCommands()

	var buf bytes.Buffer
	defaultApp.completionCtx.out = &buf

	Run("c", completionFlag)
	got1 := buf.String()
	assert.Contains(t, got1, "cmd1\n")
	assert.NotContains(t, got1, "cmd2")
	assert.NotContains(t, got1, "completion")

	buf.Reset()
	os.Setenv("SHELL", "/bin/zsh")
	Run("c", completionFlag)
	got2 := buf.String()
	assert.Contains(t, got2, "cmd1:A cmd1 description\n")
	assert.NotContains(t, got2, "cmd2")
	assert.NotContains(t, got2, "completion")

	buf.Reset()
	Run("group1", "c", completionFlag)
	got3 := buf.String()
	assert.Contains(t, got3, "cmd1:")
	assert.Contains(t, got3, "cmd2:")
	assert.Contains(t, got3, "cmd3\n")

	buf.Reset()
	Run("group1", "cme", completionFlag)
	got4 := buf.String()
	assert.Zero(t, got4)
}

func TestSuggestCommandWithoutAddingGroup(t *testing.T) {
	resetDefaultApp()
	Add("s", dummyCmd, "Serve with port and dir")
	AddGroup("cmd", "CMD")
	Add("cmd ox", dummyCmd, "Second serve")
	Add("cmd ax", dummyCmd, "Second serve")
	Add("ot ix", dummyCmd, "Second serve")
	Add("group3 sub1 subsub1", dummyCmd, "Group3 sub1 subsub1")

	var buf bytes.Buffer
	defaultApp.completionCtx.out = &buf

	Run("o", completionFlag)
	got1 := buf.String()
	assert.Contains(t, got1, "ot\n")

	buf.Reset()
	Run("ot", completionFlag)
	got2 := buf.String()
	assert.Contains(t, got2, "ix\n")

	buf.Reset()
	Run("group3", "s", completionFlag)
	got3 := buf.String()
	assert.Contains(t, got3, "sub1\n")

	buf.Reset()
	Run("group3", "sub1", completionFlag)
	got4 := buf.String()
	assert.Contains(t, got4, "subsub1\n")
}

func TestSuggestFlags(t *testing.T) {
	resetDefaultApp()
	addTestCompletionCommands()

	testCmd := func() {
		args := &struct {
			A  bool     `cli:"-a,  -a-flag, description a flag"`
			A1 bool     `cli:"-1,  -a1-flag"`
			B  int32    `cli:"-b,  -a2-flag, description b flag"`
			J  []string `cli:"-j,  -j-flag, description j flag"`

			ValueImpl2 flagValueImpl2 `cli:"-v, -v-flag, description v flag"`

			Args []string `cli:"some-args"`
		}{}
		Parse(args)
	}
	Add("group1 cmd3", testCmd, "A group1 cmd3 description")

	var buf bytes.Buffer
	defaultApp.completionCtx.out = &buf

	reset := func() {
		buf.Reset()
		defaultApp.resetParsingContext()
	}

	reset()
	Run("group1", "cmd3", "-", completionFlag)
	got1 := buf.String()
	assert.Contains(t, got1, "-a\n")
	assert.Contains(t, got1, "-1\n")
	assert.Contains(t, got1, "-b\n")
	assert.Contains(t, got1, "-j\n")

	// Mock zsh.
	os.Setenv("SHELL", "/bin/zsh")

	reset()
	Run("group1", "cmd3", "-a", completionFlag)
	got2 := buf.String()
	assert.Contains(t, got2, "-a:description a flag\n")
	assert.NotContains(t, got2, "-1")
	assert.Contains(t, got2, "--a1-flag")
	assert.Contains(t, got2, "--a2-flag:description b flag")

	reset()
	Run("group1", "cmd3", "--a", completionFlag)
	got3 := buf.String()
	assert.Contains(t, got3, "--a-flag:description a flag\n")
	assert.Contains(t, got3, "--a2-flag:description b flag\n")
	assert.NotContains(t, got3, "-j")

	reset()
	Run("group1", "cmd3", "-j", completionFlag)
	got4 := buf.String()
	assert.NotContains(t, got4, "-a")
	assert.Contains(t, got4, "-j:description j flag\n")

	reset()
	Run("group1", "cmd3", "-j", "abc", "-j", completionFlag)
	got5 := buf.String()
	assert.Contains(t, got5, "-j:description j flag\n")

	reset()
	Run("group1", "cmd3", "-b", "5", "-j", "abc", "--", completionFlag)
	got6 := buf.String()
	assert.Contains(t, got6, "--a-flag:description a flag\n")
	assert.Contains(t, got6, "--a1-flag\n")
	assert.NotContains(t, got6, "a2-flag")
	assert.Contains(t, got6, "--j-flag:description j flag\n")

	t.Run("noCompletion", func(t *testing.T) {
		reset()
		Run("completion", "-", completionFlag)
		got := buf.String()
		assert.Zero(t, got)
	})
}
