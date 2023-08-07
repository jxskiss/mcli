package mcli

import (
	"bytes"
	"log"
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
	cases := []struct {
		shell       string
		description string
		connector   string
	}{
		{
			description: "Bash shell suggestions",
			shell:       "bash",
			connector:   "\t",
		},
		{
			description: "zsh shell suggestions",
			shell:       "zsh",
			connector:   ":",
		},
		{
			description: "fish shell suggestions",
			shell:       "fish",
			connector:   "\t",
		},
	}
	for _, tt := range cases {
		t.Run(tt.description, func(t *testing.T) {
			resetDefaultApp()
			addTestCompletionCommands()

			var buf bytes.Buffer
			defaultApp.completionCtx.out = &buf
			defaultApp.completionCtx.shell = tt.shell

			Run("c", completionFlag, tt.shell)
			got := buf.String()
			log.Println(got)
			assert.Contains(t, got, "cmd1"+tt.connector+"A cmd1 description\n")
			assert.NotContains(t, got, "cmd2"+tt.connector+"A hidden cmd2 description")
			assert.NotContains(t, got, "completion")

			buf.Reset()
			Run("group1", "c", completionFlag, tt.shell)
			got = buf.String()
			assert.Contains(t, got, "cmd1"+tt.connector+"A group1 cmd1 description\n")
			assert.Contains(t, got, "cmd2"+tt.connector+"A group1 cmd2 description\n")
			assert.Contains(t, got, "cmd3\n")

			buf.Reset()
			Run("group1", "cme", completionFlag, tt.shell)
			got = buf.String()
			assert.Zero(t, got)

			buf.Reset()
			Run("unknown", completionFlag, tt.shell)
			got = buf.String()
			assert.Zero(t, got)
		})
	}

	noDescCases := []struct {
		shell       string
		description string
		connector   string
	}{
		{
			description: "powershell shell suggestions",
			shell:       "powershell",
			connector:   ":",
		},
	}
	for _, tt := range noDescCases {
		t.Run(tt.description, func(t *testing.T) {
			resetDefaultApp()
			addTestCompletionCommands()

			var buf bytes.Buffer
			defaultApp.completionCtx.out = &buf
			defaultApp.completionCtx.shell = tt.shell

			Run("c", completionFlag, tt.shell)
			got := buf.String()
			assert.Contains(t, got, "cmd1\n")
			assert.NotContains(t, got, "cmd2\n")
			assert.NotContains(t, got, "completion\n")

			buf.Reset()
			Run("group1", "c", completionFlag, tt.shell)
			got = buf.String()
			assert.Contains(t, got, "cmd1\n")
			assert.Contains(t, got, "cmd2\n")
			assert.Contains(t, got, "cmd3\n")

			buf.Reset()
			Run("group1", "cme", completionFlag, tt.shell)
			got = buf.String()
			assert.Zero(t, got)

			buf.Reset()
			Run("unknown", completionFlag, tt.shell)
			got = buf.String()
			assert.Zero(t, got)
		})
	}

	unsupportedCases := []struct {
		shell       string
		description string
		connector   string
	}{
		{
			description: "unknown shell suggestions",
			shell:       "off",
			connector:   " -- ",
		},
	}

	for _, tt := range unsupportedCases {
		t.Run(tt.description, func(t *testing.T) {
			resetDefaultApp()
			addTestCompletionCommands()

			var buf bytes.Buffer
			defaultApp.completionCtx.out = &buf
			defaultApp.completionCtx.shell = tt.shell

			Run("c", completionFlag, tt.shell)
			got := buf.String()
			log.Println(len(got))
			assert.Contains(t, got, "cmd1\n")
			assert.NotContains(t, got, "cmd2\n")
			assert.NotContains(t, got, "cmd3\n")
		})
	}
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
	Run("group1", "cmd3", "-", completionFlag, "zsh")
	got1 := buf.String()
	assert.Contains(t, got1, "-a:description a flag\n")
	assert.Contains(t, got1, "-1\n")
	assert.Contains(t, got1, "-b:description b flag\n")
	assert.Contains(t, got1, "-j:description j flag\n")

	reset()
	Run("group1", "cmd3", "-a", completionFlag, "zsh")
	got2 := buf.String()
	assert.Contains(t, got2, "-a:description a flag\n")
	assert.NotContains(t, got2, "-1")
	assert.Contains(t, got2, "--a1-flag")
	assert.Contains(t, got2, "--a2-flag:description b flag")

	reset()
	Run("group1", "cmd3", "--a", completionFlag, "zsh")
	got3 := buf.String()
	assert.Contains(t, got3, "--a-flag:description a flag\n")
	assert.Contains(t, got3, "--a2-flag:description b flag\n")
	assert.NotContains(t, got3, "-j")

	reset()
	Run("group1", "cmd3", "-j", completionFlag, "zsh")
	got4 := buf.String()
	assert.NotContains(t, got4, "-a")
	assert.Contains(t, got4, "-j:description j flag\n")

	reset()
	Run("group1", "cmd3", "-j", "abc", "-j", completionFlag, "zsh")
	got5 := buf.String()
	assert.Contains(t, got5, "-j:description j flag\n")

	reset()
	Run("group1", "cmd3", "-b", "5", "-j", "abc", "--", completionFlag, "zsh")
	got6 := buf.String()
	assert.Contains(t, got6, "--a-flag:description a flag\n")
	assert.Contains(t, got6, "--a1-flag\n")
	assert.NotContains(t, got6, "a2-flag")
	assert.Contains(t, got6, "--j-flag:description j flag\n")

	t.Run("noCompletion", func(t *testing.T) {
		reset()
		Run("completion", "-", completionFlag, "zsh")
		got := buf.String()
		assert.Zero(t, got)
	})
}

func TestFormatCompletion(t *testing.T) {
	cases := []struct {
		shell       string
		description string
		connector   string
	}{
		{
			description: "Bash shell suggestion",
			shell:       "bash",
			connector:   "\t",
		},
		{
			description: "zsh shell suggestion",
			shell:       "zsh",
			connector:   ":",
		},
		{
			description: "fish shell suggestion",
			shell:       "fish",
			connector:   "\t",
		},
		{
			description: "unknown shell suggestion",
			shell:       "off",
			connector:   " -- ",
		},
	}
	noDescCases := []struct {
		shell       string
		description string
		connector   string
	}{
		{
			description: "powershell shell suggestions",
			shell:       "powershell",
			connector:   ":",
		},
	}

	for _, tt := range cases {
		t.Run(tt.description, func(t *testing.T) {
			resetDefaultApp()
			addTestCompletionCommands()

			var buf bytes.Buffer
			defaultApp.completionCtx.out = &buf
			defaultApp.completionCtx.shell = tt.shell

			got := formatCompletion(defaultApp, "option", "some option description")
			assert.Equal(t, got, "option"+tt.connector+"some option description")
		})
	}
	for _, tt := range noDescCases {
		t.Run(tt.description, func(t *testing.T) {
			resetDefaultApp()
			addTestCompletionCommands()

			var buf bytes.Buffer
			defaultApp.completionCtx.out = &buf
			defaultApp.completionCtx.shell = tt.shell

			got := formatCompletion(defaultApp, "option", "")
			assert.Equal(t, got, "option")
		})
	}
}

func TestHasCompletionFlag(t *testing.T) {
	t.Run("Has more than two arguments with completion", func(t *testing.T) {
		passedArgs := []string{"cmd", completionFlag, "bash"}
		isCompletion, args, shell := hasCompletionFlag(passedArgs)
		assert.Equal(t, isCompletion, true)
		assert.Equal(t, args, []string{"cmd"})
		assert.Equal(t, shell, "bash")
	})

	t.Run("Has more than two arguments without completion", func(t *testing.T) {
		passedArgs := []string{"cmd", "--test", "bash"}
		isCompletion, args, shell := hasCompletionFlag(passedArgs)
		assert.Equal(t, isCompletion, false)
		assert.Equal(t, args, []string{"cmd", "--test", "bash"})
		assert.Equal(t, shell, "unsupported")
	})

	t.Run("Has more than one argument with completion", func(t *testing.T) {
		passedArgs := []string{completionFlag, "bash"}
		isCompletion, args, shell := hasCompletionFlag(passedArgs)
		assert.Equal(t, isCompletion, true)
		assert.Equal(t, args, []string{})
		assert.Equal(t, shell, "bash")
	})

	t.Run("Has more than one argument without completion", func(t *testing.T) {
		passedArgs := []string{"--test", "bash"}
		isCompletion, args, shell := hasCompletionFlag(passedArgs)
		assert.Equal(t, isCompletion, false)
		assert.Equal(t, args, []string{"--test", "bash"})
		assert.Equal(t, shell, "unsupported")
	})

	t.Run("Has no arguments without completion", func(t *testing.T) {
		passedArgs := []string{}
		isCompletion, args, shell := hasCompletionFlag(passedArgs)
		assert.Equal(t, isCompletion, false)
		assert.Equal(t, args, []string{})
		assert.Equal(t, shell, "unsupported")
	})

	t.Run("Has flag in between", func(t *testing.T) {
		passedArgs := []string{completionFlag, "--test", "bash"}
		isCompletion, args, shell := hasCompletionFlag(passedArgs)
		assert.Equal(t, isCompletion, true)
		assert.Equal(t, args, []string{})
		assert.Equal(t, shell, "unsupported")
	})

	t.Run("Should not panic when completion flag is last", func(t *testing.T) {
		passedArgs := []string{"com", "command2", completionFlag}
		isCompletion, args, shell := hasCompletionFlag(passedArgs)
		assert.Equal(t, isCompletion, false)
		assert.Equal(t, args, []string{"com", "command2", "--mcli-generate-completion"})
		assert.Equal(t, shell, "unsupported")
	})

	t.Run("Should not panic with only completion flag", func(t *testing.T) {
		passedArgs := []string{completionFlag}
		isCompletion, args, shell := hasCompletionFlag(passedArgs)
		assert.Equal(t, isCompletion, true)
		assert.Equal(t, args, []string{completionFlag})
		assert.Equal(t, shell, "unsupported")
	})
}
