package mcli

import (
	"embed"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"
)

const completionFlag = "--mcli-generate-completion"

type completionCtx struct {
	out      io.Writer // help in testing to inspect completion output
	postFunc func()    // help in testing to not exit the program
	shell    string
	userArgs []string
	cmd      *cmdTree

	lastArg           string
	hasFlag           bool
	wantFlagValue     bool
	flagName          string
	wantPositionalArg bool
	prefixWord        string

	cmdArgs    *[]string
	parsedArgs any
}

func getAllowedShells() []string {
	return []string{"bash", "zsh", "fish", "powershell"}
}

func hasCompletionFlag(args []string) (bool, []string, string) {
	shell := "unsupported"
	completionFlagIndex := find(args, completionFlag)
	if completionFlagIndex < 0 {
		return false, args, shell
	}
	if completionFlagIndex < len(args)-1 {
		proposedShell := args[completionFlagIndex+1]
		if contains(getAllowedShells(), proposedShell) {
			shell = proposedShell
		}
	}
	args = args[:completionFlagIndex]
	return true, args, shell
}

func (p *App) setupCompletionCtx(userArgs []string, completionShell string) {
	p.isCompletion = true
	if p.completionCtx.out == nil {
		p.completionCtx.out = os.Stdout
	}
	if p.completionCtx.postFunc == nil {
		p.completionCtx.postFunc = func() {
			os.Exit(0)
		}
	}
	p.completionCtx.shell = completionShell

	// Parse completion ctx information.
	for _, arg := range userArgs {
		if strings.HasPrefix(arg, "-") {
			p.completionCtx.hasFlag = true
			break
		}
	}
	if n := len(userArgs); n > 0 {
		p.completionCtx.lastArg = userArgs[n-1]
		userArgs = userArgs[:n-1]
	}
	p.completionCtx.userArgs = userArgs
	p.completionCtx.cmdArgs = &userArgs
}

func (p *App) shouldSuggestSubCommands(tree *cmdTree, hasFlag bool) bool {
	isGroupCmd := tree.Cmd == nil || tree.Cmd.isGroup
	isLeafCmd := len(tree.SubCmds) == 0
	return !hasFlag && isGroupCmd && !isLeafCmd
}

func (p *App) doAutoCompletion(userArgs []string) {
	ctx := p.getParsingContext()
	tree := p.parseCompletionCmdTree()

	cmdNames := userArgs
	hasFlag := false
	for i, x := range userArgs {
		if strings.HasPrefix(x, "-") {
			hasFlag = true
			cmdNames = userArgs[:i]
			break
		}
	}

	var leftArgs []string
	tree, leftArgs = tree.findCommand(cmdNames)
	if tree == nil {
		return
	}

	p.completionCtx.cmd = tree
	ctx.cmd = tree.Cmd

	if p.shouldSuggestSubCommands(tree, hasFlag) {
		// Suggest sub-commands.
		cmdWord := ""
		if len(leftArgs) > 0 {
			cmdWord = leftArgs[0]
		}
		suggestions := tree.suggestedSubCommands(p, cmdWord)
		if len(suggestions) > 0 {
			printLines(p.completionCtx.out, suggestions)
			return
		} else {
			if p.rootCmd != nil {
				tree.Cmd = p.rootCmd
				p.completionCtx.cmd = tree
				ctx.cmd = tree.Cmd
			}
			tree.suggestFlagAndArgs(p)
			return
		}
	}

	if p.rootCmd != nil && tree.Cmd == nil {
		tree.Cmd = p.rootCmd
		p.completionCtx.cmd = tree
		ctx.cmd = tree.Cmd
	}

	tree.suggestFlagAndArgs(p)
}

func (p *App) checkLastArgForCompletion() {
	pCtx := p.getParsingContext()
	compCtx := &p.completionCtx
	if compCtx.lastArg != "" {
		if strings.HasPrefix(compCtx.lastArg, "-") {
			if valIdx := strings.Index(compCtx.lastArg, "="); valIdx >= 0 {
				// The last incomplete word is a flag, and it wants a value.
				compCtx.wantFlagValue = true
				compCtx.flagName = compCtx.lastArg[:valIdx]
				compCtx.prefixWord = compCtx.lastArg[valIdx+1:]
			} else {
				// The last word is a flag, it could be either complete or incomplete,
				// try to suggest a flag.
				compCtx.prefixWord = strings.TrimLeft(compCtx.lastArg, "-")
			}
		} else {
			// The last incomplete word is not a flag,
			// check the second last word.
			var secondLastWord string
			if len(compCtx.userArgs) > 0 {
				secondLastWord = compCtx.userArgs[len(compCtx.userArgs)-1]
			}
			if secondLastWord != "" {
				// The second last word is a flag.
				if strings.HasPrefix(secondLastWord, "-") {
					if strings.Contains(secondLastWord, "=") {
						// The second last word is a flag and has its value,
						// the user is requesting a positional arg.
						compCtx.wantPositionalArg = true
						compCtx.prefixWord = compCtx.lastArg
					} else {
						flagName := cleanFlagName(secondLastWord)
						for _, f := range pCtx.flags {
							if f.name == flagName || f.short == flagName {
								if f.isBoolean() {
									// Boolean flags do not accept values,
									// the user is requesting a positional arg.
									compCtx.wantPositionalArg = true
									compCtx.prefixWord = compCtx.lastArg
								} else {
									// A non-boolean flag wants a value from command line.
									compCtx.wantFlagValue = true
									compCtx.flagName = flagName
									compCtx.prefixWord = compCtx.lastArg
									// The second last arg is the flag name,
									// don't pass this incomplete flag to parsing the FlagSet.
									*compCtx.cmdArgs = compCtx.userArgs[:len(compCtx.userArgs)-1]
								}
								break
							}
						}
					}
				} else if compCtx.cmd.isLeaf() {
					compCtx.wantPositionalArg = true
					compCtx.prefixWord = compCtx.lastArg
				} else if compCtx.cmd.isRoot(p) {
					compCtx.wantPositionalArg = true
					compCtx.prefixWord = compCtx.lastArg
				} else {
					// The user may be requesting a command or a positional arg.
					compCtx.prefixWord = compCtx.lastArg
				}
			} else {
				if compCtx.cmd.isRoot(p) {
					compCtx.wantPositionalArg = true
					compCtx.prefixWord = compCtx.lastArg
				}
			}
		}
	} else {
		// The last word is complete, check if it's a flag,
		// and if it's a boolean flag, a boolean flag does not take a value.
		var lastWord string
		if len(compCtx.userArgs) > 0 {
			lastWord = compCtx.userArgs[len(compCtx.userArgs)-1]
		}
		if lastWord != "" {
			if strings.HasPrefix(lastWord, "-") {
				if strings.Contains(lastWord, "=") {
					// The last word is a flag and has its value,
					// the user is most probably requesting a positional arg.
					compCtx.wantPositionalArg = true
				} else {
					flagName := cleanFlagName(lastWord)
					for _, f := range pCtx.flags {
						if f.name == flagName || f.short == flagName {
							if f.isBoolean() {
								// Boolean flags do not accept values,
								// the user is requesting a positional arg.
								compCtx.wantPositionalArg = true
							} else {
								// A non-boolean flag wants a value from command line.
								compCtx.wantFlagValue = true
								compCtx.flagName = flagName
								// The last word is the flag name,
								// don't pass this incomplete flag to parsing the FlagSet.
								*compCtx.cmdArgs = compCtx.userArgs[:len(compCtx.userArgs)-1]
							}
							break
						}
					}
				}
			} else if compCtx.cmd.isLeaf() {
				compCtx.wantPositionalArg = true
			} else if compCtx.cmd.isRoot(p) {
				compCtx.wantPositionalArg = true
			} else {
				// The user may be requesting a command or a positional arg.
				// pass
			}
		} else {
			if compCtx.cmd.isRoot(p) {
				compCtx.wantPositionalArg = true
			}
		}
	}
}

func (p *App) parseArgsForCompletion() {
	ctx := p.getParsingContext()
	fs := ctx.getFlagSet()
	cmdArgs := p.getCmdArgs()
	flagsReordered := ctx.args != nil
	if !flagsReordered {
		cmdArgs = ctx.reorderFlags(cmdArgs)
	}

	var err error

	// If the command does not receive arguments, but there are still
	// arguments before flags, it is absolutely an invalid command.
	if !checkNonflagsLength(ctx.nonflags, ctx.ambiguousArgs) {
		err = newInvalidCmdError(ctx)
		ctx.failError(err)
		return
	}

	// Expand the posix-style single-token-multiple-values flags.
	if p.Options.AllowPosixSTMO {
		cmdArgs = expandSTMOFlags(ctx.flagMap, cmdArgs)
	}

	if err = fs.Parse(cmdArgs); err != nil {
		return
	}
	nonflagArgs, err := ctx.parseNonflags()
	if err != nil {
		return
	}
	tidyFlags(fs, ctx.flags, nonflagArgs)
}

func (p *App) parseCompletionCmdTree() *cmdTree {
	p.cmds.sort()
	rootCmd := newCmdTree("", nil)
	for _, cmd := range p.cmds {
		rootCmd.add(cmd)
	}
	return rootCmd
}

type cmdTree struct {
	Name    string
	Cmd     *Command
	SubCmds []*cmdTree
	SubTree map[string]*cmdTree
}

func newCmdTree(name string, cmd *Command) *cmdTree {
	return &cmdTree{
		Name:    name,
		Cmd:     cmd,
		SubTree: make(map[string]*cmdTree),
	}
}

func (t *cmdTree) isLeaf() bool {
	return len(t.SubCmds) == 0 ||
		(t.Cmd != nil && t.Cmd == t.Cmd.app.rootCmd)
}

func (t *cmdTree) isRoot(p *App) bool {
	return t.Cmd == p.rootCmd
}

func (t *cmdTree) add(cmd *Command) {
	cmdNames := strings.Fields(cmd.Name)
	cur := t
	for i := 0; i < len(cmdNames)-1; i++ {
		name := cmdNames[i]
		subNode := cur.SubTree[name]
		if subNode == nil {
			subNode = newCmdTree(name, nil)
			cur.SubTree[name] = subNode
			cur.SubCmds = append(cur.SubCmds, subNode)
		}
		cur = subNode
	}
	lastCmdName := cmdNames[len(cmdNames)-1]
	if cur.SubTree[lastCmdName] != nil {
		cur.SubTree[lastCmdName].Cmd = cmd
	} else {
		newTree := newCmdTree(lastCmdName, cmd)
		cur.SubCmds = append(cur.SubCmds, newTree)
		cur.SubTree[lastCmdName] = newTree
	}
}

func (t *cmdTree) findCommand(cmdNames []string) (tree *cmdTree, leftArgs []string) {
	cur := t
	i := 0
	for i < len(cmdNames) {
		name := cmdNames[i]
		sub := cur.SubTree[name]
		if sub == nil {
			return cur, cmdNames[i:]
		}
		if sub.Cmd != nil && sub.Cmd.isCompletion {
			return nil, nil
		}
		cur = sub
		i++
	}
	return cur, nil
}

func (t *cmdTree) suggestedSubCommands(app *App, cmdWord string) []string {
	matchFunc := func(n *cmdTree) bool {
		return strings.HasPrefix(n.Name, cmdWord)
	}
	result := make([]string, 0, 16)
	for _, sub := range t.SubCmds {
		if sub.Cmd == nil && len(sub.SubCmds) == 0 {
			continue
		}
		if sub.Cmd != nil && (sub.Cmd.isCompletion || sub.Cmd.Hidden) {
			continue
		}
		if !matchFunc(sub) {
			continue
		}
		desc := ""
		if sub.Cmd != nil {
			desc = sub.Cmd.Description
		}
		suggestion := app.formatCompletion(sub.Name, desc)
		result = append(result, suggestion)
	}
	return result
}

func (t *cmdTree) suggestFlagAndArgs(app *App) {
	cmd := t.Cmd
	if cmd == nil || cmd.isGroup || cmd.isCompletion {
		return
	}

	// check that flag completion is enabled for the command.
	cmdOpts := newCmdOptions(cmd.cmdOpts...)
	isCmdFlagEnabled := app.EnableFlagCompletionForAllCommands || cmdOpts.enableFlagCompletion
	if !isCmdFlagEnabled {
		return
	}

	// Parse flags for the command,
	// then transmit the executing to the parsing function.
	cmd.f()
}

func (p *App) continueCompletion(parsedArgs any) {
	compCtx := &p.completionCtx
	compCtx.parsedArgs = parsedArgs
	p.checkLastArgForCompletion()
	p.parseArgsForCompletion()
	defer p.completionCtx.postFunc()

	if compCtx.wantPositionalArg {
		p.continuePositionalArgCompletion()
		return
	}
	if compCtx.wantFlagValue {
		p.continueFlagValueCompletion()
		return
	}

	// Else try to complete flags.
	p.continueFlagCompletion()
}

func (p *App) continueFlagCompletion() {
	compCtx := &p.completionCtx
	getUsage := func(f *_flag) string {
		if compCtx.shell == "powershell" {
			return ""
		}
		_, usage := f.getUsage(false)
		usage = strings.TrimSpace(usage)
		nIdx := strings.IndexByte(usage, '\n')
		if nIdx > 0 {
			usage = usage[:nIdx] + " ..."
		}
		return usage
	}

	seenFlags := make(map[string]bool)
	cmdArgs := p.getCmdArgs()
	for _, arg := range cmdArgs {
		if !strings.HasPrefix(arg, "-") {
			continue
		}
		arg = strings.TrimLeft(arg, "-")
		valIdx := strings.IndexByte(arg, '=')
		if valIdx > 0 {
			arg = arg[:valIdx]
		}
		if arg != "" {
			seenFlags[arg] = true
		}
	}
	isSeenFlag := func(f *_flag) bool {
		return seenFlags[f.short] || seenFlags[f.name]
	}

	pCtx := p.getParsingContext()
	prefixWord := compCtx.prefixWord
	result := make([]string, 0, 16)
	for _, flag := range pCtx.flags {
		if flag.short != "" && strings.HasPrefix(flag.short, prefixWord) && (flag.isCompositeType() || !isSeenFlag(flag)) {
			usage := getUsage(flag)
			suggestion := p.formatCompletion("-"+flag.short, usage)
			result = append(result, suggestion)
		}

		if flag.name != "" && strings.HasPrefix(flag.name, prefixWord) && (flag.isCompositeType() || !isSeenFlag(flag)) {
			usage := getUsage(flag)
			suggestion := p.formatCompletion("--"+flag.name, usage)
			result = append(result, suggestion)
		}
	}
	printLines(p.completionCtx.out, result)
}

func (p *App) continueFlagValueCompletion() {
	pCtx := p.getParsingContext()
	compCtx := &p.completionCtx
	flagName := cleanFlagName(compCtx.flagName)

	var f *_flag
	for _, x := range pCtx.flags {
		if x.name == flagName || x.short == flagName {
			f = x
			break
		}
	}
	if f == nil {
		return
	}
	compFunc := pCtx.opts.argCompFuncs["-"+f.name]
	if compFunc == nil {
		compFunc = pCtx.opts.argCompFuncs["-"+f.short]
		if compFunc == nil {
			return
		}
	}
	acc := p.newArgCompletionContext()
	compItems := compFunc(acc)
	p.printCompletionItems(compItems)
}

func (p *App) continuePositionalArgCompletion() {
	pCtx := p.getParsingContext()
	fs := pCtx.getFlagSet()

	if len(pCtx.nonflags) == 0 {
		return
	}

	nf := pCtx.nonflags[0]
	var i, j int
	for i <= fs.NArg() && j < len(pCtx.nonflags) {
		nf = pCtx.nonflags[j]
		if !nf.isCompositeType() {
			j++
		}
		i++
	}
	if i <= fs.NArg() {
		// There are still more positional args, this completion request is invalid.
		return
	}

	compFunc := pCtx.opts.argCompFuncs[nf.name]
	if compFunc == nil {
		return
	}
	acc := p.newArgCompletionContext()
	compItems := compFunc(acc)
	p.printCompletionItems(compItems)
}

func (p *App) printCompletionItems(items []CompletionItem) {
	result := make([]string, 0, len(items))
	for _, x := range items {
		s := p.formatCompletion(x.Value, x.Description)
		result = append(result, s)
	}
	printLines(p.completionCtx.out, result)
}

func (p *App) formatCompletion(opt string, desc string) string {
	if desc == "" {
		return opt
	}

	switch p.completionCtx.shell {
	case "bash":
		return fmt.Sprintf("%s\t%s", opt, desc)
	case "zsh":
		return fmt.Sprintf("%s:%s", opt, desc)
	case "fish":
		return fmt.Sprintf("%s\t%s", opt, desc)
	default:
		return opt
	}
}

func printLines(w io.Writer, lines []string) {
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
}

func (p *App) addCompletionCommands(name string) {
	p.completionCmdName = name

	grpCmd := newUntypedCommand(func() {
		p.parseArgs(nil, DisableGlobalFlags())
		p.printUsage()
	})
	grpCmd.isCompletion = true
	p._add(name, grpCmd, "Generate shell completion scripts")

	for _, shell := range getAllowedShells() {
		cmdName := name + " " + shell
		desc := "Generate the completion script for " + shell
		compCmd := p.completionCmd(shell)
		shellCmd := p._add(cmdName, compCmd, desc)
		shellCmd.isCompletion = true
	}
}

func (p *App) completionCmd(shellType string) func() {
	return func() {
		customUsage := p.completionUsage(shellType)
		p.parseArgs(nil, DisableGlobalFlags(), ReplaceUsage(customUsage))

		data := map[string]any{
			"ProgramName":       getProgramName(),
			"CompletionCmdName": p.completionCmdName,
		}

		tplName := ""
		switch shellType {
		case "bash":
			tplName = "autocomplete/bash_autocomplete"
		case "zsh":
			tplName = "autocomplete/zsh_autocomplete"
		case "powershell":
			tplName = "autocomplete/powershell_autocomplete.ps1"
		case "fish":
			tplName = "autocomplete/fish_autocomplete"
		default:
			panic("unreachable")
		}
		tplContent, err := autoCompleteTpl.ReadFile(tplName)
		if err != nil {
			panic("unreachable")
		}

		tpl := template.Must(template.New("").Parse(string(tplContent)))
		builder := &strings.Builder{}
		tpl.Execute(builder, data)
		fmt.Println(builder.String())
	}
}

// Templates forked from github.com/urfave/cli/v2/autocomplete.
//
//go:embed autocomplete
var autoCompleteTpl embed.FS
