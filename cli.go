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

// Command holds the information of a command.
type Command struct {
	Name        string
	Description string
	Hidden      bool

	f func()

	level int
}

func normalizeCmdName(name string) string {
	name = strings.TrimSpace(name)
	return strings.Join(strings.Fields(name), " ")
}

func isSubCommand(parent, sub string) bool {
	return parent != sub && strings.HasPrefix(sub, parent+" ")
}

type commands []*Command

func (p *commands) add(cmd *Command) {
	cmd.Name = normalizeCmdName(cmd.Name)
	if cmd.Name == "" {
		panic("command name must not be empty")
	}
	for _, x := range *p {
		if x.Name == cmd.Name {
			panic("command name must be unique")
		}
	}
	cmd.level = len(strings.Fields(cmd.Name))
	*p = append(*p, cmd)
}

func (p commands) sort() {
	sort.Slice(p, func(i, j int) bool {
		return p[i].Name < p[j].Name
	})
}

func (p commands) search(cmdArgs []string) (ctx *parsingContext, hasSub bool) {
	ctx = &parsingContext{}
	flagIdx := len(cmdArgs)
	for i, x := range cmdArgs {
		if strings.HasPrefix(x, "-") {
			flagIdx = i
			break
		}
	}
	tryName := ""
	ambiguousIdx := -1
	args := cmdArgs[:]
	var cmd *Command
	for i := 0; i < len(cmdArgs); i++ {
		arg := cmdArgs[i]
		if strings.HasPrefix(arg, "-") {
			break
		}
		args = cmdArgs[i+1:]
		if tryName != "" {
			tryName += " "
		}
		tryName += arg
		idx := sort.Search(len(p), func(i int) bool {
			return p[i].Name >= tryName
		})
		if idx < len(p) &&
			(p[idx].Name == tryName || isSubCommand(tryName, p[idx].Name)) {
			hasSub = true
			ambiguousIdx = i + 1
			ctx.name = tryName
			if p[idx].Name == tryName {
				cmd = p[idx]
			} else {
				cmd = nil
			}
			continue
		}
		if ambiguousIdx == -1 {
			ambiguousIdx = i
		}
		args = cmdArgs[flagIdx:]
		break
	}
	ctx.cmd = cmd
	ctx.args = &args
	if ambiguousIdx >= 0 {
		ctx.ambiguousArgs = clip(cmdArgs[ambiguousIdx:flagIdx])
	}
	return
}

func (p commands) listSubCommandsToPrint(name string, showHidden bool) (sub commands) {
	sub = p._listSubCommandsToPrint(name, showHidden, false)
	if len(sub) > 10 {
		sub = p._listSubCommandsToPrint(name, showHidden, true)
	}
	return sub
}

func (p commands) _listSubCommandsToPrint(name string, showHidden, onlyNextLevel bool) (sub commands) {
	name = normalizeCmdName(name)
	wantLevel := len(strings.Fields(name)) + 1
	var preCmd = &Command{}
	for _, cmd := range p {
		if cmd.Name != name && strings.HasPrefix(cmd.Name, name) {
			// Don't print hidden commands.
			if cmd.Hidden && !showHidden {
				continue
			}
			if onlyNextLevel {
				if cmd.level < wantLevel {
					continue
				}
				if cmd.level > wantLevel {
					_names := strings.Fields(cmd.Name)
					parentCmdName := strings.Join(_names[:wantLevel], " ")
					if parentCmdName == preCmd.Name {
						continue
					}
					cmd = &Command{
						Name:        parentCmdName,
						Description: "(Use -h to see available sub commands)",
					}
				}
				// else cmd.Level == wantLevel
			}
			sub = append(sub, cmd)
			preCmd = cmd
		}
	}
	return
}

func (p commands) suggest(name string) []string {
	type withDistance struct {
		name     string
		distance int
	}
	const minDistance = 2
	var levenshteinSuggestions []string
	var prefixSuggestions []withDistance
	for _, cmd := range p {
		if !cmd.Hidden {
			levenshteinDistance := ld(name, cmd.Name, true)
			isPrefix := strings.HasPrefix(strings.ToLower(cmd.Name), strings.ToLower(name))
			if levenshteinDistance <= minDistance {
				levenshteinSuggestions = append(levenshteinSuggestions, cmd.Name)
			} else if isPrefix {
				prefixSuggestions = append(prefixSuggestions, withDistance{cmd.Name, levenshteinDistance})
			}
		}
	}
	sort.SliceStable(prefixSuggestions, func(i, j int) bool {
		return prefixSuggestions[i].distance < prefixSuggestions[j].distance
	})
	suggestions := levenshteinSuggestions
	for _, x := range prefixSuggestions {
		suggestions = append(suggestions, x.name)
	}
	if len(suggestions) > 5 {
		suggestions = suggestions[:5]
	}
	return suggestions
}

// ld compares two strings and returns the levenshtein distance between them.
func ld(s, t string, ignoreCase bool) int {
	if ignoreCase {
		s = strings.ToLower(s)
		t = strings.ToLower(t)
	}
	d := make([][]int, len(s)+1)
	for i := range d {
		d[i] = make([]int, len(t)+1)
	}
	for i := range d {
		d[i][0] = i
	}
	for j := range d[0] {
		d[0][j] = j
	}
	for j := 1; j <= len(t); j++ {
		for i := 1; i <= len(s); i++ {
			if s[i-1] == t[j-1] {
				d[i][j] = d[i-1][j-1]
			} else {
				min := d[i-1][j]
				if d[i][j-1] < min {
					min = d[i][j-1]
				}
				if d[i-1][j-1] < min {
					min = d[i-1][j-1]
				}
				d[i][j] = min + 1
			}
		}

	}
	return d[len(s)][len(t)]
}

var state struct {
	cmds commands
	*parsingContext
}

type parsingContext struct {
	name string
	args *[]string

	ambiguousArgs []string
	showHidden    bool

	cmd      *Command
	fs       *flag.FlagSet
	flags    []*_flag
	nonflags []*_flag
}

func getParsingContext() *parsingContext {
	if state.parsingContext != nil {
		return state.parsingContext
	}
	return &parsingContext{}
}

func (ctx *parsingContext) getFlagSet() *flag.FlagSet {
	if ctx.fs == nil {
		ctx.fs = flag.NewFlagSet("", flag.ExitOnError)
		ctx.fs.Usage = ctx.printUsage
	}
	return ctx.fs
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
	flags, nonflags, err := parseTags(fs, rv)
	if err != nil {
		if _, ok := err.(*programingError); ok {
			panic(err)
		}
		ctx.failError(err)
		return err
	}
	ctx.flags = flags
	ctx.nonflags = nonflags
	return nil
}

func (ctx *parsingContext) parseNonflags() (err error) {
	ambiguousArgs := ctx.ambiguousArgs
	afterFlagArgs := ctx.getFlagSet().Args()
	nonflags := ctx.nonflags
	i, j := 0, 0
	for i < len(nonflags) && j < len(ambiguousArgs) {
		f := nonflags[i]
		if !(f.isSlice() || f.isMap()) {
			i++
		}
		j++
	}
	if j < len(ambiguousArgs) {
		err = &ambiguousArgumentsError{ambiguousArgs}
		ctx.failError(err)
		return
	}
	i, j = 0, 0
	allArgs := append(ambiguousArgs, afterFlagArgs...)
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
	return
}

func (ctx *parsingContext) readEnvForNonSliceValues() (err error) {
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

func (ctx *parsingContext) tidyFlagSet() {
	fs := ctx.getFlagSet()
	flags := ctx.flags
	m := make(map[string]*_flag)
	for _, f := range flags {
		m[f.name] = f
		if f.short != "" {
			m[f.short] = f
		}
	}

	// This is awkward, but we can not simply call flag.Value's Set
	// method, the Set operation may be not idempotent.
	// Thus, we unsafely modify FlagSet's unexported internal data,
	// this may break in a future Go release.

	actual := _flagSet_getActual(fs)
	formal := _flagSet_getFormal(fs)
	fs.Visit(func(ff *flag.Flag) {
		f := m[ff.Name]
		if f == nil {
			return
		}
		if f.name != ff.Name {
			formal[f.name].Value = ff.Value
			actual[f.name] = formal[f.name]
		}
		if f.short != "" && f.short != ff.Name {
			formal[f.short].Value = ff.Value
			actual[f.short] = formal[f.short]
		}
	})
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
	fmt.Fprintln(getFlagSetOutput(fs), err.Error())
	if _, ok := err.(*ambiguousArgumentsError); ok {
		ctx.printSuggestions(ctx.getInvalidCmdName())
	}
	fs.Usage()
	switch fs.ErrorHandling() {
	case flag.ExitOnError:
		os.Exit(2)
	case flag.PanicOnError:
		panic(err)
	}
}

func (ctx *parsingContext) printSuggestions(invalidCmdName string) {
	cmds := state.cmds
	out := getFlagSetOutput(ctx.getFlagSet())
	if invalidCmdName != "" {
		sugg := cmds.suggest(invalidCmdName)
		if len(sugg) > 0 {
			fmt.Fprintf(out, "Did you mean this?\n")
			for _, cmdName := range sugg {
				fmt.Fprintf(out, "    \t%s\n", cmdName)
			}
			fmt.Fprint(out, "\n")
		}
	}
}

func (ctx *parsingContext) printUsage() {
	cmds := state.cmds
	fs := ctx.getFlagSet()
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

	out := getFlagSetOutput(fs)
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

	if flagCount > 0 {
		var flagLines [][2]string
		for _, f := range flags {
			if f.hidden && !showHidden {
				continue
			}
			name, usage := f.getUsage(hasShortFlag)
			flagLines = append(flagLines, [2]string{name, usage})
		}
		fmt.Fprint(out, "FLAGS:\n")
		printWithAlignment(out, flagLines)
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

	if hasSubCmds {
		printAvailableCommands(out, cmdName, cmds, showHidden)
		fmt.Fprint(out, "\n")
	}
}

func printAvailableCommands(out io.Writer, name string, cmds commands, showHidden bool) {
	if sub := cmds.listSubCommandsToPrint(name, showHidden); len(sub) > 0 {
		cmds = sub
	}
	if len(cmds) == 0 {
		return
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
	for _, line := range lines {
		x, y := line[0], line[1]
		fmt.Fprint(out, x)
		if y != "" {
			if len(x) < _N {
				fmt.Fprint(out, strings.Repeat(" ", maxPrefixLen+4-len(x)))
				fmt.Fprint(out, strings.ReplaceAll(y, "\n", "\n    \t"))
			} else {
				fmt.Fprint(out, "\n    \t")
				fmt.Fprint(out, strings.ReplaceAll(y, "\n", "\n    \t"))
			}
		}
		fmt.Fprint(out, "\n")
	}
}

// Add adds a command to global state.
func Add(name string, f func(), description string) {
	state.cmds.add(&Command{
		Name:        name,
		Description: description,
		f:           f,
	})
}

// AddHidden adds a hidden command to global state.
//
// A hidden command won't be showed in help, except that when a special flag
// "--mcli-show-hidden" is provided.
func AddHidden(name string, f func(), description string) {
	state.cmds.add(&Command{
		Name:        name,
		Description: description,
		Hidden:      true,
		f:           f,
	})
}

// AddGroup adds a group to global state.
// A group is a common prefix for some commands.
func AddGroup(name string, description string) {
	state.cmds.add(&Command{
		Name:        name,
		Description: description,
		f:           groupCmd,
	})
}

// AddHelp enables the "help" command to print help about any command.
func AddHelp() {
	state.cmds.add(&Command{
		Name:        "help",
		Description: "Help about any command",
		f:           helpCmd,
	})
}

func groupCmd() {
	Parse(nil)
	PrintHelp()
}

func helpCmd() {
	ctx := getParsingContext()
	if len(ctx.ambiguousArgs) == 0 {
		runWithArgs(nil)
		return
	}
	runWithArgs(append(ctx.ambiguousArgs, "-h"))
}

// Run runs the program, it will parse the command line and search
// for a registered command, it runs the command if a command is found,
// else it will report an error and exit the program.
func Run() {
	runWithArgs(os.Args[1:])
}

func runWithArgs(osArgs []string) {
	ctx, invalidCmdName, found := _search(osArgs)
	if found && ctx.cmd != nil {
		ctx.cmd.f()
		return
	}
	if invalidCmdName != "" {
		out := getFlagSetOutput(ctx.getFlagSet())
		progName := getProgramName()
		fmt.Fprintf(out, "'%s' is not a valid command. See '%s -h' for help.\n", invalidCmdName, progName)
		ctx.printSuggestions(invalidCmdName)
	}
	ctx.showHidden = hasBoolFlag(showHiddenFlag, os.Args[1:])
	ctx.printUsage()
}

// _search helps to do testing.
func _search(osArgs []string) (ctx *parsingContext, invalidCmdName string, found bool) {
	cmds := state.cmds
	cmds.sort()
	ctx, hasSub := cmds.search(osArgs)
	state.parsingContext = ctx

	// A command is matched exactly or is parent of the requested
	// command, just run the command.
	if ctx.cmd != nil {
		return ctx, "", true
	}

	// There are sub commands available, don't report "command not found".
	if hasSub {
		return ctx, "", false
	}

	// Else the requested command must be invalid.
	return ctx, ctx.getInvalidCmdName(), false
}

// Parse parses the command line for flags and arguments.
// v should be a pointer to a struct, else it panics.
func Parse(v interface{}, opts ...ParseOpt) (fs *flag.FlagSet, err error) {
	if v == nil {
		v = &struct{}{}
	}
	options := &parseOptions{
		errorHandling: flag.ExitOnError,
	}
	for _, o := range opts {
		o(options)
	}

	ctx := getParsingContext()
	fs = ctx.getFlagSet()
	fs.Init("", options.errorHandling)
	if err = ctx.parseTags(reflect.ValueOf(v).Elem()); err != nil {
		return fs, err
	}
	if options.cmdName != nil {
		ctx.name = *options.cmdName
	}

	osArgs := os.Args[1:]
	if ctx.args != nil {
		osArgs = *ctx.args
	}
	if options.args != nil {
		osArgs = *options.args
	}

	if hasBoolFlag(showHiddenFlag, osArgs) {
		ctx.showHidden = true
		fs.BoolVar(&ctx.showHidden, showHiddenFlag, true, "show hidden commands and flags")
	}

	// Read env for non-slice values before parsing command line
	// flags and arguments.
	if err = ctx.readEnvForNonSliceValues(); err != nil {
		return fs, err
	}

	if err = fs.Parse(osArgs); err != nil {
		return fs, err
	}
	if err = ctx.parseNonflags(); err != nil {
		return fs, err
	}
	if err = ctx.checkRequired(); err != nil {
		return fs, err
	}
	ctx.tidyFlagSet()
	return fs, err
}

// PrintHelp prints usage doc of the current command to stderr.
func PrintHelp() {
	ctx := getParsingContext()
	ctx.printUsage()
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

type programingError struct {
	msg string
}

func (e *programingError) Error() string {
	return e.msg
}

type ambiguousArgumentsError struct {
	args []string
}

func (e *ambiguousArgumentsError) Error() string {
	return fmt.Sprintf("cannot resolve ambiguous arguments: %q", e.args)
}
