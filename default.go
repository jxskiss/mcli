package mcli

import "flag"

func init() {
	defaultApp = NewApp()
}

var defaultApp *App

// SetGlobalFlags sets global flags, global flags are available to all commands.
// DisableGlobalFlags may be used to disable global flags for a specific
// command when calling Parse.
func SetGlobalFlags(v interface{}) {
	defaultApp.SetGlobalFlags(v)
}

// Add adds a command.
// f must be a function of signature `func()` or `func(*Context)`, else it panics.
func Add(name string, f interface{}, description string, opts ...CmdOpt) {
	defaultApp.Add(name, f, description, opts...)
}

// AddRoot adds a root command processor.
// When no sub command specified, a root command will be executed.
func AddRoot(f interface{}, opts ...CmdOpt) {
	defaultApp.AddRoot(f, opts...)
}

// AddAlias adds an alias name for a command.
func AddAlias(aliasName, target string, opts ...CmdOpt) {
	defaultApp.AddAlias(aliasName, target, opts...)
}

// AddHidden adds a hidden command.
// f must be a function of signature `func()` or `func(*Context)`, else it panics.
//
// A hidden command won't be showed in help, except that when a special flag
// "--mcli-show-hidden" is provided.
func AddHidden(name string, f interface{}, description string, opts ...CmdOpt) {
	defaultApp.AddHidden(name, f, description, opts...)
}

// AddGroup adds a group explicitly.
// A group is a common prefix for some commands.
// It's not required to add group before adding sub commands, but user
// can use this function to add a description to a group, which will be
// showed in help.
func AddGroup(name string, description string, opts ...CmdOpt) {
	defaultApp.AddGroup(name, description, opts...)
}

// AddHelp enables the "help" command to print help about any command.
func AddHelp() {
	defaultApp.AddHelp()
}

// AddCompletion enables the "completion" command to generate auto-completion script.
// If you want a different name other than "completion", use AddCompletionWithName.
func AddCompletion() {
	defaultApp.AddCompletion()
}

// AddCompletionWithName enables the completion command with custom command name.
func AddCompletionWithName(name string) {
	defaultApp.AddCompletionWithName(name)
}

// Run runs the program, it parses the command line and searches for a
// registered command, it runs the command if a command is found,
// else it will report an error and exit the program.
//
// Optionally you may specify args to parse, by default it parses the
// command line arguments os.Args[1:].
func Run(args ...string) {
	defaultApp.Run(args...)
}

// Parse parses the command line for flags and arguments.
// v must be a pointer to a struct, else it panics.
func Parse(v interface{}, opts ...ParseOpt) (fs *flag.FlagSet, err error) {
	return defaultApp.parseArgs(v, opts...)
}

// PrintHelp prints usage doc of the current command to stderr.
func PrintHelp() {
	defaultApp.printUsage()
}
