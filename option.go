package mcli

import "flag"

type parseOptions struct {
	cmdName       *string
	args          *[]string
	errorHandling flag.ErrorHandling
}

// ParseOpt specifies options to customize the behavior of Parse.
type ParseOpt func(*parseOptions)

// WithArgs indicates Parse to parse from the given args, instead of
// parsing from the program's command line arguments.
func WithArgs(args []string) ParseOpt {
	return func(options *parseOptions) {
		options.args = &args
	}
}

// WithErrorHandling indicates Parse to use the given ErrorHandling.
// By default, Parse exits the program when an error happens.
func WithErrorHandling(h flag.ErrorHandling) ParseOpt {
	return func(options *parseOptions) {
		options.errorHandling = h
	}
}

// WithName specifies the name to use when printing usage doc.
func WithName(name string) ParseOpt {
	return func(options *parseOptions) {
		options.cmdName = &name
	}
}
