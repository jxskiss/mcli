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
	assert.Contains(t, got, "FLAGS:")
	assert.Contains(t, got, "  -a value")
	assert.Contains(t, got, "  -b value")
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
	app.Add("dummy1", dummyCmd, "dummy cmd 1")

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
	app.Add("dummy1", dummyCmd, "dummy cmd 1")

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
	app.Add("cmd1", app.dummyCmd_flagContinueOnError, "test cmd1", WithLongDesc(`
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
	app := NewApp()
	app.Add("cmd1", app.dummyCmd_flagContinueOnError, "test cmd1", WithExamples(`
$ gh issue create --title "I found a bug" --body "Nothing works"
$ gh issue create --label "bug,help wanted"
$ gh issue create --label bug --label "help wanted"
$ gh issue create --assignee monalisa,hubot
$ gh issue create --assignee "@me"
$ gh issue create --project "Roadmap"`))

	app.Run("cmd1", "-h")

	var buf bytes.Buffer
	fs := app.getFlagSet()
	fs.SetOutput(&buf)
	fs.Usage()

	got := buf.String()
	assert.Contains(t, got, "EXAMPLES:\n  $ gh issue create --title \"I found a bug\" --body \"Nothing works\"\n  $ gh issue create --label \"bug,help wanted\"\n  $ gh")
}
