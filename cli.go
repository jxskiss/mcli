package mcli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

const showHiddenFlag = "mcli-show-hidden"

// Options specifies optional options for an App.
type Options struct {

	// KeepCommandOrder makes Parse to print commands in the order of
	// adding the commands.
	// By default, it prints commands in lexicographic order.
	KeepCommandOrder bool

	// AllowPosixSTMO enables using the posix-style single token to specify
	// multiple boolean options. e.g. ‘-abc’ is equivalent to ‘-a -b -c’.
	AllowPosixSTMO bool
}

// NewApp creates a new cli application instance.
// Typically, there is no need to manually create an application, using
// the package-level functions with the default application is preferred.
func NewApp() *App {
	return &App{}
}

// App holds the state of a cli application.
type App struct {
	opts        Options
	cmds        commands
	groups      map[string]bool
	globalFlags interface{}

	ctx *parsingContext
	fs  *flag.FlagSet
}

// SetOptions sets optional options for App.
func (p *App) SetOptions(opts Options) {
	p.opts = opts
}

func (p *App) addCommand(cmd *Command) {
	p.cmds.add(cmd)
	if p.groups == nil {
		p.groups = make(map[string]bool)
	}
	if group := getGroupName(cmd.Name); group != "" {
		p.groups[group] = true
	}
}

func (p *App) getGlobalFlags() interface{} {
	return p.globalFlags
}

func (p *App) getParsingContext() *parsingContext {
	if p.ctx == nil {
		p.resetParsingContext()
	}
	return p.ctx
}

func (p *App) resetParsingContext() {
	p.ctx = &parsingContext{app: p, opts: newParseOptions()}
	p.fs = nil
}

func (p *App) getFlagSet() *flag.FlagSet {
	if p.fs == nil {
		p.fs = flag.NewFlagSet("", flag.ExitOnError)
		p.fs.Usage = p.printUsage
	}
	return p.fs
}

type parsingContext struct {
	app *App

	name string
	args *[]string
	opts *parseOptions

	ambiguousArgs []string
	isHelpCmd     bool
	showHidden    bool

	cmd      *Command
	flagMap  map[string]*_flag
	flags    []*_flag
	nonflags []*_flag
	parsed   bool
}

func (ctx *parsingContext) getFlagSet() *flag.FlagSet {
	return ctx.app.getFlagSet()
}

func (ctx *parsingContext) getInvalidCmdName() string {
	name := ctx.name
	if name != "" && len(ctx.ambiguousArgs) > 0 {
		name += " "
	}
	name += strings.Join(ctx.ambiguousArgs, " ")
	return name
}

func (ctx *parsingContext) parseTags(rv reflect.Value) (err error) {
	fs := ctx.getFlagSet()
	flagMap := make(map[string]*_flag)
	flags, nonflags, err := parseTags(false, fs, rv, flagMap)
	if err != nil {
		if _, ok := err.(*programingError); ok {
			panic(err)
		}
		ctx.failError(err)
		return err
	}
	ctx.flagMap = flagMap
	ctx.flags = flags
	ctx.nonflags = nonflags
	ctx.parsed = true
	return nil
}

func (ctx *parsingContext) reorderFlags(args []string) []string {
	ambiguousIdx := 0
	flagIdx := len(args)
	for i, x := range args {
		if strings.HasPrefix(x, "-") {
			flagIdx = i
			break
		}
	}
	ctx.ambiguousArgs = clip(args[ambiguousIdx:flagIdx])
	return clip(args[flagIdx:])
}

func (ctx *parsingContext) parseNonflags() (allArgs []string, err error) {
	ambiguousArgs := clip(ctx.ambiguousArgs)
	afterFlagArgs := ctx.getFlagSet().Args()

	allArgs = append(ambiguousArgs, afterFlagArgs...)
	nonflags := ctx.nonflags
	i, j := 0, 0
	for i < len(nonflags) && j < len(allArgs) {
		f := nonflags[i]
		arg := allArgs[j]
		e := f.Set(arg)
		if e != nil {
			ctx.failf(&err, "invalid value %q for %s: %v", arg, f.helpName(), e)
			return
		}
		if !(f.isSlice() || f.isMap()) {
			i++
		}
		j++
	}
	if j < len(allArgs) {
		err = &unexpectedArgsError{allArgs[j:]}
		ctx.failError(err)
		return
	}
	return allArgs, nil
}

func (ctx *parsingContext) readEnvValues() (err error) {
	fs := ctx.getFlagSet()
	flags := ctx.flags
	nonflags := ctx.nonflags
	for _, f := range flags {
		if !f.isSlice() && !f.isMap() {
			_, err = readEnv(fs, f)
			if err != nil {
				return err
			}
		}
	}
	for _, f := range nonflags {
		if !f.isSlice() && !f.isMap() {
			_, err = readEnv(fs, f)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func readEnv(fs *flag.FlagSet, f *_flag) (found bool, err error) {
	for _, name := range f.envNames {
		value := os.Getenv(name)
		if value == "" {
			continue
		}
		found = true
		if f.nonflag {
			err = f.Set(value)
		} else {
			err = fs.Set(f.name, value)
		}
		if err != nil {
			err = fmt.Errorf("invalid value %q for %s from env %s: %v", value, f.helpName(), name, err)
		}
		break
	}
	return
}

func (ctx *parsingContext) checkRequired() (err error) {
	flags := ctx.flags
	nonflags := ctx.nonflags
	for _, f := range flags {
		if f.required && f.isZero() {
			ctx.failf(&err, "flag is required but not set: -%s", f.name)
			return
		}
	}
	for _, f := range nonflags {
		if f.required && f.isZero() {
			ctx.failf(&err, "argument is required but not given: %v", f.name)
			return
		}
	}
	return
}

func (ctx *parsingContext) failf(errp *error, format string, a ...interface{}) {
	err := fmt.Errorf(format, a...)
	if *errp == nil {
		*errp = err
	}
	ctx.failError(err)
}

func (ctx *parsingContext) failError(err error) {
	fs := ctx.getFlagSet()
	out := getFlagSetOutput(fs)
	fmt.Fprintln(out, err.Error())
	if _, ok := err.(*invalidCmdError); ok {
		ctx.app.printSuggestions(ctx.getInvalidCmdName())
		fmt.Fprintln(out, "")
	} else {
		fs.Usage()
	}

	// Keep same behavior with (*flag.FlagSet).Parse.
	switch fs.ErrorHandling() {
	case flag.ExitOnError:
		os.Exit(2)
	case flag.PanicOnError:
		panic(err)
	}
}

func (p *App) printSuggestions(invalidCmdName string) {
	cmds := p.cmds
	ctx := p.getParsingContext()
	out := getFlagSetOutput(ctx.getFlagSet())
	if invalidCmdName != "" {
		sugg := cmds.suggest(invalidCmdName)
		if len(sugg) > 0 {
			fmt.Fprintf(out, "Did you mean this?\n")
			for _, cmdName := range sugg {
				fmt.Fprintf(out, "    \t%s\n", cmdName)
			}
		}
	}
}

func (p *App) printUsage() {
	globalFlags := p.getGlobalFlags()
	ctx := p.getParsingContext()

	fs := ctx.getFlagSet()
	out := getFlagSetOutput(fs)

	if ctx.opts.customUsage != nil {
		help := strings.TrimSpace(ctx.opts.customUsage())
		fmt.Fprintf(out, "%s\n\n", help)
		return
	}

	if !ctx.parsed && globalFlags != nil {
		wrapArgs := &withGlobalFlagArgs{
			GlobalFlags: globalFlags,
		}
		err := ctx.parseTags(reflect.ValueOf(wrapArgs).Elem())
		if err != nil {
			return
		}
	}

	cmds := p.cmds
	keepCmdOrder := p.opts.KeepCommandOrder

	cmd := ctx.cmd
	cmdName := ctx.name
	flags := ctx.flags
	nonflags := ctx.nonflags
	showHidden := ctx.showHidden

	flagCount := 0
	hasShortFlag := false
	for _, f := range flags {
		if !f.hidden || showHidden {
			flagCount++
			hasShortFlag = hasShortFlag || f.short != ""
		}
	}
	subCmds := cmds.listSubCommandsToPrint(cmdName, showHidden)

	progName := getProgramName()
	hasFlags, hasNonflags := flagCount > 0, len(nonflags) > 0
	hasSubCmds := len(subCmds) > 0
	usage := ""
	if cmd != nil && cmd.Description != "" {
		usage += cmd.Description + "\n\n"
	}
	usage += "USAGE:\n  " + progName
	if cmdName != "" {
		usage += " " + cmdName
	}
	if hasFlags {
		usage += " [flags]"
	}
	if hasNonflags {
		for _, f := range nonflags {
			name := f.name
			if f.isSlice() {
				name += "..."
			} else if f.isMap() {
				name += "{...}"
			}
			if f.required {
				usage += fmt.Sprintf(" <%s>", name)
			} else {
				usage += fmt.Sprintf(" [%s]", name)
			}
		}
	}
	if !hasFlags && !hasNonflags && hasSubCmds {
		usage += " <command> ..."
	}
	fmt.Fprint(out, usage, "\n\n")

	if hasSubCmds {
		printSubCommands(out, subCmds, showHidden, keepCmdOrder)
		fmt.Fprint(out, "\n")
	}

	var globalFlagHelp [][2]string
	var cmdFlagHelp [][2]string
	if flagCount > 0 {
		for _, f := range flags {
			if f.hidden && !showHidden {
				continue
			}
			name, usage := f.getUsage(hasShortFlag)
			if f.isGlobal {
				globalFlagHelp = append(globalFlagHelp, [2]string{name, usage})
			} else {
				cmdFlagHelp = append(cmdFlagHelp, [2]string{name, usage})
			}
		}
	}
	if len(cmdFlagHelp) > 0 {
		fmt.Fprint(out, "FLAGS:\n")
		printWithAlignment(out, cmdFlagHelp)
		fmt.Fprint(out, "\n")
	}
	if len(nonflags) > 0 {
		var nonflagLines [][2]string
		for _, f := range nonflags {
			name, usage := f.getUsage(false)
			nonflagLines = append(nonflagLines, [2]string{name, usage})
		}
		fmt.Fprint(out, "ARGUMENTS:\n")
		printWithAlignment(out, nonflagLines)
		fmt.Fprint(out, "\n")
	}
	if len(globalFlagHelp) > 0 {
		fmt.Fprint(out, "GLOBAL FLAGS:\n")
		printWithAlignment(out, globalFlagHelp)
		fmt.Fprint(out, "\n")
	}

	if ctx.opts.helpFooter != nil {
		footer := strings.TrimSpace(ctx.opts.helpFooter())
		fmt.Fprintf(out, "%s\n\n", footer)
	}
}

func printSubCommands(out io.Writer, cmds commands, showHidden bool, keepCmdOrder bool) {
	if len(cmds) == 0 {
		return
	}
	if keepCmdOrder {
		sort.Slice(cmds, func(i, j int) bool {
			return cmds[i].idx < cmds[j].idx
		})
	}

	var cmdLines [][2]string
	prefix := []string{""}
	preName := ""
	for _, cmd := range cmds {
		if cmd.Name == "" || (cmd.Hidden && !showHidden) {
			continue
		}
		if preName != "" && cmd.Name != preName {
			if strings.HasPrefix(cmd.Name, preName) {
				prefix = append(prefix, preName)
			} else {
				for i := len(prefix) - 1; i > 0; i-- {
					if !strings.HasPrefix(cmd.Name, prefix[i]) {
						prefix = prefix[:i]
					}
				}
			}
		}
		leafCmdName := strings.TrimSpace(strings.TrimPrefix(cmd.Name, prefix[len(prefix)-1]))
		name := strings.Repeat("  ", len(prefix)) + leafCmdName
		description := cmd.Description
		if cmd.Hidden {
			name += " (HIDDEN)"
		}
		cmdLines = append(cmdLines, [2]string{name, description})
		preName = cmd.Name
	}
	fmt.Fprint(out, "COMMANDS:\n")
	printWithAlignment(out, cmdLines)
}

func printWithAlignment(out io.Writer, lines [][2]string) {
	const _N = 36
	maxPrefixLen := 0
	for _, line := range lines {
		if n := len(line[0]); n > maxPrefixLen && n <= _N {
			maxPrefixLen = n
		}
	}
	padding := "\n" + strings.Repeat(" ", maxPrefixLen+4)
	for _, line := range lines {
		x, y := line[0], line[1]
		fmt.Fprint(out, x)
		if y != "" {
			if len(x) <= _N {
				fmt.Fprint(out, strings.Repeat(" ", maxPrefixLen+4-len(x)))
				fmt.Fprint(out, strings.ReplaceAll(y, "\n", padding))
			} else {
				fmt.Fprint(out, padding)
				fmt.Fprint(out, strings.ReplaceAll(y, "\n", padding))
			}
		}
		fmt.Fprint(out, "\n")
	}
}

// Add adds a command.
func (p *App) Add(name string, f func(), description string) {
	p.addCommand(&Command{
		Name:        name,
		Description: description,
		f:           f,
	})
}

// AddHidden adds a hidden command.
//
// A hidden command won't be showed in help, except that when a special flag
// "--mcli-show-hidden" is provided.
func (p *App) AddHidden(name string, f func(), description string) {
	p.addCommand(&Command{
		Name:        name,
		Description: description,
		Hidden:      true,
		f:           f,
	})
}

// AddGroup adds a group explicitly.
// A group is a common prefix for some commands.
// It's not required to add group before adding sub commands, but user
// can use this function to add a description to a group, which will be
// showed in help.
func (p *App) AddGroup(name string, description string) {
	p.addCommand(&Command{
		Name:        name,
		Description: description,
		f:           p.groupCmd,
	})
}

// AddHelp enables the "help" command to print help about any command.
func (p *App) AddHelp() {
	p.addCommand(&Command{
		Name:        "help",
		Description: "Help about any command",
		f:           p.helpCmd,
	})
}

func (p *App) groupCmd() {
	p.Parse(nil)
	p.PrintHelp()
}

func (p *App) helpCmd() {
	ctx := p.getParsingContext()
	ctx.isHelpCmd = true

	// i.e. "program help"
	if len(ctx.ambiguousArgs) == 0 {
		p.runWithArgs(nil)
		return
	}

	// i.e. "program help group cmd"
	cmdName := strings.Join(ctx.ambiguousArgs, " ")
	isValid := p.validateHelpCommand(cmdName)
	if !isValid {
		// failError will exit the program, we modify ctx.name here to
		// help to check suggestions.
		ctx.name = ""
		ctx.failError(newInvalidCmdError(ctx))
		return
	}

	// We got a valid command, print the help.
	p.runWithArgs(append(ctx.ambiguousArgs, "-h"))
}

func (p *App) validateHelpCommand(name string) bool {
	for _, cmd := range p.cmds {
		if cmd.Name == name {
			return true
		}
	}
	return p.groups[name]
}

// Run is the entry point to an application, it parses the command line
// and searches for a registered command, it runs the command if a command
// is found, else it will report an error and exit the program.
//
// Optionally you may specify args to parse, by default it parses the
// command line arguments os.Args[1:].
func (p *App) Run(args ...string) {
	if len(args) == 0 {
		args = os.Args[1:]
	}
	p.runWithArgs(args)
}

func (p *App) runWithArgs(args []string) {
	invalidCmdName, found := p.searchCmd(args)
	ctx := p.getParsingContext()
	if found && ctx.cmd != nil {
		ctx.cmd.f()
		return
	}
	if invalidCmdName != "" {
		err := newInvalidCmdError(ctx)
		ctx.failError(err)
		os.Exit(2)
	}
	ctx.showHidden = hasBoolFlag(showHiddenFlag, args)
	p.printUsage()
}

// searchCmd helps to do testing.
func (p *App) searchCmd(osArgs []string) (invalidCmdName string, found bool) {
	cmds := p.cmds
	cmds.sort()

	if p.getParsingContext().isHelpCmd {
		p.resetParsingContext()
	}

	ctx := p.getParsingContext()
	hasSub := cmds.search(ctx, osArgs)

	// A command is matched exactly or is parent of the requested command.
	if ctx.cmd != nil {
		return "", true
	}

	// There are sub commands available, don't report "command not found".
	if hasSub {
		return "", false
	}

	// Else the requested command must be invalid.
	return ctx.getInvalidCmdName(), false
}

// Parse parses the command line for flags and arguments.
// v should be a pointer to a struct, else it panics.
func (p *App) Parse(v interface{}, opts ...ParseOpt) (fs *flag.FlagSet, err error) {
	if v == nil {
		v = &struct{}{}
	}
	assertStructPointer(v)

	ctx := p.getParsingContext()
	ctx.opts = newParseOptions(opts...)
	options := ctx.opts

	wrapArgs := &withGlobalFlagArgs{
		GlobalFlags: nil,
		CmdArgs:     v,
	}
	if !ctx.opts.disableGlobalFlags {
		wrapArgs.GlobalFlags = p.getGlobalFlags()
	}

	fs = ctx.getFlagSet()
	fs.Init("", options.errorHandling)
	if err = ctx.parseTags(reflect.ValueOf(wrapArgs).Elem()); err != nil {
		return fs, err
	}
	if options.cmdName != nil {
		ctx.name = *options.cmdName
	}

	var cmdArgs []string
	if options.args != nil {
		cmdArgs = *options.args
	} else if ctx.args != nil {
		cmdArgs = *ctx.args
	} else {
		cmdArgs = os.Args[1:]
	}
	flagsReordered := ctx.args != nil
	if !flagsReordered {
		cmdArgs = ctx.reorderFlags(cmdArgs)
	}

	if hasBoolFlag(showHiddenFlag, cmdArgs) {
		ctx.showHidden = true
		fs.BoolVar(&ctx.showHidden, showHiddenFlag, true, "show hidden commands and flags")
	}

	// Read env values before parsing command line flags and arguments.
	if err = ctx.readEnvValues(); err != nil {
		return fs, err
	}

	// If the command does not receive arguments, but there are still
	// arguments before flags, it is absolutely an invalid command.
	if !checkNonflagsLength(ctx.nonflags, ctx.ambiguousArgs) {
		err = newInvalidCmdError(ctx)
		ctx.failError(err)
		return fs, err
	}

	// Expand the posix-style single-token-multiple-values flags.
	if p.opts.AllowPosixSTMO {
		cmdArgs = expandSMTOFlags(ctx.flagMap, cmdArgs)
	}

	if err = fs.Parse(cmdArgs); err != nil {
		return fs, err
	}
	nonflagArgs, err := ctx.parseNonflags()
	if err != nil {
		return fs, err
	}
	if err = ctx.checkRequired(); err != nil {
		return fs, err
	}
	tidyFlagSet(fs, ctx.flags, nonflagArgs)
	return fs, err
}

func assertStructPointer(v interface{}) {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Struct {
		panic("argument must be a pointer to struct")
	}
}

func checkNonflagsLength(nonflags []*_flag, args []string) (valid bool) {
	i, j := 0, 0
	for i < len(nonflags) && j < len(args) {
		f := nonflags[i]
		if !(f.isSlice() || f.isMap()) {
			i++
		}
		j++
	}
	return j == len(args)
}

func expandSMTOFlags(flagMap map[string]*_flag, args []string) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		if !strings.HasPrefix(a, "-") || strings.HasPrefix(a, "--") {
			out = append(out, a)
			continue
		}
		name := a[1:]
		if f := flagMap[name]; f != nil {
			out = append(out, a)
			continue
		}
		shouldExpand := true
		for i := 0; i < len(name); i++ {
			f := flagMap[name[i:i+1]]
			if f == nil || !f.isBoolean() {
				shouldExpand = false
				break
			}
		}
		if !shouldExpand {
			out = append(out, a)
			continue
		}
		for i := 0; i < len(name); i++ {
			out = append(out, "-"+name[i:i+1])
		}
	}
	return out
}

// PrintHelp prints usage doc of the current command to stderr.
func (p *App) PrintHelp() {
	p.printUsage()
}

// SetGlobalFlags sets global flags, global flags are available to all commands.
// DisableGlobalFlags may be used to disable global flags for a specific
// command when calling Parse.
func (p *App) SetGlobalFlags(v interface{}) {
	if v != nil {
		assertStructPointer(v)
		p.globalFlags = v
	}
}

type withGlobalFlagArgs struct {
	GlobalFlags interface{}
	CmdArgs     interface{}
}

func clip(s []string) []string {
	return s[:len(s):len(s)]
}

var isExampleTest bool

// getFlagSetOutput helps to do testing.
// When in example testing, it returns os.Stdout instead of fs.Output().
func getFlagSetOutput(fs *flag.FlagSet) io.Writer {
	if isExampleTest {
		return os.Stdout
	}
	return fs.Output()
}

func getProgramName() string {
	return filepath.Base(os.Args[0])
}

func newProgramingError(format string, args ...interface{}) *programingError {
	msg := fmt.Sprintf(format, args...)
	return &programingError{msg: msg}
}

type programingError struct {
	msg string
}

func (e *programingError) Error() string {
	return e.msg
}

func newInvalidCmdError(ctx *parsingContext) *invalidCmdError {
	return &invalidCmdError{
		groupName:      ctx.name,
		invalidCmdName: ctx.getInvalidCmdName(),
	}
}

type invalidCmdError struct {
	groupName      string
	invalidCmdName string
}

func (e *invalidCmdError) Error() string {
	cmdName := getProgramName()
	if e.groupName != "" {
		cmdName += " " + e.groupName
	}
	return fmt.Sprintf("'%s' is not a valid command. See '%s -h' for help.", e.invalidCmdName, cmdName)
}

type unexpectedArgsError struct {
	args []string
}

func (e *unexpectedArgsError) Error() string {
	return fmt.Sprintf("got unexpected %s", formatErrorArguments(e.args))
}

func formatErrorArguments(args []string) string {
	if len(args) == 1 {
		return fmt.Sprintf("argument: '%s'", args[0])
	}
	return fmt.Sprintf("arguments: '%s'", strings.Join(args, " "))
}
