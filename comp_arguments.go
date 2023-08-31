package mcli

import (
	"context"
)

/*
Arg completion tag syntax (WIP):

	`comp:"nofile"`
	`comp:"enums=a,b,c,d;"`
	`comp:"fn=funcName"`
	`comp:"listExt=json,yaml,toml"`
	`comp:"listFiles"`  // and listDirs, listDirsAndFiles, etc.

*/

// ShellCompDirective is a bit map representing the different behaviors the shell
// can be instructed to have once completions have been provided.
type ShellCompDirective int

const (
	// ShellCompDirectiveError indicates an error occurred and completions should be ignored.
	ShellCompDirectiveError ShellCompDirective = 1 << iota

	// ShellCompDirectiveNoSpace indicates that the shell should not add a space
	// after the completion even if there is a single completion provided.
	ShellCompDirectiveNoSpace

	// ShellCompDirectiveNoFileComp indicates that the shell should not provide
	// file completion even when no completion is provided.
	ShellCompDirectiveNoFileComp

	// ShellCompDirectiveFilterFileExt indicates that the provided completions
	// should be used as file extension filters.
	// For flags, using Command.MarkFlagFilename() and Command.MarkPersistentFlagFilename()
	// is a shortcut to using this directive explicitly.  The BashCompFilenameExt
	// annotation can also be used to obtain the same behavior for flags.
	ShellCompDirectiveFilterFileExt

	// ShellCompDirectiveFilterDirs indicates that only directory names should
	// be provided in file completion.  To request directory names within another
	// directory, the returned completions should specify the directory within
	// which to search.  The BashCompSubdirsInDir annotation can be used to
	// obtain the same behavior but only for flags.
	ShellCompDirectiveFilterDirs

	// ShellCompDirectiveKeepOrder indicates that the shell should preserve the order
	// in which the completions are provided
	ShellCompDirectiveKeepOrder

	// ===========================================================================

	// All directives using iota should be above this one.
	// For internal use.
	shellCompDirectiveMaxValue

	// ShellCompDirectiveDefault indicates to let the shell perform its default
	// behavior after completions have been provided.
	// This one must be last to avoid messing up the iota count.
	ShellCompDirectiveDefault ShellCompDirective = 0
)

// ArgCompletionFunc is a function to do completion for flag value or positional argument.
type ArgCompletionFunc func(ctx ArgCompletionContext) ([][]string, ShellCompDirective)

// ArgCompletionContext provides essential information to do suggestion
// for flag value and positional argument completion.
type ArgCompletionContext interface {
	context.Context

	Args() []string
}

type compContextImpl struct {
	*Context
	app  *App
	args []string
}

func (c compContextImpl) Args() []string {
	return c.args
}

// func (c compContextImpl) App() []string {
// 	return c.app
// }

// WithArgCompFuncs for struct holding completion functions for use with flags args completion
func WithArgCompFuncs(funcMap map[string]ArgCompletionFunc) ParseOpt {
	return ParseOpt{
		f: func(options *parseOptions) {
			options.argCompFuncs = funcMap
		},
	}
}

// WithCommandCompFuncs for holding completion functions for use with args completions
func WithCommandCompFunc(function ArgCompletionFunc) CmdOpt {
	return CmdOpt{f: func(options *cmdOptions) {
		options.argCompFunc = function
	}}
}

func compByFunc(funcName string) ArgCompletionFunc {
	return func(ctx ArgCompletionContext) ([][]string, ShellCompDirective) {
		panic("not implemented")
	}
}

func compNofile(ctx ArgCompletionContext) ([][]string, ShellCompDirective) {
	return nil, ShellCompDirectiveNoFileComp
}

// func compEnums(values []string) ArgCompletionFunc {
// 	return func(ctx ArgCompletionContext) ([][]string, ShellCompDirective) {
// 		return values, ShellCompDirectiveDefault
// 	}
// }

func compListFilesExt(extList [][]string) ArgCompletionFunc {
	panic("not implemented")
}
