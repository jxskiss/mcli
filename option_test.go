package mcli

import (
	"bytes"
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithName(t *testing.T) {
	resetDefaultApp()
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
	assert.Contains(t, got, "Flags:\n")
	assert.Contains(t, got, "  -a <value>")
	assert.Contains(t, got, "  -b <value>")
}

func TestDisableGlobalFlags(t *testing.T) {
	var globalFlags struct {
		GlobalA string `cli:"-a, --global-a, dummy global flag a"`
	}

	var cmdArgs struct {
		B bool `cli:"-b, --cmd-args-b"`
	}

	app := NewApp()
	app.SetGlobalFlags(&globalFlags)

	fs, err := app.parseArgs(&cmdArgs, WithArgs([]string{"-h"}), WithErrorHandling(flag.ContinueOnError))
	assert.Error(t, err, flag.ErrHelp)
	assert.NotNil(t, fs.Lookup("b"))
	assert.NotNil(t, fs.Lookup("cmd-args-b"))
	assert.NotNil(t, fs.Lookup("a"))
	assert.NotNil(t, fs.Lookup("global-a"))

	app.resetParsingContext()
	fs, err = app.parseArgs(&cmdArgs, WithArgs([]string{"-h"}), DisableGlobalFlags(), WithErrorHandling(flag.ContinueOnError))
	assert.Error(t, err, flag.ErrHelp)
	assert.NotNil(t, fs.Lookup("b"))
	assert.NotNil(t, fs.Lookup("cmd-args-b"))
	assert.Nil(t, fs.Lookup("a"))
	assert.Nil(t, fs.Lookup("global-a"))
}

func TestReplaceUsage(t *testing.T) {
	app := NewApp()
	app.Add("dummy1", dummyCmdWithContext, "dummy cmd 1")

	var args struct {
		A string `cli:"-a, --args-a"`
		B int    `cli:"-b, --args-b"`
	}

	usage := func() string {
		return "test replace usage custom usage text\nanother line"
	}
	fs, err := app.parseArgs(&args,
		WithErrorHandling(flag.ContinueOnError),
		ReplaceUsage(usage),
		WithArgs([]string{}))
	assert.Nil(t, err)

	var buf bytes.Buffer
	fs.SetOutput(&buf)
	fs.Usage()

	got := buf.String()
	assert.NotContains(t, got, "--args-a")
	assert.NotContains(t, got, "--args-b")
	assert.Contains(t, got, "test replace usage custom usage text\nanother line")
}

func TestWithFooter(t *testing.T) {
	app := NewApp()
	app.Add("dummy1", dummyCmdWithContext, "dummy cmd 1")

	var args struct {
		A string `cli:"-a, --args-a"`
		B int    `cli:"-b, --args-b"`
	}

	footer := func() string {
		return "test with footer custom footer text\nanother line"
	}
	fs, err := app.parseArgs(&args,
		WithErrorHandling(flag.ContinueOnError),
		WithFooter(footer),
		WithArgs([]string{}))
	assert.Nil(t, err)

	var buf bytes.Buffer
	fs.SetOutput(&buf)
	fs.Usage()

	got := buf.String()
	assert.Contains(t, got, "--args-a")
	assert.Contains(t, got, "--args-b")
	assert.Contains(t, got, "test with footer custom footer text\nanother line")
}

func TestWithLongDesc(t *testing.T) {
	app := NewApp()
	app.Add("cmd1", dummyCmdWithContext, "test cmd1", WithLongDesc(`
Adding an issue to projects requires authorization with the "project" scope.
To authorize, run "gh auth refresh -s project".`))

	app.Run("cmd1", "-h")

	var buf bytes.Buffer
	fs := app.getFlagSet()
	fs.SetOutput(&buf)
	fs.Usage()

	got := buf.String()
	assert.Contains(t, got, "test cmd1\n\nAdding an issue to projects requires authorization with the \"project\" scope.\nTo authorize, run \"gh auth refresh -s project\".\n\n")
}

func TestWithExamples(t *testing.T) {

	cmdWithExamples := func(ctx *Context) {
		examples := `
$ gh issue create --title "I found a bug" --body "Nothing works"
$ gh issue create --label "bug,help wanted"
$ gh issue create --label bug --label "help wanted"
$ gh issue create --assignee monalisa,hubot
$ gh issue create --assignee "@me"
$ gh issue create --project "Roadmap"
`
		ctx.Parse(nil, WithErrorHandling(flag.ContinueOnError),
			WithExamples(examples))
		ctx.PrintHelp()
	}

	app := NewApp()
	app.Add("cmd1", cmdWithExamples, "test cmd1")
	app.Run("cmd1", "-h")

	var buf bytes.Buffer
	fs := app.getFlagSet()
	fs.SetOutput(&buf)
	fs.Usage()

	got := buf.String()
	assert.Contains(t, got, "Examples:\n  $ gh issue create --title \"I found a bug\" --body \"Nothing works\"\n  $ gh issue create --label \"bug,help wanted\"\n  $ gh")
}

func TestWithDefaults_BasicTypes(t *testing.T) {
	resetDefaultApp()
	var args struct {
		BoolFlag  bool    `cli:"-b, --bool" default:"false"`
		IntFlag   int     `cli:"-i, --int" default:"10"`
		StrFlag   string  `cli:"-s, --str" default:"default"`
		FloatFlag float64 `cli:"-f, --float" default:"1.5"`
	}

	defaults := map[string]any{
		"bool":  true,
		"int":   42,
		"str":   "custom",
		"float": 3.14,
	}

	fs, err := Parse(&args, WithErrorHandling(flag.ContinueOnError),
		WithArgs([]string{}), WithDefaults(defaults))
	assert.Nil(t, err)
	assert.Equal(t, true, args.BoolFlag)
	assert.Equal(t, 42, args.IntFlag)
	assert.Equal(t, "custom", args.StrFlag)
	assert.Equal(t, 3.14, args.FloatFlag)

	// Verify defaults appear in help
	var buf bytes.Buffer
	fs.SetOutput(&buf)
	fs.Usage()
	got := buf.String()
	assert.Contains(t, got, "[default: true]")
	assert.Contains(t, got, "[default: 42]")
	assert.Contains(t, got, `[default: "custom"]`)
	assert.Contains(t, got, "[default: 3.14]")
}

func TestWithDefaults_ShortName(t *testing.T) {
	resetDefaultApp()
	var args struct {
		A int `cli:"-a, --alpha" default:"1"`
		B int `cli:"-b, --beta" default:"2"`
	}

	// Use short names as keys
	defaults := map[string]any{
		"a": 100,
		"b": 200,
	}

	fs, err := Parse(&args, WithErrorHandling(flag.ContinueOnError),
		WithArgs([]string{}), WithDefaults(defaults))
	assert.Nil(t, err)
	assert.Equal(t, 100, args.A)
	assert.Equal(t, 200, args.B)

	var buf bytes.Buffer
	fs.SetOutput(&buf)
	fs.Usage()
	got := buf.String()
	assert.Contains(t, got, "[default: 100]")
	assert.Contains(t, got, "[default: 200]")
}

func TestWithDefaults_LongNamePriority(t *testing.T) {
	resetDefaultApp()
	var args struct {
		Value int `cli:"-v, --value" default:"1"`
	}

	// Provide both short and long names in defaults - long should take precedence
	defaults := map[string]any{
		"v":     100, // short name
		"value": 200, // long name - should win
	}

	_, err := Parse(&args, WithErrorHandling(flag.ContinueOnError),
		WithArgs([]string{}), WithDefaults(defaults))
	assert.Nil(t, err)
	assert.Equal(t, 200, args.Value) // Long name value should be used
}

func TestWithDefaults_OverrideStructTag(t *testing.T) {
	resetDefaultApp()
	var args struct {
		Flag1 string `cli:"--flag1" default:"tag_default"`
		Flag2 int    `cli:"--flag2" default:"100"`
	}

	// WithDefaults should override struct tag defaults
	defaults := map[string]any{
		"flag1": "option_default",
		"flag2": 200,
	}

	fs, err := Parse(&args, WithErrorHandling(flag.ContinueOnError),
		WithArgs([]string{}), WithDefaults(defaults))
	assert.Nil(t, err)
	assert.Equal(t, "option_default", args.Flag1)
	assert.Equal(t, 200, args.Flag2)

	var buf bytes.Buffer
	fs.SetOutput(&buf)
	fs.Usage()
	got := buf.String()
	assert.Contains(t, got, `[default: "option_default"]`)
	assert.Contains(t, got, "[default: 200]")
}

func TestWithDefaults_FallbackToStructTag(t *testing.T) {
	resetDefaultApp()
	var args struct {
		Flag1 string `cli:"--flag1" default:"tag_default"`
		Flag2 int    `cli:"--flag2" default:"100"`
		Flag3 bool   `cli:"--flag3" default:"true"`
	}

	// Only provide defaults for some flags
	defaults := map[string]any{
		"flag1": "option_default",
	}

	fs, err := Parse(&args, WithErrorHandling(flag.ContinueOnError),
		WithArgs([]string{}), WithDefaults(defaults))
	assert.Nil(t, err)
	assert.Equal(t, "option_default", args.Flag1) // From WithDefaults
	assert.Equal(t, 100, args.Flag2)              // From struct tag
	assert.Equal(t, true, args.Flag3)             // From struct tag

	var buf bytes.Buffer
	fs.SetOutput(&buf)
	fs.Usage()
	got := buf.String()
	assert.Contains(t, got, `[default: "option_default"]`)
	assert.Contains(t, got, "[default: 100]")
	assert.Contains(t, got, "[default: true]")
}

func TestWithDefaults_CmdlineOverride(t *testing.T) {
	resetDefaultApp()
	var args struct {
		Flag1 string `cli:"--flag1" default:"tag_default"`
		Flag2 int    `cli:"--flag2" default:"100"`
	}

	defaults := map[string]any{
		"flag1": "option_default",
		"flag2": 200,
	}

	// Command line args should override both WithDefaults and struct tags
	_, err := Parse(&args, WithErrorHandling(flag.ContinueOnError),
		WithArgs([]string{"--flag1", "cmdline", "--flag2", "999"}), WithDefaults(defaults))
	assert.Nil(t, err)
	assert.Equal(t, "cmdline", args.Flag1)
	assert.Equal(t, 999, args.Flag2)
}

func TestWithDefaults_PointerTypes(t *testing.T) {
	resetDefaultApp()
	var args struct {
		StrPtr *string `cli:"--str-ptr"`
		IntPtr *int    `cli:"--int-ptr"`
	}

	defaults := map[string]any{
		"str-ptr": "hello",
		"int-ptr": 42,
	}

	_, err := Parse(&args, WithErrorHandling(flag.ContinueOnError),
		WithArgs([]string{}), WithDefaults(defaults))
	assert.Nil(t, err)
	assert.NotNil(t, args.StrPtr)
	assert.Equal(t, "hello", *args.StrPtr)
	assert.NotNil(t, args.IntPtr)
	assert.Equal(t, 42, *args.IntPtr)
}

func TestWithDefaults_Arguments(t *testing.T) {
	defaults := map[string]any{
		"arg1": "from-option",
		"arg2": 999,
		"arg3": true,
	}

	// Parse with no positional args - defaults should apply
	t.Run("no positional args", func(t *testing.T) {
		resetDefaultApp()
		var args struct {
			Arg1 string `cli:"arg1" default:"default1"`
			Arg2 int    `cli:"arg2" default:"1"`
			Arg3 bool   `cli:"arg3" default:"false"`
		}
		_, err := Parse(&args, WithErrorHandling(flag.ContinueOnError),
			WithArgs([]string{}), WithDefaults(defaults))
		assert.Nil(t, err)
		assert.Equal(t, "from-option", args.Arg1)
		assert.Equal(t, 999, args.Arg2)
		assert.Equal(t, true, args.Arg3)
	})

	// Parse with some positional args - they should override defaults
	t.Run("with positional args", func(t *testing.T) {
		resetDefaultApp()
		var args2 struct {
			Arg1 string `cli:"arg1" default:"default1"`
			Arg2 int    `cli:"arg2" default:"1"`
		}
		_, err := Parse(&args2, WithErrorHandling(flag.ContinueOnError),
			WithArgs([]string{"cmdline"}), WithDefaults(defaults))
		assert.Nil(t, err)
		assert.Equal(t, "cmdline", args2.Arg1) // Overridden by command line
		assert.Equal(t, 999, args2.Arg2)       // From WithDefaults
	})
}

func TestWithDefaults_EmptyMap(t *testing.T) {
	resetDefaultApp()
	var args struct {
		Flag1 string `cli:"--flag1" default:"tag_default"`
		Flag2 int    `cli:"--flag2" default:"100"`
	}

	// Empty map should fall back to struct tags
	defaults := map[string]any{}

	_, err := Parse(&args, WithErrorHandling(flag.ContinueOnError),
		WithArgs([]string{}), WithDefaults(defaults))
	assert.Nil(t, err)
	assert.Equal(t, "tag_default", args.Flag1)
	assert.Equal(t, 100, args.Flag2)
}

func TestWithDefaults_BoolPtr(t *testing.T) {
	resetDefaultApp()
	var args struct {
		Flag *bool `cli:"--flag"`
	}

	defaults := map[string]any{
		"flag": true,
	}

	_, err := Parse(&args, WithErrorHandling(flag.ContinueOnError),
		WithArgs([]string{}), WithDefaults(defaults))
	assert.Nil(t, err)
	assert.NotNil(t, args.Flag)
	assert.Equal(t, true, *args.Flag)
}

func TestWithDefaults_NilOption(t *testing.T) {
	resetDefaultApp()
	var args struct {
		Flag1 string `cli:"--flag1" default:"tag_default"`
		Flag2 int    `cli:"--flag2" default:"100"`
	}

	// Passing nil as WithDefaults should not panic
	_, err := Parse(&args, WithErrorHandling(flag.ContinueOnError),
		WithArgs([]string{}), WithDefaults(nil))
	assert.Nil(t, err)
	assert.Equal(t, "tag_default", args.Flag1)
	assert.Equal(t, 100, args.Flag2)
}

func TestWithDefaults_NonMatchingKeys(t *testing.T) {
	resetDefaultApp()
	var args struct {
		Flag1 string `cli:"--flag1" default:"tag_default1"`
		Flag2 int    `cli:"--flag2" default:"100"`
	}

	// Keys that don't match any flag should be ignored
	defaults := map[string]any{
		"nonexistent": "ignored",
		"unknown":     999,
	}

	_, err := Parse(&args, WithErrorHandling(flag.ContinueOnError),
		WithArgs([]string{}), WithDefaults(defaults))
	assert.Nil(t, err)
	assert.Equal(t, "tag_default1", args.Flag1)
	assert.Equal(t, 100, args.Flag2)
}

func TestWithDefaults_UnsignedTypes(t *testing.T) {
	resetDefaultApp()
	var args struct {
		Uint   uint   `cli:"--uint"`
		Uint8  uint8  `cli:"--uint8"`
		Uint16 uint16 `cli:"--uint16"`
		Uint32 uint32 `cli:"--uint32"`
		Uint64 uint64 `cli:"--uint64"`
	}

	defaults := map[string]any{
		"uint":   uint(1000),
		"uint8":  uint8(255),
		"uint16": uint16(65535),
		"uint32": uint32(4294967295),
		"uint64": uint64(18446744073709551615),
	}

	_, err := Parse(&args, WithErrorHandling(flag.ContinueOnError),
		WithArgs([]string{}), WithDefaults(defaults))
	assert.Nil(t, err)
	assert.Equal(t, uint(1000), args.Uint)
	assert.Equal(t, uint8(255), args.Uint8)
	assert.Equal(t, uint16(65535), args.Uint16)
	assert.Equal(t, uint32(4294967295), args.Uint32)
	assert.Equal(t, uint64(18446744073709551615), args.Uint64)
}

func TestWithDefaults_SignedTypes(t *testing.T) {
	resetDefaultApp()
	var args struct {
		Int   int   `cli:"--int"`
		Int8  int8  `cli:"--int8"`
		Int16 int16 `cli:"--int16"`
		Int32 int32 `cli:"--int32"`
		Int64 int64 `cli:"--int64"`
	}

	defaults := map[string]any{
		"int":   int(-100),
		"int8":  int8(-128),
		"int16": int16(-32768),
		"int32": int32(-2147483648),
		"int64": int64(-9223372036854775808),
	}

	_, err := Parse(&args, WithErrorHandling(flag.ContinueOnError),
		WithArgs([]string{}), WithDefaults(defaults))
	assert.Nil(t, err)
	assert.Equal(t, int(-100), args.Int)
	assert.Equal(t, int8(-128), args.Int8)
	assert.Equal(t, int16(-32768), args.Int16)
	assert.Equal(t, int32(-2147483648), args.Int32)
	assert.Equal(t, int64(-9223372036854775808), args.Int64)
}

func TestWithEnums_BasicString(t *testing.T) {
	resetDefaultApp()
	var args struct {
		Format string `cli:"-f, --format, Output format"`
	}

	enums := map[string][]string{
		"format": {"json", "yaml", "toml"},
	}

	// Valid value
	_, err := Parse(&args, WithErrorHandling(flag.ContinueOnError),
		WithArgs([]string{"--format", "json"}), WithEnums(enums))
	assert.Nil(t, err)
	assert.Equal(t, "json", args.Format)
}

func TestWithEnums_ShortName(t *testing.T) {
	resetDefaultApp()
	var args struct {
		Level string `cli:"-l, --level, Log level"`
	}

	enums := map[string][]string{
		"l": {"debug", "info", "warn", "error"},
	}

	_, err := Parse(&args, WithErrorHandling(flag.ContinueOnError),
		WithArgs([]string{"-l", "info"}), WithEnums(enums))
	assert.Nil(t, err)
	assert.Equal(t, "info", args.Level)
}

func TestWithEnums_InvalidValue(t *testing.T) {
	resetDefaultApp()
	var args struct {
		Mode string `cli:"-m, --mode, Operation mode"`
	}

	enums := map[string][]string{
		"mode": {"fast", "slow"},
	}

	_, err := Parse(&args, WithErrorHandling(flag.ContinueOnError),
		WithArgs([]string{"--mode", "invalid"}), WithEnums(enums))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "value for flag '-mode' is invalid")
	assert.Contains(t, err.Error(), "must be one of: fast, slow")
}

func TestWithEnums_HelpDisplay(t *testing.T) {
	resetDefaultApp()
	var args struct {
		Output string `cli:"-o, --output, Output format"`
	}

	enums := map[string][]string{
		"output": {"json", "xml", "text"},
	}

	fs, err := Parse(&args, WithErrorHandling(flag.ContinueOnError),
		WithArgs([]string{}), WithEnums(enums))
	assert.Nil(t, err)

	var buf bytes.Buffer
	fs.SetOutput(&buf)
	fs.Usage()

	got := buf.String()
	assert.Contains(t, got, "[valid: json, xml, text]")
}

func TestWithEnums_LongNamePriority(t *testing.T) {
	resetDefaultApp()
	var args struct {
		Color string `cli:"-c, --color, Color output"`
	}

	// Provide both short and long names in enums - long should take precedence
	enums := map[string][]string{
		"c":     {"never"},
		"color": {"auto", "always", "never"},
	}

	fs, err := Parse(&args, WithErrorHandling(flag.ContinueOnError),
		WithArgs([]string{"--color", "auto"}), WithEnums(enums))
	assert.Nil(t, err)
	assert.Equal(t, "auto", args.Color)

	// Verify help shows correct enum values
	var buf bytes.Buffer
	fs.SetOutput(&buf)
	fs.Usage()
	got := buf.String()
	assert.Contains(t, got, "[valid: auto, always, never]")
}

func TestWithEnums_WithDefaults(t *testing.T) {
	resetDefaultApp()
	var args struct {
		Format string `cli:"-f, --format, Format" default:"json"`
	}

	enums := map[string][]string{
		"format": {"json", "yaml", "toml"},
	}

	defaults := map[string]any{
		"format": "yaml",
	}

	// Default value should also be validated
	_, err := Parse(&args, WithErrorHandling(flag.ContinueOnError),
		WithArgs([]string{}), WithDefaults(defaults), WithEnums(enums))
	assert.Nil(t, err)
	assert.Equal(t, "yaml", args.Format)
}

func TestWithEnums_InvalidDefaultValue(t *testing.T) {
	resetDefaultApp()
	var args struct {
		Format string `cli:"-f, --format, Format"`
	}

	enums := map[string][]string{
		"format": {"json", "yaml"},
	}
	defaults := map[string]any{
		"format": "invalid", // Invalid default value
	}

	var errMessage string
	func() {
		defer func() {
			if r := recover(); r != nil {
				errMessage, _ = r.(string)
			}
		}()
		_, _ = Parse(&args, WithErrorHandling(flag.ContinueOnError),
			WithArgs([]string{}), WithDefaults(defaults), WithEnums(enums))
	}()
	assert.Contains(t, errMessage, "default value \"invalid\" for flag '-format' is invalid")
	assert.Contains(t, errMessage, "must be one of: json, yaml")
}

func TestWithEnums_Arguments(t *testing.T) {
	enums := map[string][]string{
		"action": {"start", "stop", "restart"},
	}

	t.Run("valid enum value", func(t *testing.T) {
		resetDefaultApp()
		var args struct {
			Action string `cli:"action, Action to perform"`
		}

		_, err := Parse(&args, WithErrorHandling(flag.ContinueOnError),
			WithArgs([]string{"start"}), WithEnums(enums))
		assert.Nil(t, err)
		assert.Equal(t, "start", args.Action)

	})

	t.Run("invalid enum value", func(t *testing.T) {
		resetDefaultApp()
		var args struct {
			Action string `cli:"action, Action to perform"`
		}

		_, err := Parse(&args, WithErrorHandling(flag.ContinueOnError),
			WithArgs([]string{"invalid"}), WithEnums(enums))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be one of: start, stop, restart")
	})
}

func TestWithDefaults_FloatTypes(t *testing.T) {
	resetDefaultApp()
	var args struct {
		Float32 float32 `cli:"--float32"`
		Float64 float64 `cli:"--float64"`
	}

	defaults := map[string]any{
		"float32": float32(3.14),
		"float64": float64(2.71828),
	}

	_, err := Parse(&args, WithErrorHandling(flag.ContinueOnError),
		WithArgs([]string{}), WithDefaults(defaults))
	assert.Nil(t, err)
	assert.Equal(t, float32(3.14), args.Float32)
	assert.Equal(t, float64(2.71828), args.Float64)
}
