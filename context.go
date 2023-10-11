package mcli

import (
	"context"
	"flag"
)

func newContext(app *App) *Context {
	return &Context{
		Context: context.Background(),
		Command: app.getParsingContext().cmd,
		app:     app,
	}
}

// Context holds context-specific information, which is passed to a
// Command when executing it.
// Context embeds context.Context, it can be passed to functions which
// take context.Context as parameter.
type Context struct {
	context.Context
	Command *Command

	app *App
}

// Parse parses the command line for flags and arguments.
// `args` must be a pointer to a struct, else it panics.
//
// By default, it prints help and exits the program if an error occurs
// when parsing, instead of returning the error,
// which is the same behavior with package "flag".
// Generally, user can safely ignore the return value of this function,
// except that an option `WithErrorHandling(flag.ContinueOnError)`
// is explicitly passed to it if you want to inspect the error.
//
// Note:
//  1. if you enable flag completion for a command, you must call this
//     in the command function to make the completion work correctly
//  2. if the command is created by NewCommand, mcli automatically calls
//     this to parse flags and arguments, then pass args to user command,
//     you must not call this again, else it panics
func (ctx *Context) Parse(args any, opts ...ParseOpt) (*flag.FlagSet, error) {
	return ctx.app.parseArgs(args, opts...)
}

// ArgsError returns the error of parsing arguments.
// If no error occurs, it returns nil.
func (ctx *Context) ArgsError() error {
	return ctx.app.getParsingContext().flagErr
}

// FlagSet returns the flag.FlagSet parsed from arguments.
// This is for compatibility to work with standard library, in most cases,
// using the strongly-typed parsing result is more convenient.
func (ctx *Context) FlagSet() *flag.FlagSet {
	return ctx.app.getFlagSet()
}

// PrintHelp prints usage doc of the current command to stderr.
func (ctx *Context) PrintHelp() {
	ctx.app.printUsage()
}
