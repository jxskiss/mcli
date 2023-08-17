package mcli

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func resetDefaultApp() {
	*defaultApp = *NewApp()
	defaultApp.completionCtx.postFunc = func() {}
	for _, env := range os.Environ() {
		key := strings.SplitN(env, "=", 2)[0]
		os.Unsetenv(key)
	}
}

func dummyCmd() {
	Parse(nil)
	PrintHelp()
}

func dummyCmdWithContext(ctx *Context) {
	ctx.Parse(nil, WithErrorHandling(flag.ContinueOnError))
	ctx.PrintHelp()
}

func TestAddCommands(t *testing.T) {
	resetDefaultApp()
	Add("cmd1", dummyCmd, "A cmd1 description")
	AddHidden("cmd2", dummyCmd, "A hidden cmd2 description")
	AddGroup("group1", "A group1 description")
	Add("group1 cmd1", dummyCmd, "A group1 cmd1 description")
	Add("group1 cmd2", dummyCmd, "A group1 cmd2 description")
	Add("group1 cmd3 sub1", dummyCmd, "A group1 cmd3 sub1 description")
	AddHelp()
	AddCompletion()

	assert.Equal(t, 12, len(defaultApp.cmds))
	assert.Nil(t, defaultApp.ctx)
	assert.True(t, defaultApp.cmds.isValid("help"))
	assert.True(t, defaultApp.cmds.isValid("completion bash"))
	assert.True(t, defaultApp.cmds.isValid("completion zsh"))
	assert.True(t, defaultApp.cmds.isValid("completion powershell"))
	assert.True(t, defaultApp.cmds.isValid("completion fish"))
}

func TestParsing_WithoutCallingRun(t *testing.T) {
	resetDefaultApp()
	var args struct {
		A bool  `cli:"-a, -a-flag, description a flag"`
		B bool  `cli:"-b, description b flag" default:"true"`
		C int32 `cli:"-c-flag, description c flag"`
	}
	Parse(&args, WithArgs([]string{"-a", "-c-flag", "12345"}))

	// assert we do modify the global state
	assert.Equal(t, 0, len(defaultApp.cmds))
	assert.NotNil(t, defaultApp.ctx)

	// assert the arg values
	assert.True(t, args.A)
	assert.True(t, args.B)
	assert.Equal(t, args.C, int32(12345))
}

func TestParsing_CheckFlagSetValues(t *testing.T) {
	resetDefaultApp()
	var args struct {
		A  bool          `cli:"-a,  -a-flag, description a flag"`
		A1 bool          `cli:"-1, -a1-flag"`
		B  int32         `cli:"-b,  -b-flag, description b flag"`
		C  int64         `cli:"-c, --c-flag, description c flag"`
		D  float32       `cli:"-D, --d-flag, description d flag"`
		E  float64       `cli:"-E, --e-flag, description e flag"`
		F  string        `cli:"-f,  -f-flag, description f flag"`
		G  uint          `cli:"-g, --g-flag, description g flag"`
		H  []bool        `cli:"-H, --h-flag, description h flag"`
		I  []uint        `cli:"-i,  -i-flag, description i flag"`
		J  []string      `cli:"-j,  -j-flag, description j flag"`
		K  time.Duration `cli:"-k, --k-flag, description k flag"`

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
		"-k", "1.5s",
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
	assert.Equal(t, 1500*time.Millisecond, args.K)
	assert.Equal(t, []string{"some-args 0", "some-args 1"}, args.Args)
	assert.Equal(t, []byte("abc123"), args.ValueImpl2.Data)

	flagCount := 13 * 2
	fs.Visit(func(flag *flag.Flag) {
		flagCount--
	})
	assert.Zero(t, flagCount)
	for _, tt := range []struct {
		flag  string
		want  string
		value any
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
		{"k", "1.5s", 1500 * time.Millisecond},
		{"k-flag", "1.5s", 1500 * time.Millisecond},
		{"v", "abc123", []byte("abc123")},
		{"v-flag", "abc123", []byte("abc123")},
	} {
		got := fs.Lookup(tt.flag).Value.String()
		assert.Equalf(t, tt.want, got, "flag= %v", tt.flag)
		gotValue := fs.Lookup(tt.flag).Value.(flag.Getter).Get()
		assert.Equalf(t, tt.value, gotValue, "flag= %v", tt.flag)
	}
}

func TestParsing_PointerValues(t *testing.T) {
	var args struct {
		A  *bool          `cli:"-a,  -a-flag, description a flag"`
		A1 *bool          `cli:"-1, -a1-flag"`
		B  *int32         `cli:"-b,  -b-flag, description b flag"`
		C  *int64         `cli:"-c, --c-flag, description c flag"`
		D  *float32       `cli:"-D, --d-flag, description d flag"`
		E  *float64       `cli:"-E, --e-flag, description e flag"`
		F  *string        `cli:"-f,  -f-flag, description f flag"`
		G  *uint          `cli:"-g, --g-flag, description g flag"`
		K  *time.Duration `cli:"-k, --k-flag, description k flag"`

		ValueImpl2 *flagValueImpl2 `cli:"-v, -v-flag, description v flag"`

		Arg1 *string `cli:"arg1"`
	}

	t.Run("all empty", func(t *testing.T) {
		resetDefaultApp()
		fs, err := Parse(&args, WithArgs([]string{}))
		assert.Nil(t, err)
		for _, x := range []any{
			args.A, args.A1, args.B, args.C, args.D, args.E, args.F, args.G, args.K, args.ValueImpl2,
			args.Arg1,
		} {
			assert.True(t, reflect.ValueOf(x).IsNil())
		}
		_ = fs
	})

	t.Run("set flags", func(t *testing.T) {
		resetDefaultApp()
		fs, err := Parse(&args, WithArgs([]string{
			"-a-flag",
			"-1=false",
			"-b", "1",
			"-c-flag", "2",
			"--D", "3",
			"--e-flag", "4",
			"-f", "fstr",
			"-g-flag", "5",
			"-k", "1.5s",
			"-v", "abc",
			"-v", "123",

			"arg1 value",
		}))
		assert.Nil(t, err)

		assert.True(t, args.A != nil && *args.A)
		assert.True(t, args.A1 != nil && !*args.A1)
		assert.Equal(t, int32(1), *args.B)
		assert.Equal(t, int64(2), *args.C)
		assert.Equal(t, float32(3), *args.D)
		assert.Equal(t, float64(4), *args.E)
		assert.Equal(t, "fstr", *args.F)
		assert.Equal(t, uint(5), *args.G)
		assert.Equal(t, 1500*time.Millisecond, *args.K)
		assert.Equal(t, []byte("abc123"), args.ValueImpl2.Data)
		assert.Equal(t, "arg1 value", *args.Arg1)

		flagCount := 10 * 2
		fs.Visit(func(flag *flag.Flag) {
			flagCount--
		})
		assert.Zero(t, flagCount)
		for _, tt := range []struct {
			flag  string
			want  string
			value any
		}{
			{"a", "true", true},
			{"a-flag", "true", true},
			{"1", "false", false},
			{"a1-flag", "false", false},
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
			{"k", "1.5s", 1500 * time.Millisecond},
			{"k-flag", "1.5s", 1500 * time.Millisecond},
			{"v", "abc123", []byte("abc123")},
			{"v-flag", "abc123", []byte("abc123")},
		} {
			got := fs.Lookup(tt.flag).Value.String()
			assert.Equalf(t, tt.want, got, "flag= %v", tt.flag)
			gotValue := fs.Lookup(tt.flag).Value.(flag.Getter).Get()
			assert.Equalf(t, tt.value, gotValue, "flag= %v", tt.flag)
		}
	})

	t.Run("env and default values", func(t *testing.T) {
		var args struct {
			A1 *bool          `cli:"-a1"` // default false
			A2 *bool          `cli:"-a2"`
			B1 *int           `cli:"-b1" default:"1024"`
			C1 *string        `cli:"-c1" env:"C1_STR"`
			C2 *string        `cli:"-c2" default:"c2default" env:"C2_STR"`
			C3 *string        `cli:"-c3" default:"c3default" env:"C3_STR"`
			C4 *string        `cli:"-c4"`
			C5 *string        `cli:"-c5"`
			D1 *time.Duration `cli:"-d1" default:"1.5s"`
		}
		resetDefaultApp()
		os.Setenv("C1_STR", "c1EnvValue")
		os.Setenv("C3_STR", "c3EnvValue")
		fs, err := Parse(&args, WithArgs([]string{
			"-a2",
			"-c3", "c3arg",
			"-c4", "c4arg",
		}))
		_ = fs
		assert.Nil(t, err)
		assert.True(t, args.A1 == nil)
		assert.True(t, *args.A2)
		assert.Equal(t, 1024, *args.B1)
		assert.Equal(t, "c1EnvValue", *args.C1)
		assert.Equal(t, "c2default", *args.C2)
		assert.Equal(t, "c3arg", *args.C3)
		assert.Equal(t, "c4arg", *args.C4)
		assert.True(t, args.C5 == nil)
		assert.Equal(t, 1500*time.Millisecond, *args.D1)
	})

	t.Run("usage", func(t *testing.T) {
		var args struct {
			A1 *bool          `cli:"-a1, a1 description"` // default false
			A2 *bool          `cli:"-a2  a2 description"`
			B1 *int           `cli:"-b1,   b1 description" default:"1024"`
			C1 *string        `cli:"-c1" env:"C1_STR"`
			C2 *string        `cli:"-c2" default:"c2default" env:"C2_STR"`
			C3 *string        `cli:"-c3" default:"c3default" env:"C3_STR"`
			C4 *string        `cli:"-c4, a 'c4' value"`
			C5 *string        `cli:"-c5, c5 description"`
			D1 *time.Duration `cli:"-d1" default:"1.5s"`
		}
		resetDefaultApp()

		buf := &bytes.Buffer{}
		defaultApp.getFlagSet().SetOutput(buf)
		fs, err := Parse(&args,
			WithArgs([]string{"-h"}),
			WithErrorHandling(flag.ContinueOnError))
		_ = fs
		assert.Equal(t, flag.ErrHelp, err)
		got := buf.String()
		want := `USAGE:
  mcli.test [flags]

FLAGS:
  -a1             a1 description
  -a2             a2 description
  -b1 int         b1 description (default 1024)
  -c1 string      (env "C1_STR")
  -c2 string      (default c2default) (env "C2_STR")
  -c3 string      (default c3default) (env "C3_STR")
  -c4 c4          a c4 value
  -c5 string      c5 description
  -d1 duration    (default 1.5s)

`
		assert.Equal(t, want, got)
	})
}

func TestParsing_DefaultValues(t *testing.T) {
	resetDefaultApp()
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
}

func TestParse_EnvValues(t *testing.T) {
	resetDefaultApp()
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
		` (env "A3_BOOL", "A3_BOOL_1")`,
		` (default "b2default")`,
		` (env "B2_STRING")`,
		` (default "b3default")`,
		` (env "B3_STRING", "B3_STRING_1")`,
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

		// comma separated
		A int `cli:"-a, -a-flag       description can be separated by spaces"`
		B int `cli:"-b, --b-flag      description can be separated by spaces"`
		C int `cli:"#D, -c, --c-flag, description of 'DVALUE' flag"`

		SomeCommonFlags

		// manually ignored
		AnotherCommonArgs `cli:"-"`
	}

	resetDefaultApp()
	_, err := Parse(&args, WithErrorHandling(flag.ContinueOnError), WithArgs([]string{}))
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "flag is required but not set: -m1")

	resetDefaultApp()
	_, err = Parse(&args, WithErrorHandling(flag.ContinueOnError), WithArgs([]string{
		"-i", "ignoredstr",
	}))
	assert.NotNil(t, err)
	assert.Equal(t, err.Error(), "flag provided but not defined: -i")

	resetDefaultApp()
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

func _search(args []string) (ctx *parsingContext, invalidCmdName string, found bool) {
	invalidCmdName, found = defaultApp.searchCmd(args)
	ctx = defaultApp.getParsingContext()
	return
}

func Test_searchCommand(t *testing.T) {
	addCommands := func() {
		Add("cmd1", dummyCmd, "A cmd1 description")
		AddHidden("cmd2", dummyCmd, "A hidden cmd2 description")
		AddGroup("group1", "A group1 description")
		Add("group1 cmd1", dummyCmd, "A group1 cmd1 description")
		Add("group1 cmd2", dummyCmd, "A group1 cmd2 description")
		Add("group1 cmd3 sub1", dummyCmd, "A group1 cmd3 sub1 description")
	}

	resetDefaultApp()
	addCommands()
	ctx, _, found := _search([]string{"cmd1"})
	assert.True(t, found)
	assert.Equal(t, "cmd1", ctx.name)
	assert.Equal(t, "cmd1", ctx.cmd.Name)

	resetDefaultApp()
	addCommands()
	ctx, _, found = _search([]string{"cmd2", "-h"})
	assert.True(t, found)
	assert.Equal(t, "cmd2", ctx.name)
	assert.Equal(t, "cmd2", ctx.cmd.Name)

	resetDefaultApp()
	addCommands()
	ctx, _, found = _search([]string{"group1"})
	assert.True(t, found)
	assert.Equal(t, "group1", ctx.name)

	resetDefaultApp()
	addCommands()
	ctx, _, found = _search([]string{"group1", "cmd99"})
	assert.True(t, found)
	assert.Equal(t, "group1", ctx.cmd.Name)

	resetDefaultApp()
	addCommands()
	ctx, invalidCmdName, found := _search([]string{"cmd9", "sub1", "sub2"})
	assert.False(t, found)
	assert.Equal(t, "cmd9 sub1 sub2", invalidCmdName)

	resetDefaultApp()
	addCommands()
	ctx, invalidCmdName, found = _search([]string{"group1", "cmd3", "sub2"})
	assert.False(t, found)
	assert.Nil(t, ctx.cmd)
	assert.Equal(t, "group1 cmd3 sub2", invalidCmdName)
	assert.Equal(t, "group1 cmd3", ctx.name)
	assert.Equal(t, []string{"sub2"}, ctx.ambiguousArgs)
}

func Test_runGroupCommand(t *testing.T) {
	resetDefaultApp()
	Add("cmd1", dummyCmd, "Dummy cmd1 command")
	AddGroup("group1", "Dummy group1 group")
	Add("group1 cmd1", dummyCmd, "Dummy group1 cmd1 command")

	ctx, _, found := _search([]string{"group1"})
	assert.True(t, found)
	assert.Equal(t, "group1", ctx.name)
	assert.Equal(t, "group1", ctx.cmd.Name)

	var buf bytes.Buffer
	defaultApp.getParsingContext().getFlagSet().SetOutput(&buf)
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
	resetDefaultApp()
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
	resetDefaultApp()
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
	defaultApp.getFlagSet().SetOutput(&buf)
	defaultApp.printUsage()

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
	defaultApp.getFlagSet().SetOutput(&buf)
	defaultApp.printUsage()

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

func (f *flagValueImpl2) Get() any {
	return f.Data
}

func (f *flagValueImpl2) String() string {
	if f == nil {
		return ""
	}
	return string(f.Data)
}

func (f *flagValueImpl2) Set(s string) error {
	f.Data = append(f.Data, s...)
	return nil
}

func TestParse_FlagValue(t *testing.T) {
	resetDefaultApp()
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
	resetDefaultApp()
	var args1 struct {
		C *SomeComplexType `cli:"-c"`
	}

	for _, args := range []any{&args1} {
		assert.Panics(t, func() {
			Parse(args)
		})
	}
}

func TestShowHidden(t *testing.T) {
	var addCommands = func() {
		Add("cmd1", dummyCmd, "A cmd1 description")
		AddHidden("cmd2", dummyCmd, "A hidden cmd2 description")
		AddGroup("group1", "A group1 description")
		Add("group1 cmd1", dummyCmd, "A group1 cmd1 description")
		Add("group1 cmd2", dummyCmd, "A group1 cmd2 description")
		Add("group1 cmd3 sub1", dummyCmd, "A group1 cmd3 sub1 description")
		AddHidden("group1 cmd4", dummyCmd, "A group1 cmd4 hidden command")
	}

	resetDefaultApp()
	addCommands()
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

	resetDefaultApp()
	addCommands()
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

func TestReorderFlags(t *testing.T) {

	type args struct {
		Name string `cli:"-n, --name, Who do you want to say to" default:"tom"`
		Text string `cli:"#R, text, The 'message' you want to send"`
	}

	resetDefaultApp()
	args1 := &args{}
	fs, err := Parse(args1,
		WithErrorHandling(flag.ContinueOnError),
		WithArgs([]string{"hello", "-n", "Daniel"}))
	assert.Nil(t, err)
	assert.Equal(t, "Daniel", args1.Name)
	assert.Equal(t, "hello", args1.Text)
	assert.Equal(t, []string{"hello"}, fs.Args())

	resetDefaultApp()
	args2 := &args{}
	fs, err = Parse(args2,
		WithErrorHandling(flag.ContinueOnError),
		WithArgs([]string{"-n", "Daniel", "hello"}))
	assert.Nil(t, err)
	assert.Equal(t, "Daniel", args1.Name)
	assert.Equal(t, "hello", args1.Text)
	assert.Equal(t, []string{"hello"}, fs.Args())
}

func TestApp_AllowPosixSTMO(t *testing.T) {
	var args1 struct {
		ABool bool   `cli:"-a, --abool, axxx"`
		BBool bool   `cli:"-b, --bbool, bxxx"`
		CBool bool   `cli:"-c, --cbool, cxxx"`
		DStr  string `cli:"-d, --dstr, dxxx"`
		EBool bool   `cli:"-e, -ebool, exxx"`
	}

	resetDefaultApp()
	defaultApp.AllowPosixSTMO = true
	fs, err := Parse(&args1, WithArgs([]string{"-abce"}))
	assert.Nil(t, err)
	assert.Equal(t, "true", fs.Lookup("ebool").Value.String())
	assert.True(t, args1.ABool)
	assert.True(t, args1.BBool)
	assert.True(t, args1.CBool)
	assert.True(t, args1.EBool)
}

func TestApp_AliasCommand(t *testing.T) {
	resetDefaultApp()
	Add("cmd1", dummyCmd, "dummy cmd1")
	Add("cmd2", dummyCmd, "dummy cmd2")
	AddAlias("cmd3", "cmd1")

	var buf bytes.Buffer
	defaultApp.getFlagSet().SetOutput(&buf)
	PrintHelp()

	got := buf.String()
	assert.Contains(t, got, "COMMANDS")
	assert.Contains(t, got, "cmd1    dummy cmd1")
	assert.Contains(t, got, "cmd2    dummy cmd2")
	assert.Contains(t, got, `cmd3    Alias of command "cmd1"`)
}

func TestApp_FunctionWithContext(t *testing.T) {
	resetDefaultApp()
	Add("cmd1", dummyCmdWithContext, "dummy cmd1")
	Add("cmd2", dummyCmdWithContext, "dummy cmd2")
	AddAlias("cmd3", "cmd1")

	var buf bytes.Buffer
	defaultApp.getFlagSet().SetOutput(&buf)
	PrintHelp()

	got := buf.String()
	assert.Contains(t, got, "COMMANDS")
	assert.Contains(t, got, "cmd1    dummy cmd1")
	assert.Contains(t, got, "cmd2    dummy cmd2")
	assert.Contains(t, got, `cmd3    Alias of command "cmd1"`)

	buf.Reset()
	Run("cmd3")

	assert.Equal(t, "cmd3", defaultApp.getParsingContext().name)

	got = buf.String()
	assert.Contains(t, got, `Alias of command "cmd1"`)
	assert.Contains(t, got, "dummy cmd1")
}

func TestApp_printSuggestion(t *testing.T) {
	resetDefaultApp()
	defaultApp.getFlagSet().Init("", flag.ContinueOnError)

	Add("group-one cmd-one", dummyCmdWithContext, "group one command one")
	Add("group-one cmd-two", dummyCmdWithContext, "group one command two")
	Add("group-two cmd-three", dummyCmdWithContext, "group two command three")
	Add("group-two cmd-four", dummyCmdWithContext, "group two command four")

	var buf bytes.Buffer
	defaultApp.getFlagSet().SetOutput(&buf)
	defaultApp.runWithArgs([]string{"group-one", "cmd-ona"}, false)
	got := buf.String()
	assert.Contains(t, got, "Did you mean this?\n")
	assert.Contains(t, got, "group-one cmd-one")

	buf.Reset()
	defaultApp.runWithArgs([]string{"group-two", "cmd-tree"}, false)
	got = buf.String()
	assert.Contains(t, got, "Did you mean this?\n")
	assert.Contains(t, got, "group-two cmd-three")
}

func TestAppDescription(t *testing.T) {

	newTestApp := func() *App {
		app := NewApp()
		app.Description = `Test app description.

Line 3 in Description.`
		app.Add("group-one cmd-one", dummyCmdWithContext, "group one command one")
		app.getFlagSet().Init("", flag.ContinueOnError)
		return app
	}

	var buf bytes.Buffer

	// Test without root command.
	app1 := newTestApp()
	app1.getFlagSet().SetOutput(&buf)
	app1.runWithArgs([]string{}, false)
	got1 := buf.String()
	assert.Contains(t, got1, app1.Description)

	buf.Reset()
	app2 := newTestApp()
	app2.getFlagSet().SetOutput(&buf)
	app2.runWithArgs([]string{"group-one", "cmd-one"}, false)
	got2 := buf.String()
	assert.NotContains(t, got2, app2.Description)

	// Test root command.
	buf.Reset()
	app3 := newTestApp()
	app3.AddRoot(dummyCmdWithContext)
	app3.getFlagSet().SetOutput(&buf)
	app3.runWithArgs([]string{}, false)
	got3 := buf.String()
	assert.Contains(t, got3, app3.Description)

	buf.Reset()
	app4 := newTestApp()
	app4.AddRoot(dummyCmdWithContext)
	app4.getFlagSet().SetOutput(&buf)
	app4.runWithArgs([]string{"group-one", "cmd-one"}, false)
	got4 := buf.String()
	assert.NotContains(t, got4, app4.Description)
}

func TestAppOptions(t *testing.T) {
	t.Run("HelpFooter", func(t *testing.T) {
		app := NewApp()
		app.HelpFooter = `
LEARN MORE:
  Use 'program help <command> <subcommand>' for more information of a command.
`
		cmd2 := func(ctx *Context) {
			ctx.Parse(nil, WithErrorHandling(flag.ContinueOnError),
				WithFooter(func() string {
					return "Footer from parsing option."
				}))
			ctx.PrintHelp()
		}
		app.Add("cmd1", dummyCmdWithContext, "test cmd1")
		app.Add("cmd2", cmd2, "test cmd2")

		var buf bytes.Buffer
		app.resetParsingContext()
		app.getFlagSet().SetOutput(&buf)
		app.Run("cmd1", "-h")
		app.printUsage()
		got1 := buf.String()
		assert.Contains(t, got1, "LEARN MORE:\n  Use 'program help <command> <subcommand>' for more information of a command.\n\n")

		buf.Reset()
		app.resetParsingContext()
		app.getFlagSet().SetOutput(&buf)
		app.Run("cmd2", "-h")
		app.printUsage()
		got2 := buf.String()
		assert.NotContains(t, got2, "LEARN MORE")
		assert.Contains(t, got2, "Footer from parsing option.\n\n")
	})
}

func TestCoverage(t *testing.T) {
	t.Run("setupCompletionCtx", func(t *testing.T) {
		app := NewApp()
		app.Add("cmd1", dummyCmdWithContext, "cmd1")
		app.AddHidden("cmd2-hidden", dummyCmdWithContext, "cmd2 hidden")
		app.setupCompletionCtx([]string{}, "")
	})
}
