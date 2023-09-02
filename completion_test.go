package mcli

import (
	"bytes"
	"fmt"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

func addTestCompletionCommands() {
	defaultApp.Options.EnableFlagCompletionForAllCommands = true
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

	defaultApp.resetParsingContext()
	Run("completion", "bash")

	defaultApp.resetParsingContext()
	Run("completion", "zsh")

	defaultApp.resetParsingContext()
	Run("completion", "powershell")

	defaultApp.resetParsingContext()
	Run("completion", "fish")
}

func TestCompletionUsage(t *testing.T) {
	resetDefaultApp()
	addTestCompletionCommands()

	for _, shellType := range []string{
		"bash", "zsh", "fish", "powershell",
	} {
		usage := defaultApp.completionUsage(shellType)()
		want := fmt.Sprintf("USAGE:\n  %s completion %s", getProgramName(), shellType)
		assert.Contains(t, usage, want)
	}
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
	Add("group1 cmd3", testCmd, "A group1 cmd3 description",
		EnableFlagCompletion())
	Add("group1 cmd3 sub2", testCmd, "A group1 cmd3 sub2 description",
		EnableFlagCompletion())

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

	// INFO: no longer valid | 2023-09-01
	// t.Run("without trailing hyphen", func(t *testing.T) {
	// 	reset()
	// 	Run("group1", "cmd3", "-b", "5", completionFlag, "zsh")
	// 	got := buf.String()
	// 	assert.Contains(t, got, "-a:description a flag\n")
	// 	assert.NotContains(t, got, "description b flag")
	// 	assert.Contains(t, got, "-j:description j flag\n")
	// })

	t.Run("leaf command", func(t *testing.T) {
		reset()
		Run("group1", "cmd3", "sub2", "-", completionFlag, "zsh")
		got := buf.String()
		assert.Contains(t, got, "-a:description a flag\n")
		assert.Contains(t, got, "-1\n")
		assert.Contains(t, got, "-b:description b flag\n")
		assert.Contains(t, got, "-j:description j flag\n")
	})

	t.Run("noCompletion", func(t *testing.T) {
		reset()
		Run("completion", "-", completionFlag, "zsh")
		got := buf.String()
		assert.Zero(t, got)
	})
}

func flagArguments(ctx ArgCompletionContext) ([][]string, ShellCompDirective) {
	// fmt.Printf("%s, %#v\n", config, ctx.Args())
	return [][]string{
		{"alfa", "description alfa"},
		{"beta", "description beta"},
	}, ShellCompDirectiveFilterDirs
}

func TestSuggestFlagArgs(t *testing.T) {
	resetDefaultApp()
	addTestCompletionCommands()

	funcs := make(map[string]ArgCompletionFunc)
	funcs["flagArguments"] = flagArguments

	testCmd := func() {
		args := &struct {
			A  bool `cli:"-a, --a-flag, description a flag" cmpl:"flagArguments"`
			A1 bool `cli:"-1, --a1-flag"`
		}{}
		Parse(args, WithArgCompFuncs(funcs))
	}
	Add("group1 cmd3", testCmd, "A group1 cmd3 description",
		EnableFlagCompletion())

	var buf bytes.Buffer
	defaultApp.completionCtx.out = &buf

	reset := func() {
		buf.Reset()
		defaultApp.resetParsingContext()
	}

	reset()
	Run("group1", "cmd3", "-a", completionFlag, "zsh")
	flagWithFunction := buf.String()
	assert.Equal(t, flagWithFunction, "alfa:description alfa\nbeta:description beta\n")

	reset()
	Run("group1", "cmd3", "--a-flag", completionFlag, "zsh")
	flagWithFunction = buf.String()
	assert.Equal(t, flagWithFunction, "alfa:description alfa\nbeta:description beta\n")

	// TODO: bug? suggestion of same flag right after? when one? | 2023-09-01
	reset()
	Run("group1", "cmd3", "-1", completionFlag, "zsh")
	flagWoWithFunction := buf.String()
	assert.Equal(t, flagWoWithFunction, "-1\n")

	reset()
	Run("group1", "cmd3", "--a1-flag", completionFlag, "zsh")
	flagWoWithFunction = buf.String()
	assert.Equal(t, flagWoWithFunction, "--a1-flag\n")
}

func commandArguments(ctx ArgCompletionContext) ([][]string, ShellCompDirective) {
	// fmt.Printf("%s, %#v\n", config, ctx.Args())
	return [][]string{
		{"value a", "description of value a"},
		{"value b", "description of value b"},
	}, ShellCompDirectiveFilterDirs
}

func TestSuggestCommandArgs(t *testing.T) {
	resetDefaultApp()
	addTestCompletionCommands()

	testCmd := func() {
		args := &struct {
			A  bool `cli:"-a, --a-flag, description a flag"`
			A1 bool `cli:"-1, --a1-flag"`
		}{}
		Parse(args)
	}
	Add("group1 cmdv", testCmd, "A group1 cmd2 description",
		WithCommandCompFunc(commandArguments),
		EnableFlagCompletion(),
	)
	Add("group1 cmd3", testCmd, "A group1 cmd3 description",
		EnableFlagCompletion(),
	)

	var buf bytes.Buffer
	defaultApp.completionCtx.out = &buf

	reset := func() {
		buf.Reset()
		defaultApp.resetParsingContext()
	}

	reset()
	Run("group1", "cmdv", completionFlag, "zsh")
	commandWithFunction := buf.String()
	assert.Equal(t, commandWithFunction, "value a:description of value a\nvalue b:description of value b\n")

	reset()
	Run("group1", "cmdv", "value a", completionFlag, "zsh")
	commandWithFunctionAgain := buf.String()
	assert.Equal(t, commandWithFunctionAgain, "value a:description of value a\nvalue b:description of value b\n")

	reset()
	Run("group1", "cmd3", completionFlag, "zsh")
	commandWoWithFunction := buf.String()
	assert.Equal(t, commandWoWithFunction, "")
}

func TestSuggestArgsMixed(t *testing.T) {
	resetDefaultApp()
	addTestCompletionCommands()

	funcs := make(map[string]ArgCompletionFunc)
	funcs["flagArguments"] = flagArguments

	testCmd := func() {
		args := &struct {
			A  []string `cli:"-a, --a-flag, description a flag" cmpl:"flagArguments"`
			A1 bool     `cli:"-1, --1-flag"`
			S1 bool     `cli:"-s0, --s0-flag" cmpl:"flagArguments"`
			S2 bool     `cli:"-s1, --s1-flag" cmpl:"flagArguments"`
		}{}
		Parse(args, WithArgCompFuncs(funcs))
	}
	Add("group1 cmdv", testCmd, "A group1 cmd2 description",
		WithCommandCompFunc(commandArguments),
		EnableFlagCompletion(),
	)
	Add("group1 cmd3", testCmd, "A group1 cmd3 description",
		EnableFlagCompletion(),
	)

	var buf bytes.Buffer
	defaultApp.completionCtx.out = &buf

	reset := func() {
		buf.Reset()
		defaultApp.resetParsingContext()
	}

	reset()
	Run("group1", "cmdv", completionFlag, "zsh")
	got1 := buf.String()
	assert.Equal(t, got1, "value a:description of value a\nvalue b:description of value b\n")

	reset()
	Run("group1", "cmdv", "value a", completionFlag, "zsh")
	got2 := buf.String()
	assert.Equal(t, got2, "value a:description of value a\nvalue b:description of value b\n")

	reset()
	Run("group1", "cmdv", "value a", "-a", completionFlag, "zsh")
	got3 := buf.String()
	assert.Equal(t, got3, "alfa:description alfa\nbeta:description beta\n")

	reset()
	Run("group1", "cmdv", "value a", "-a", "alfa", completionFlag, "zsh")
	got4 := buf.String()
	assert.Equal(t, got4, "value a:description of value a\nvalue b:description of value b\n")

	reset()
	Run("group1", "cmdv", "value a", "-a", "alfa", "-a", completionFlag, "zsh")
	got5 := buf.String()
	assert.Equal(t, got5, "alfa:description alfa\nbeta:description beta\n")

	// very simiar flags, flag completion instead of value because of similarity
	reset()
	Run("group1", "cmdv", "value a", "-a", "alfa", "-a", "alfa", "-s", completionFlag, "zsh")
	got6 := buf.String()
	assert.Equal(t, got6, "--s0:--s0-flag\n--s1:--s1-flag\n")
}

func TestSuggestFlagKeyNotMatching(t *testing.T) {
	resetDefaultApp()
	addTestCompletionCommands()

	funcs := make(map[string]ArgCompletionFunc)
	funcs["flagArguments"] = flagArguments

	testCmd := func() {
		args := &struct {
			A  []string `cli:"-a, --a-flag, description a flag" cmpl:"flagArgumentsX"`
			A1 bool     `cli:"-1, --1-flag"`
			S1 bool     `cli:"-s0, --s0-flag" cmpl:"flagArguments"`
			S2 bool     `cli:"-s1, --s1-flag" cmpl:"flagArguments"`
		}{}
		Parse(args, WithArgCompFuncs(funcs))
	}
	Add("group1 cmdv", testCmd, "A group1 cmd2 description",
		WithCommandCompFunc(commandArguments),
		EnableFlagCompletion(),
	)
	Add("group1 cmd3", testCmd, "A group1 cmd3 description",
		EnableFlagCompletion(),
	)

	var buf bytes.Buffer
	defaultApp.completionCtx.out = &buf

	reset := func() {
		buf.Reset()
		defaultApp.resetParsingContext()
	}

	reset()
	assert.PanicsWithValue(
		t,
		"mcli: flag argument completion called not passed function 'flagArgumentsX'",
		func() {
			Run("group1", "cmdv", "-a", completionFlag, "zsh")
		},
	)
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
		{
			description: "unknown shell suggestion",
			shell:       "off",
			connector:   " -- ",
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

func TestCompletedCommand(t *testing.T) {
	cases := []struct {
		args        []string
		description string
		command     *Command
	}{
		{
			args:        []string{"group1"},
			description: "Catches command group to suggest command",
			command:     defaultApp.cmds[2],
		},
		{
			args:        []string{"group1", "cmd3"},
			description: "Catches longest command to suggest command",
			command:     defaultApp.cmds[2],
		},
		{
			args:        []string{"group1", "cmd3", "sub1"},
			description: "Catches longest command to suggest command args",
			command:     defaultApp.cmds[5],
		},
		{
			args:        []string{"cmd1"},
			description: "Catches longest command to suggest command args",
			command:     defaultApp.cmds[0],
		},
		{
			args:        []string{"cmdX"},
			description: "Command not found",
		},
	}
	for _, tt := range cases {
		t.Run(tt.description, func(t *testing.T) {
			cm := completedCommand(tt.args, defaultApp.cmds)
			assert.Equal(t, tt.command, cm)
		})
	}
}

func TestDetectCompletionMethod(t *testing.T) {
	cases := []struct {
		args        []string
		description string
		cm          completionMethod
	}{
		{
			args:        []string{},
			description: "Executes without args",
			cm: completionMethod{
				userArgs: []string{},
			},
		},
		{
			args:        []string{"-"},
			description: "Detects flag when last arg is hyphen",
			cm: completionMethod{
				isFlag:   true,
				flagName: "-",
				userArgs: []string{},
			},
		},
		{
			args:        []string{"test", "-"},
			description: "Detects flag when last arg is hyphen 2",
			cm: completionMethod{
				isFlag:   true,
				flagName: "-",
				userArgs: []string{"test"},
			},
		},
		{
			args:        []string{"test", "-a"},
			description: "Suggests named flag arguments",
			cm: completionMethod{
				isFlagValue: true,
				flagName:    "-a",
				userArgs:    []string{"test", "-a"},
			},
		},
		{
			args:        []string{"group1"},
			description: "Catches command group to suggest command",
			cm: completionMethod{
				isCommand:    true,
				foundCommand: defaultApp.cmds[2],
				userArgs:     []string{"group1"},
			},
		},
		{
			args:        []string{"group1", "cmd3"},
			description: "Catches longest command to suggest command",
			cm: completionMethod{
				isCommand:    true,
				foundCommand: defaultApp.cmds[2],
				userArgs:     []string{"group1", "cmd3"},
			},
		},
		{
			args:        []string{"group1", "cmd3", "sub1"},
			description: "Catches longest command to suggest command args",
			cm: completionMethod{
				isCommandArg: true,
				foundCommand: defaultApp.cmds[5],
				userArgs:     []string{"group1", "cmd3", "sub1"},
			},
		},
		{
			args:        []string{"cmd1"},
			description: "Catches longest command to suggest command args",
			cm: completionMethod{
				isCommandArg: true,
				foundCommand: defaultApp.cmds[0],
				userArgs:     []string{"cmd1"},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.description, func(t *testing.T) {
			cm := detectCompletionMethod(tt.args, defaultApp.cmds)
			assert.Equal(t, tt.cm, cm)
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
		assert.Equal(t, isCompletion, true)
		assert.Equal(t, args, []string{"com", "command2"})
		assert.Equal(t, shell, "unsupported")
	})

	t.Run("Should not panic with only completion flag", func(t *testing.T) {
		passedArgs := []string{completionFlag}
		isCompletion, args, shell := hasCompletionFlag(passedArgs)
		assert.Equal(t, isCompletion, true)
		assert.Equal(t, args, []string{})
		assert.Equal(t, shell, "unsupported")
	})

	t.Run("Should handle missing shell value as unsupported", func(t *testing.T) {
		passedArgs := []string{"c", completionFlag}
		isCompletion, args, shell := hasCompletionFlag(passedArgs)
		assert.Equal(t, isCompletion, true)
		assert.Equal(t, args, []string{"c"})
		assert.Equal(t, shell, "unsupported")
	})
}
