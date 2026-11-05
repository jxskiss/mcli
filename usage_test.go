package mcli

import (
	"bytes"
	"flag"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestQuoteUsageName(t *testing.T) {
	app := NewApp()
	var args struct {
		Arg1 string        `cli:"-a, this is escaped quoted words \\'not name\\' and this is the 'name' and more quoted words \\'actually no need to quote this\\'"`
		Arg2 string        "cli:\"-b, using backtick to quote name `value` some more description\""
		Arg3 time.Duration "cli:\"-c, only one backtick, `some thing, using type name\""
	}
	app.AddRoot(func(ctx *Context) {
		ctx.Parse(&args)
		ctx.PrintHelp()
	})

	var buf bytes.Buffer
	app.getFlagSet().SetOutput(&buf)

	defer mockOSArgs("run")()
	app.Run()

	got := buf.String()
	assert.Contains(t, got, `-a <name>        this is escaped quoted words 'not name' and this is the name and more quoted words 'actually no need to quote this'`)
	assert.Contains(t, got, "-b <value>       using backtick to quote name value some more description")
	assert.Contains(t, got, "-c <duration>    only one backtick, `some thing, using type name")
}

func TestUsageEnvVariables(t *testing.T) {
	app := NewApp()
	var args struct {
		CNServiceSecret    string `mcli:"#E, CN service account secret blah blahblah blah blahblahblah" env:"CN_SERVICE_BEARER_SECRET"`
		I18NServiceSecret  string `mcli:"#E, I18N service account secret" env:"I18N_SERVICE_BEARER_SECRET"`
		CNPersonalSecret   string `mcli:"#E, CN personal account secret" env:"CN_PERSON_BEARER_SECRET"`
		I18NPersonalSecret string `mcli:"#E, I18N personal account secret" env:"I18N_PERSON_BEARER_SECRET"`
	}
	app.AddRoot(func(ctx *Context) {
		ctx.MustParse(&args)
		ctx.PrintHelp()
	})

	var buf bytes.Buffer
	app.getFlagSet().SetOutput(&buf)

	defer mockOSArgs("run")()
	app.Run()

	got := buf.String()
	assert.Contains(t, got, "Usage:\n  run\n\nEnvironment Variables:\n")
	assert.Contains(t, got, "  - CN_SERVICE_BEARER_SECRET <string>\n    CN service account secret blah blahblah blah blahblahblah\n")
	assert.Contains(t, got, "  - I18N_SERVICE_BEARER_SECRET <string>\n    I18N service account secret\n")
	assert.Contains(t, got, "  - CN_PERSON_BEARER_SECRET <string>\n    CN personal account secret\n")
	assert.Contains(t, got, "  - I18N_PERSON_BEARER_SECRET <string>\n    I18N personal account secret\n\n")
}

func TestUsageSharedAlignment(t *testing.T) {
	app := NewApp()
	app.SetGlobalFlags(&struct {
		Configuration string `cli:"--configuration, Global configuration"`
	}{})
	var args struct {
		Verbose bool   `cli:"-v, --verbose, Enable verbose output"`
		Long    string `cli:"--very-long-configuration-name, First line\nSecond line"`
		Target  string `cli:"target, Deployment target"`
	}
	var out bytes.Buffer
	app.getFlagSet().SetOutput(&out)
	_, err := app.parseArgs(&args, WithArgs([]string{"--help"}), WithErrorHandling(flag.ContinueOnError))
	assert.ErrorIs(t, err, flag.ErrHelp)
	lines := strings.Split(out.String(), "\n")
	column := -1
	for _, description := range []string{"Enable verbose output", "First line", "Second line", "Deployment target", "Global configuration"} {
		found := false
		for _, line := range lines {
			if pos := strings.Index(line, description); pos >= 0 {
				if column < 0 {
					column = pos
				}
				assert.Equal(t, column, pos, description)
				found = true
				break
			}
		}
		assert.True(t, found, description)
	}
	assert.Contains(t, out.String(), "      --very-long-configuration-name <string>\n")
}

func TestUsageCompactMetadata(t *testing.T) {
	app := NewApp()
	var args struct {
		Format string `cli:"-f, --format, Output format" default:"json" env:"TOOL_FORMAT"`
		Output string `cli:"--output" default:"stdout" env:"TOOL_OUTPUT"`
	}
	var out bytes.Buffer
	app.getFlagSet().SetOutput(&out)
	_, err := app.parseArgs(&args, WithArgs([]string{"--help"}), WithErrorHandling(flag.ContinueOnError),
		WithEnums(map[string][]string{"format": {"json", "yaml"}}))
	assert.ErrorIs(t, err, flag.ErrHelp)
	assert.Contains(t, out.String(), `[default: "json"] [env: TOOL_FORMAT] [valid: json, yaml]`)
	assert.Contains(t, out.String(), `[default: "stdout"] [env: TOOL_OUTPUT]`)
}
