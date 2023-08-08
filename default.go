package mcli

import "flag"

var defaultApp = NewApp()
var runningApp = defaultApp

func setRunningApp(app *App) func() {
	old := runningApp
	runningApp = app
	return func() { runningApp = old }
}

// SetOptions updates options of the default application.
func SetOptions(options Options) {
	defaultApp.Options = options
}

// SetGlobalFlags sets global flags, global flags are available to all commands.
// DisableGlobalFlags may be used to disable global flags for a specific
// command when calling Parse.
func SetGlobalFlags(v interface{}) {
	defaultApp.SetGlobalFlags(v)
}

// Add adds a command.
//
// Param cmd must be type of one of the following:
//   - `func()`, user should call `mcli.Parse` inside the function
//   - `func(ctx *mcli.Context)`, user should call `ctx.Parse` inside the function
//   - a Command created by NewCommand
func Add(name string, cmd interface{}, description string, opts ...CmdOpt) {
	defaultApp.Add(name, cmd, description, opts...)
}

// AddRoot adds a root command processor.
// When no sub command specified, a root command will be executed.
//
// See Add for valid types of cmd.
func AddRoot(cmd interface{}, opts ...CmdOpt) {
	defaultApp.AddRoot(cmd, opts...)
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
//
// See Add for valid types of cmd.
func AddHidden(name string, cmd interface{}, description string, opts ...CmdOpt) {
	defaultApp.AddHidden(name, cmd, description, opts...)
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
//
// Note: by default this command only enables command completion,
// to enable flag completion, user should either set
// `App.Options.EnableFlagCompletionForAllCommands` to enable flag completion
// for the whole application, or provide command option `EnableFlagCompletion`
// when adding a command to enable for a specific command.
func AddCompletion() {
	defaultApp.AddCompletion()
}

// AddCompletionWithName enables the completion command with custom command name.
//
// Note: by default this command only enables command completion,
// to enable flag completion, user should either set
// `App.Options.EnableFlagCompletionForAllCommands` to enable flag completion
// for the whole application, or provide command option `EnableFlagCompletion`
// when adding a command to enable for a specific command.
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

	// Check running App to work correctly in case of misuse of calling
	// `mcli.Parse` inside command function not with the default App.
	return runningApp.parseArgs(v, opts...)
}

// PrintHelp prints usage doc of the current command to stderr.
func PrintHelp() {

	// Check running App to work correctly in case of misuse of calling
	// `mcli.PrintHelp` inside command function not with the default App.
	runningApp.printUsage()
}
