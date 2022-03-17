package mcli

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func resetState() {
	state.cmds = nil
	state.parsingContext = nil
	for _, env := range os.Environ() {
		key := strings.SplitN(env, "=", 2)[0]
		os.Unsetenv(key)
	}
}

func dummyCmd() {
	PrintHelp()
}

func TestAddCommands(t *testing.T) {
	resetState()
	Add("cmd1", dummyCmd, "A cmd1 description")
	AddHidden("cmd2", dummyCmd, "A hidden cmd2 description")
	AddGroup("group1", "A group1 description")
	Add("group1 cmd1", dummyCmd, "A group1 cmd1 description")
	Add("group1 cmd2", dummyCmd, "A group1 cmd2 description")
	Add("group1 cmd3 sub1", dummyCmd, "A group1 cmd3 sub1 description")

	assert.Equal(t, 6, len(state.cmds))
	assert.Nil(t, state.parsingContext)
}

func TestParsing_WithoutCallingRun(t *testing.T) {
	resetState()
	var args struct {
		A bool  `cli:"-a, -a-flag, description a flag"`
		B bool  `cli:"-b, description b flag" default:"true"`
		C int32 `cli:"-c-flag, description c flag"`
	}
	Parse(&args, WithArgs([]string{"-a", "-c-flag", "12345"}))

	// assert we don't modify the global state
	assert.Equal(t, 0, len(state.cmds))
	assert.Nil(t, state.parsingContext)

	// assert the arg values
	assert.True(t, args.A)
	assert.True(t, args.B)
	assert.Equal(t, args.C, int32(12345))
}

func TestParsing_CheckFlagSetValues(t *testing.T) {
	resetState()
	var args struct {
		A  bool     `cli:"-a,  -a-flag, description a flag"`
		A1 bool     `cli:"-1, -a1-flag"`
		B  int32    `cli:"-b,  -b-flag, description b flag"`
		C  int64    `cli:"-c, --c-flag, description c flag"`
		D  float32  `cli:"-D, --d-flag, description d flag"`
		E  float64  `cli:"-E, --e-flag, description e flag"`
		F  string   `cli:"-f,  -f-flag, description f flag"`
		G  uint     `cli:"-g, --g-flag, description g flag"`
		H  []bool   `cli:"-H, --h-flag, description h flag"`
		I  []uint   `cli:"-i,  -i-flag, description i flag"`
		J  []string `cli:"-j,  -j-flag, description j flag"`

		ValueImpl2 flagValueImpl2 `cli:"-v, -v-flag, description v flag"`

		Args []string `cli:"some-args"`
	}
	fs, err := Parse(&args, WithArgs([]string{
		"-a-flag",
		"-1",
		"-b", "1",
		"-c-flag", "2",
		"-D", "3",
		"-e-flag", "4",
		"-f", "fstr",
		"-g-flag", "5",
		"-H", "true",
		"-H", "F",
		"-H", "1",
		"-H", "0",
		"-i", "5",
		"-i-flag", "6",
		"-i", "7",
		"-i-flag", "8",
		"-j-flag", "j1",
		"-j-flag", "j2",
		"-j-flag", "j,3",
		"-j-flag", "j,4,5",
		"-v", "abc",
		"-v", "123",

		"some-args 0",
		"some-args 1",
	}))
	assert.Nil(t, err)

	assert.Equal(t, true, args.A)
	assert.Equal(t, true, args.A1)
	assert.Equal(t, int32(1), args.B)
	assert.Equal(t, int64(2), args.C)
	assert.Equal(t, float32(3), args.D)
	assert.Equal(t, float64(4), args.E)
	assert.Equal(t, "fstr", args.F)
	assert.Equal(t, uint(5), args.G)
	assert.Equal(t, []bool{true, false, true, false}, args.H)
	assert.Equal(t, []uint{5, 6, 7, 8}, args.I)
	assert.Equal(t, []string{"j1", "j2", `j,3`, `j,4,5`}, args.J)
	assert.Equal(t, []string{"some-args 0", "some-args 1"}, args.Args)
	assert.Equal(t, []byte("abc123"), args.ValueImpl2.Data)

	flagCount := 12 * 2
	fs.Visit(func(flag *flag.Flag) {
		flagCount--
	})
	assert.Zero(t, flagCount)
	for _, tt := range []struct {
		flag  string
		want  string
		value interface{}
	}{
		{"a", "true", true},
		{"a-flag", "true", true},
		{"1", "true", true},
		{"a1-flag", "true", true},
		{"b", "1", int32(1)},
		{"b-flag", "1", int32(1)},
		{"c", "2", int64(2)},
		{"c-flag", "2", int64(2)},
		{"D", "3", float32(3)},
		{"d-flag", "3", float32(3)},
		{"E", "4", float64(4)},
		{"e-flag", "4", float64(4)},
		{"f", "fstr", "fstr"},
		{"f-flag", "fstr", "fstr"},
		{"g", "5", uint(5)},
		{"g-flag", "5", uint(5)},
		{"H", "[true,false,true,false]", []bool{true, false, true, false}},
		{"h-flag", "[true,false,true,false]", []bool{true, false, true, false}},
		{"i", "[5,6,7,8]", []uint{5, 6, 7, 8}},
		{"i-flag", "[5,6,7,8]", []uint{5, 6, 7, 8}},
		{"j", `["j1","j2","j,3","j,4,5"]`, []string{"j1", "j2", "j,3", "j,4,5"}},
		{"j-flag", `["j1","j2","j,3","j,4,5"]`, []string{"j1", "j2", "j,3", "j,4,5"}},
		{"v", "flagValueImpl2", []byte("abc123")},
		{"v-flag", "flagValueImpl2", []byte("abc123")},
	} {
		got := fs.Lookup(tt.flag).Value.String()
		assert.Equalf(t, tt.want, got, "flag= %v", tt.flag)
		gotValue := fs.Lookup(tt.flag).Value.(flag.Getter).Get()
		assert.Equalf(t, tt.value, gotValue, "flag= %v", tt.flag)
	}
}

func TestParsing_DefaultValues(t *testing.T) {
	resetState()
	var args struct {
		A1 bool `cli:"-a1"` // default false
		A2 bool `cli:"-a2" default:"true"`
		A3 bool `cli:"-a3" default:"true"`

		B1 int `cli:"-b1"`
		B2 int `cli:"-b2" default:"1024"`
		B3 int `cli:"-b3" default:"1024"`

		S1 string `cli:"-s1"`
		S2 string `cli:"-s2" default:"s2default"`
		S3 string `cli:"-s3" default:"s3default"`

		Slice1 []string `cli:"-slice1"`
		Slice2 []string `cli:"-slice2"`

		Arg1 []int `cli:"arg1"`
		Arg2 []int `cli:"arg2"`
	}
	_, err := Parse(&args, WithArgs([]string{
		"-a1",
		"-a2=0",
		"-b2", "2048",
		"-s2=s2arg",
		"-slice2", "d",
		"-slice2", "e",
		"-slice2", "f",
		"1", "2", "3",
	}), WithErrorHandling(flag.ContinueOnError))
	assert.Nil(t, err)

	assert.Equal(t, true, args.A1)
	assert.Equal(t, false, args.A2) // override
	assert.Equal(t, true, args.A3)  // default
	assert.Equal(t, 0, args.B1)
	assert.Equal(t, 2048, args.B2) // override
	assert.Equal(t, 1024, args.B3) // default
	assert.Equal(t, "", args.S1)
	assert.Equal(t, "s2arg", args.S2)     // override
	assert.Equal(t, "s3default", args.S3) // default
	assert.Nil(t, args.Slice1)
	assert.Equal(t, []string{"d", "e", "f"}, args.Slice2)

	assert.Equal(t, []int{1, 2, 3}, args.Arg1) // from input args
	assert.Nil(t, args.Arg2)
}

func TestParse_EnvValues(t *testing.T) {
	resetState()
	var args struct {
		A1 bool   `cli:"-a1"`
		A2 bool   `cli:"-a2" default:"true" env:"A2_BOOL, A2_BOOL_1"`
		A3 bool   `cli:"-a3" default:"true" env:"A3_BOOL, A3_BOOL_1"`
		B1 string `cli:"-b1"`
		B2 string `cli:"-b2" default:"b2default" env:"B2_STRING"`
		B3 string `cli:"-b3" default:"b3default" env:"B3_STRING, B3_STRING_1"`
		B4 string `cli:"-b4" default:"b4default" env:"B4_STRING, B4_STRING_1"`
		C1 []int  `cli:"-c1"`
		C2 []int  `cli:"-c2"`
	}

	os.Setenv("A2_BOOL_1", "false")
	os.Setenv("B2_STRING", "b2env")
	os.Setenv("B3_STRING", "b3env")

	fs, err := Parse(&args, WithArgs([]string{
		"-a3=0",
		"-b3=b3arg",
		"-c2=7", "-c2=8", "-c2=9",
	}), WithErrorHandling(flag.ContinueOnError))
	assert.Nil(t, err)

	assert.Equal(t, false, args.A1)
	assert.Equal(t, false, args.A2)
	assert.Equal(t, false, args.A3)
	assert.Equal(t, "", args.B1)
	assert.Equal(t, "b2env", args.B2)
	assert.Equal(t, "b3arg", args.B3)
	assert.Equal(t, "b4default", args.B4)
	assert.Equal(t, ([]int)(nil), args.C1)
	assert.Equal(t, []int{7, 8, 9}, args.C2)

	var buf bytes.Buffer
	fs.SetOutput(&buf)
	fs.Usage()

	got := buf.String()
	for _, x := range []string{
		" (default true)",
		` (env "A2_BOOL", "A2_BOOL_1")`,
		` (default "b2default")`,
		` (env "B2_STRING")`,
	} {
		_ = x
		assert.Contains(t, got, x)
	}
}

type SomeCommonFlags struct {
	X1 string `cli:"-x x1 description"`
	Y1 string `cli:"--y y1 description"`
	Z1 string `cli:"-z, --z-flag z1 description"`

	// private field will be ignored
	private1 string `cli:"--private"`
}

type AnotherCommonArgs struct {
	Ignored string `cli:"-i, --ignored"`
}

func TestParsing_TagSyntax(t *testing.T) {

	var args struct {

		// Modifier
		M1 string `cli:"#R, --m1, modifier 1"`    // required
		M2 string `cli:"#H, --m2     modifier 2"` // hidden
		M3 string `cli:"#D, --m3"`                // deprecated

		// comma seperated
		A int `cli:"-a, -a-flag       description can be seperated by spaces"`
		B int `cli:"-b, --b-flag      description can be seperated by spaces"`
		C int `cli:"#D, -c, --c-flag, description of 'DVALUE' flag"`

		SomeCommonFlags

		// manually ignored
		AnotherCommonArgs `cli:"-"`
	}
	_, err := Parse(&args, WithErrorHandling(flag.ContinueOnError), WithArgs([]string{}))
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "flag is required but not set: -m1")

	_, err = Parse(&args, WithErrorHandling(flag.ContinueOnError), WithArgs([]string{
		"-i", "ignoredstr",
	}))
	assert.NotNil(t, err)
	assert.Equal(t, err.Error(), "flag provided but not defined: -i")

	fs, err := Parse(&args, WithErrorHandling(flag.ContinueOnError), WithArgs([]string{
		"-m1", "m1str",
		"--m2", "m2str",
		"-a", "2",
		"-b-flag", "3",
		"--c-flag", "4",
		"-x", "xstr",
		"-y", "ystr",
		"-z", "zstr",
	}))
	assert.Nil(t, err)
	assert.Equal(t, fs.Lookup("m1").Value.String(), "m1str")
	assert.Equal(t, fs.Lookup("m2").Value.String(), "m2str")
	assert.Equal(t, fs.Lookup("m3").Value.String(), "")
	assert.Equal(t, fs.Lookup("a").Value.String(), "2")
	assert.Equal(t, fs.Lookup("b").Value.String(), "3")
	assert.Equal(t, fs.Lookup("c-flag").Value.String(), "4")
	assert.Equal(t, fs.Lookup("x").Value.String(), "xstr")
	assert.Equal(t, fs.Lookup("y").Value.String(), "ystr")
	assert.Equal(t, fs.Lookup("z").Value.String(), "zstr")
	assert.Nil(t, fs.Lookup("private"))
	assert.Nil(t, fs.Lookup("i"))
	assert.Nil(t, fs.Lookup("ignored"))
}

func Test_searchCommand(t *testing.T) {
	resetState()
	Add("cmd1", dummyCmd, "A cmd1 description")
	AddHidden("cmd2", dummyCmd, "A hidden cmd2 description")
	AddGroup("group1", "A group1 description")
	Add("group1 cmd1", dummyCmd, "A group1 cmd1 description")
	Add("group1 cmd2", dummyCmd, "A group1 cmd2 description")
	Add("group1 cmd3 sub1", dummyCmd, "A group1 cmd3 sub1 description")

	ctx, _, found := _search([]string{"cmd1"})
	assert.True(t, found)
	assert.Equal(t, "cmd1", ctx.name)
	assert.Equal(t, "cmd1", ctx.cmd.Name)

	ctx, _, found = _search([]string{"cmd2", "-h"})
	assert.True(t, found)
	assert.Equal(t, "cmd2", ctx.name)
	assert.Equal(t, "cmd2", ctx.cmd.Name)

	ctx, _, found = _search([]string{"group1"})
	assert.True(t, found)
	assert.Equal(t, "group1", ctx.name)

	ctx, _, found = _search([]string{"group1", "cmd99"})
	assert.True(t, found)
	assert.Equal(t, "group1", ctx.cmd.Name)

	ctx, invalidCmdName, found := _search([]string{"cmd9", "sub1", "sub2"})
	assert.False(t, found)
	assert.Equal(t, "cmd9 sub1 sub2", invalidCmdName)

	ctx, invalidCmdName, found = _search([]string{"group1", "cmd3", "sub2"})
	assert.False(t, found)
	assert.Nil(t, ctx.cmd)
	assert.Equal(t, "", invalidCmdName)
	assert.Equal(t, "group1 cmd3", ctx.name)
	assert.Equal(t, []string{"sub2"}, ctx.ambiguousArgs)
}

func Test_runGroupCommand(t *testing.T) {
	resetState()
	Add("cmd1", dummyCmd, "Dummy cmd1 command")
	AddGroup("group1", "Dummy group1 group")
	Add("group1 cmd1", dummyCmd, "Dummy group1 cmd1 command")

	ctx, _, found := _search([]string{"group1"})
	assert.True(t, found)
	assert.Equal(t, "group1", ctx.name)
	assert.Equal(t, "group1", ctx.cmd.Name)

	var buf bytes.Buffer
	getParsingContext().getFlagSet().SetOutput(&buf)
	ctx.cmd.f()

	got := buf.String()
	assert.NotContains(t, got, "not a valid command")
	for _, x := range []string{
		"Dummy group1 group",
		"USAGE:",
		"group1 <command> ...",
		"COMMANDS:",
		"group1 cmd1    Dummy group1 cmd1 command",
	} {
		assert.Contains(t, got, x)
	}
}

func Test_runCommandNotFound(t *testing.T) {
	resetState()
	Add("cmd1", dummyCmd, "Dummy cmd1 command")
	AddGroup("group1", "Dummy group1 group")
	Add("group1 cmd1", dummyCmd, "Dummy group1 cmd1 command")

	ctx, invalidCmdName, found := _search([]string{"group2", "cmd99"})
	assert.False(t, found)
	assert.Nil(t, ctx.cmd)
	assert.Equal(t, "group2 cmd99", invalidCmdName)

	ctx, _, found = _search([]string{"group1", "cmd99"})
	assert.True(t, found)
	assert.Equal(t, "group1", ctx.cmd.Name)
	assert.Equal(t, "group1", ctx.name)
}

func Test_printAvailableCommands(t *testing.T) {
	resetState()
	Add("config", dummyCmd, "config settings")
	Add("start", dummyCmd, "start service")
	Add("stop", dummyCmd, "stop service")
	AddHidden("setup", dummyCmd, "(internal) setup a test message")
	Add("import", dummyCmd, "import data")
	Add("purge", dummyCmd, "purge data")
	Add("version", dummyCmd, "print version information")
	AddGroup("git", "dummy git group")
	AddGroup("git remote", "git remote group")
	Add("git remote add", dummyCmd, "dummy git remote add command")
	Add("git remote rm", dummyCmd, "dummy git remote rm command")
	Add("git branch", dummyCmd, "dummy git branch command")
	Add("git branch rm", dummyCmd, "dummy git branch rm command")
	Add("git branch new", dummyCmd, "dummy git branch new command")
	Add("gh pr submit", dummyCmd, "dummy gh pr submit command")

	ctx, invalidCmdName, found := _search([]string{})
	assert.False(t, found)
	assert.Equal(t, "", invalidCmdName)
	assert.Nil(t, ctx.cmd)

	var buf bytes.Buffer
	ctx.getFlagSet().SetOutput(&buf)
	ctx.printUsage()

	got := buf.String()
	assert.NotContains(t, got, "not a valid command")
	for _, x := range []string{
		"USAGE:",
		"<command> ...",
		"COMMANDS:",
		"config     config settings",
		"gh         (Use -h to see available sub commands)",
		"git        dummy git group",
		"import     import data",
		"purge      purge data",
		"start      start service",
		"stop       stop service",
		"version    print version information",
	} {
		assert.Contains(t, got, x)
	}
	for _, x := range []string{
		"git remote",
		"git branch",
		"gh pr",
	} {
		assert.NotContains(t, got, x)
	}

	ctx, invalidCmdName, found = _search([]string{"git"})
	assert.True(t, found)
	assert.NotNil(t, ctx.cmd)
	assert.Equal(t, "", invalidCmdName)
	assert.Equal(t, "git", ctx.name)

	buf.Reset()
	ctx.getFlagSet().SetOutput(&buf)
	ctx.cmd.f()

	got = buf.String()
	assert.NotContains(t, got, "not a valid command")
	for _, x := range []string{
		"USAGE:",
		"<command> ...",
		"COMMANDS:",
		"  git branch    dummy git branch command",
		"    new         dummy git branch new command",
		"    rm          dummy git branch rm command",
		"  git remote    git remote group",
		"    add         dummy git remote add command",
		"    rm          dummy git remote rm command",
	} {
		assert.Contains(t, got, x)
	}
	assert.NotContains(t, got, "gh pr")

	ctx, invalidCmdName, found = _search([]string{"gh", "pr", "-h"})
	assert.False(t, found)
	assert.Nil(t, ctx.cmd)
	assert.Equal(t, "", invalidCmdName)
	assert.Equal(t, "gh pr", ctx.name)
	assert.Equal(t, 0, len(ctx.ambiguousArgs))

	buf.Reset()
	ctx.getFlagSet().SetOutput(&buf)
	ctx.printUsage()

	got = buf.String()
	assert.NotContains(t, got, "not a valid command")
	assert.NotContains(t, got, "git branch")
	assert.NotContains(t, got, "git remote")
	assert.Contains(t, got, "gh pr submit")
}

type flagValueImpl1 []byte

func (f flagValueImpl1) String() string {
	return "flagValueImpl1"
}

func (f flagValueImpl1) Set(s string) error {
	fmt.Printf("flagValueImpl1 setting %q\n", s)
	return nil
}

type flagValueImpl2 struct {
	Data []byte
}

func (f *flagValueImpl2) Get() interface{} {
	return f.Data
}

func (f flagValueImpl2) String() string {
	return "flagValueImpl2"
}

func (f *flagValueImpl2) Set(s string) error {
	f.Data = append(f.Data, s...)
	return nil
}

func TestParse_FlagValue(t *testing.T) {
	resetState()
	var args struct {
		A flagValueImpl1 `cli:"-a"`
		B flagValueImpl2 `cli:"-b"`
	}
	_, err := Parse(&args, WithErrorHandling(flag.ContinueOnError),
		WithArgs([]string{"-a", "1234", "-b", "abcd"}))
	assert.Nil(t, err)
	assert.Equal(t, []byte("abcd"), args.B.Data)
}

type SomeComplexType struct {
	A, B int64
}

func TestParse_UnsupportedType(t *testing.T) {
	var args1 struct {
		A *bool `cli:"-a"`
	}
	var args2 struct {
		B *string `cli:"-b"`
	}
	var args3 struct {
		C *SomeComplexType `cli:"-c"`
	}

	for _, args := range []interface{}{&args1, &args2, &args3} {
		assert.Panics(t, func() {
			Parse(args)
		})
	}
}

func TestWithName(t *testing.T) {
	resetState()
	var args struct {
		A flagValueImpl1 `cli:"-a"`
		B flagValueImpl2 `cli:"-b"`
	}
	fs, err := Parse(&args, WithErrorHandling(flag.ContinueOnError),
		WithName("my awesome command"),
		WithArgs([]string{"-a", "1234", "-b", "abcd"}))
	assert.Nil(t, err)

	var buf bytes.Buffer
	fs.SetOutput(&buf)
	fs.Usage()

	got := buf.String()
	assert.Contains(t, got, "my awesome command [flags]\n")
	assert.Contains(t, got, "FLAGS:")
	assert.Contains(t, got, "  -a value")
	assert.Contains(t, got, "  -b value")
}

func TestShowHidden(t *testing.T) {
	resetState()
	Add("cmd1", dummyCmd, "A cmd1 description")
	AddHidden("cmd2", dummyCmd, "A hidden cmd2 description")
	AddGroup("group1", "A group1 description")
	Add("group1 cmd1", dummyCmd, "A group1 cmd1 description")
	Add("group1 cmd2", dummyCmd, "A group1 cmd2 description")
	Add("group1 cmd3 sub1", dummyCmd, "A group1 cmd3 sub1 description")
	AddHidden("group1 cmd4", dummyCmd, "A group1 cmd4 hidden command")

	fs, err := Parse(&struct{}{},
		WithErrorHandling(flag.ContinueOnError),
		WithName("group1"),
		WithArgs([]string{"-mcli-show-hidden"}))
	assert.Nil(t, err)
	assert.Equal(t, "true", fs.Lookup(showHiddenFlag).Value.String())

	var buf bytes.Buffer
	fs.SetOutput(&buf)
	fs.Usage()
	got := buf.String()
	assert.Contains(t, got, "group1 cmd4 (HIDDEN)")

	var args struct {
		HiddenFlag1 string `cli:"#H, -a1"`
	}
	fs, err = Parse(&args,
		WithErrorHandling(flag.ContinueOnError),
		WithName("group1 cmd1"),
		WithArgs([]string{"-mcli-show-hidden"}))
	assert.Nil(t, err)
	assert.Equal(t, "true", fs.Lookup(showHiddenFlag).Value.String())

	buf.Reset()
	fs.SetOutput(&buf)
	fs.Usage()
	got = buf.String()
	assert.Contains(t, got, "-a1 string (HIDDEN)")
}
