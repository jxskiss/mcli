package mcli

import (
	"context"
	"flag"
	"strings"
)

type CompletionItem struct {
	Value       string
	Description string
}

// ArgCompletionFunc is a function to do completion for flag value or positional argument.
type ArgCompletionFunc func(ctx ArgCompletionContext) []CompletionItem

// ArgCompletionContext provides essential information to do suggestion
// for flag value and positional argument completion.
type ArgCompletionContext interface {
	context.Context

	GlobalFlags() any
	CommandArgs() any
	FlagSet() *flag.FlagSet
	ArgPrefix() string
}

func (p *App) newArgCompletionContext() ArgCompletionContext {
	return &compContextImpl{
		Context: context.Background(),
		app:     p,
	}
}

type compContextImpl struct {
	context.Context
	app *App
}

func (c *compContextImpl) GlobalFlags() any {
	return c.app.globalFlags
}

func (c *compContextImpl) CommandArgs() any {
	return c.app.completionCtx.parsedArgs
}

func (c *compContextImpl) FlagSet() *flag.FlagSet {
	return c.app.getFlagSet()
}

func (c *compContextImpl) ArgPrefix() string {
	return c.app.completionCtx.prefixWord
}

// WithArgCompFuncs specifies completion functions to complete flag values
// or positional arguments.
// Key of funcMap should be a flag name in form "-flag" or a positional arg name "arg1".
func WithArgCompFuncs(funcMap map[string]ArgCompletionFunc) ParseOpt {
	copyMap := make(map[string]ArgCompletionFunc)
	for name, x := range funcMap {
		name = normalizeCompFlagName(name)
		copyMap[name] = x
	}
	return ParseOpt{f: func(options *parseOptions) {
		options.argCompFuncs = copyMap
	}}
}

func normalizeCompFlagName(s string) string {
	if strings.HasPrefix(s, "-") {
		s = "-" + strings.TrimLeft(s, "-")
	}
	return strings.TrimSpace(s)
}

func cleanFlagName(s string) string {
	return strings.TrimLeft(s, "-")
}
