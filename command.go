package mcli

import (
	"reflect"
	"sort"
	"strings"
)

// Command holds the information of a command.
type Command struct {
	Name        string
	AliasOf     string
	Description string
	Hidden      bool

	app *App
	f   func()

	cmdOpts   []CmdOpt
	parseOpts []ParseOpt

	idx   int
	level int

	isRoot       bool
	isGroup      bool
	isCompletion bool
}

// NewCommand accepts a typed function and returns a Command.
// The type parameter T must be a struct, else it panics.
// When the command is matched, mcli will parse "args" and pass it to f,
// thus user must not call "Parse" again in f, else it panics.
// If option `WithErrorHandling(flag.ContinueOnError)` is used,
// user can use Context.ArgsError to check error that occurred during parsing
// flags and arguments.
// In case you want to get the parsed flag.FlagSet, check Context.FlagSet.
func NewCommand[T any](f func(ctx *Context, args *T), opts ...ParseOpt) *Command {
	if reflect.TypeOf(*(new(T))).Kind() != reflect.Struct {
		panic("mcli: NewCommand args type T must be a struct")
	}
	cmd := &Command{
		cmdOpts:   []CmdOpt{EnableFlagCompletion()},
		parseOpts: opts,
	}
	cmd.f = func() {
		ctx := newContext(cmd.app)
		args := new(T)
		cmd.app.parseArgs(args, cmd.parseOpts...)
		f(ctx, args)
	}
	return cmd
}

func newUntypedCommand(f func(), opts ...CmdOpt) *Command {
	return &Command{
		f:       f,
		cmdOpts: opts,
	}
}

func newUntypedCtxCommand(f func(*Context), opts ...CmdOpt) *Command {
	cmd := &Command{
		cmdOpts: opts,
	}
	cmd.f = func() {
		ctx := newContext(cmd.app)
		f(ctx)
	}
	return cmd
}

func normalizeCmdName(name string) string {
	name = strings.TrimSpace(name)
	return strings.Join(strings.Fields(name), " ")
}

func getGroupName(name string) string {
	if name == "" {
		return ""
	}
	fields := strings.Fields(name)
	return strings.Join(fields[:len(fields)-1], " ")
}

func isSubCommand(parent, sub string) bool {
	return parent != sub && strings.HasPrefix(sub, parent+" ")
}

type commands []*Command

func (p *commands) add(cmd *Command) {
	cmd.idx = len(*p) + 1
	cmd.level = len(strings.Fields(cmd.Name))
	*p = append(*p, cmd)
}

func (p commands) isValid(cmd string) bool {
	for _, c := range p {
		if c.Name == cmd || isSubCommand(cmd, c.Name) {
			return true
		}
	}
	return false
}

func (p commands) sort() {
	sort.Slice(p, func(i, j int) bool {
		return p[i].Name < p[j].Name
	})
}

func (p commands) search(ctx *parsingContext, cmdArgs []string) (hasSub bool) {
	flagIdx := findFlagIndex(cmdArgs)
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
		} else {
			hasSub = false
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
	preCmd := &Command{}
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

type categoryCommands struct {
	category string
	commands commands
}

func (p commands) groupByCategory() ([]*categoryCommands, bool) {
	var result []*categoryCommands
	hasCategories := false
	for _, c := range p {
		opts := newCmdOptions(c.cmdOpts...)
		if opts.category != "" {
			hasCategories = true
			break
		}
	}
	if !hasCategories {
		return nil, false
	}

	categoryIdxMap := make(map[string]int)
	var noCategoryCmds []*Command
	for _, c := range p {
		if c.level > 1 {
			continue
		}
		opts := newCmdOptions(c.cmdOpts...)
		if opts.category == "" {
			noCategoryCmds = append(noCategoryCmds, c)
			continue
		}
		if idx, ok := categoryIdxMap[opts.category]; !ok {
			categoryIdxMap[opts.category] = len(result)
			result = append(result, &categoryCommands{
				category: opts.category,
				commands: commands{c},
			})
		} else {
			result[idx].commands = append(result[idx].commands, c)
		}
	}
	if len(noCategoryCmds) > 0 {
		result = append(result, &categoryCommands{
			category: "Other Commands",
			commands: noCategoryCmds,
		})
	}
	return result, true
}
