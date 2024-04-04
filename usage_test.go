package mcli

import (
	"bytes"
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
