package mcli

import (
	"context"
	"flag"
)

func newContext(app *App, cmd *Command) *Context {
	return &Context{
		Context: context.Background(),
		App:     app,
		Command: cmd,
	}
}

// Context holds context-specific information, which is passed to a
// Command when executing it.
// Context embeds context.Context, it can be passed to functions which
// take context.Context as parameter.
type Context struct {
	context.Context

	App     *App
	Command *Command
}

// Parse parses the command line for flags and arguments.
// v must be a pointer to a struct, else it panics.
func (ctx *Context) Parse(v interface{}, opts ...ParseOpt) (*flag.FlagSet, error) {
	return ctx.App.parseArgs(v, opts...)
}

// PrintHelp prints usage doc of the current command to stderr.
func (ctx *Context) PrintHelp() {
	ctx.App.printUsage()
}

func (ctx *Context) getParsingContext() *parsingContext {
	return ctx.App.getParsingContext()
}
