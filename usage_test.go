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
	assert.Contains(t, got, `-a name        this is escaped quoted words 'not name' and this is the name and more quoted words 'actually no need to quote this'`)
	assert.Contains(t, got, "-b value       using backtick to quote name value some more description")
	assert.Contains(t, got, "-c duration    only one backtick, `some thing, using type name")
}
