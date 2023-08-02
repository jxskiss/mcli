package mcli

import (
	"flag"
	"strings"
)

func newParseOptions(opts ...ParseOpt) *parseOptions {
	out := &parseOptions{
		errorHandling: flag.ExitOnError,
	}
	return out.apply(opts...)
}

type parseOptions struct {
	cmdName       *string
	args          *[]string
	errorHandling flag.ErrorHandling

	customUsage func() string
	helpFooter  func() string

	disableGlobalFlags bool
}

func (p *parseOptions) apply(opts ...ParseOpt) *parseOptions {
	for _, o := range opts {
		o.f(p)
	}
	return p
}

// ParseOpt specifies options to customize the behavior of Parse.
type ParseOpt struct {
	f func(*parseOptions)
}

// WithArgs tells Parse to parse from the given args, instead of
// parsing from the command line arguments.
func WithArgs(args []string) ParseOpt {
	return ParseOpt{f: func(options *parseOptions) {
		options.args = &args
	}}
}

// WithErrorHandling tells Parse to use the given ErrorHandling.
// By default, Parse exits the program when an error happens.
func WithErrorHandling(h flag.ErrorHandling) ParseOpt {
	return ParseOpt{f: func(options *parseOptions) {
		options.errorHandling = h
	}}
}

// WithName specifies the command name to use when printing usage doc.
func WithName(name string) ParseOpt {
	return ParseOpt{f: func(options *parseOptions) {
		options.cmdName = &name
	}}
}

// DisableGlobalFlags tells Parse to don't parse and print global flags in help.
func DisableGlobalFlags() ParseOpt {
	return ParseOpt{f: func(options *parseOptions) {
		options.disableGlobalFlags = true
	}}
}

// ReplaceUsage specifies a function to generate a usage help to replace the
// default help.
func ReplaceUsage(f func() string) ParseOpt {
	return ParseOpt{f: func(options *parseOptions) {
		options.customUsage = f
	}}
}

// WithFooter specifies a function to generate extra help text to print
// after the default help.
// If this option is provided, the option function's output overrides
// the App's optional help-footer setting.
func WithFooter(f func() string) ParseOpt {
	return ParseOpt{f: func(options *parseOptions) {
		options.helpFooter = f
	}}
}

func newCmdOptions(opts ...CmdOpt) cmdOptions {
	return *(new(cmdOptions).apply(opts...))
}

type cmdOptions struct {
	longDesc string
	examples string
}

func (p *cmdOptions) apply(opts ...CmdOpt) *cmdOptions {
	for _, o := range opts {
		o.f(p)
	}
	return p
}

// CmdOpt specifies options to customize the behavior of a Command.
type CmdOpt struct {
	f func(*cmdOptions)
}

// WithLongDesc specifies a long description of a command,
// which will be showed in the command's help.
func WithLongDesc(long string) CmdOpt {
	return CmdOpt{f: func(options *cmdOptions) {
		options.longDesc = strings.TrimSpace(long)
	}}
}

// WithExamples specifies examples for a command.
// Examples will be showed after flags in the command's help.
func WithExamples(examples string) CmdOpt {
	return CmdOpt{f: func(options *cmdOptions) {
		options.examples = strings.TrimSpace(examples)
	}}
}
