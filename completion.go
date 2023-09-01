package mcli

import (
	"embed"
	"fmt"
	"io"
	"slices"
	"strings"
	"text/template"
)

const completionFlag = "--mcli-generate-completion"

type completionMethod struct {
	isFlag       bool
	flagName     string
	isFlagValue  bool
	isCommand    bool
	foundCommand *Command
	userArgs     []string
	isCommandArg bool
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
		if slices.Contains(getAllowedShells(), proposedShell) {
			shell = proposedShell
		}
	}
	args = args[:completionFlagIndex]
	return true, args, shell
}

func completedCommand(args []string, c commands) *Command {
	cReversed := reverse(c)
	line := strings.Join(args, " ")
	for _, item := range cReversed {
		if strings.HasPrefix(line, item.Name) {
			return item
		}
	}
	return nil
}

func detectCompletionMethod(args []string, c commands) completionMethod {
	var lastArg string
	if len(args) > 0 {
		lastArg = args[len(args)-1]
	}

	if lastArg == "-" {
		return completionMethod{
			isFlag:   true,
			flagName: lastArg,
			userArgs: args[:len(args)-1],
		}
	}

	if strings.HasPrefix(lastArg, "-") {
		return completionMethod{
			isFlagValue: true,
			flagName:    lastArg,
			userArgs:    args,
		}
	}

	catched := completedCommand(args, c)
	if catched != nil {
		if catched.isGroup {
			return completionMethod{
				isCommand:    true,
				flagName:     "",
				userArgs:     args,
				foundCommand: catched,
			}
		} else {
			return completionMethod{
				flagName:     "",
				userArgs:     args,
				isCommandArg: true,
				foundCommand: catched,
			}
		}
	}

	// User has provided other flags, this completion request is for another flag.
	return completionMethod{
		flagName: "",
		userArgs: args,
	}
}

// TODO: cleanup - use in tests | 2023-08-31
// 'server s' -- argument poprzedzający brak lub komenda -> uzupełniamy komendę/subkomendę
// - jeśli nie ma dalszych subkomend? jak idzie sprawdzić to generowanie flag
// 'server s8 -' - uzupelnienie flagi
// 'server s8 --x2 ' - uzupełnienie wartości flagi
// 'server s8 --x2 v' - uzupełnienie wartości flagi
func (p *App) doAutoCompletion(args []string) {
	tree := p.parseCompletionInfo()
	cm := detectCompletionMethod(args, p.cmds)
	// fmt.Printf("%+v\n", cm)

	if cm.isCommand {
		tree.suggestCommands(p, cm)
	} else if cm.isCommandArg {
		tree.suggestCommandArgs(p, cm)
	} else if cm.isFlag {
		tree.suggestFlags(p, cm)
	} else if cm.isFlagValue {
		checkCommandArg := tree.suggestFlags(p, cm)
		if checkCommandArg {
			tree.suggestCommandArgs(p, cm)
		}
	} else {
		checkFlag := tree.suggestCommands(p, cm)
		if checkFlag {
			tree.suggestFlags(p, cm)
		}
	}
}

func (p *App) parseCompletionInfo() *cmdTree {
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

func (t *cmdTree) suggestCommands(app *App, cm completionMethod) (checkFlag bool) {
	userArgs := cm.userArgs
	cur := t
	i := 0
	for i < len(userArgs)-1 {
		name := userArgs[i]
		cur = cur.SubTree[name]
		if cur == nil || (cur.Cmd != nil && cur.Cmd.isCompletion) {
			return false
		}
		i++
	}
	gotCmd := true
	lastWord := ""
	if len(userArgs) > 0 {
		lastWord = userArgs[len(userArgs)-1]
		if n, ok := cur.SubTree[lastWord]; ok {
			cur = n
		} else {
			gotCmd = false
		}
	}
	// If the command is a leaf command, check flag for completion.
	if gotCmd && len(cur.SubCmds) == 0 {
		return true
	}

	matchFunc := func(n *cmdTree) bool { return true }
	if !gotCmd {
		matchFunc = func(n *cmdTree) bool {
			return strings.HasPrefix(n.Name, lastWord)
		}
	}
	result := make([]string, 0, 16)
	for _, sub := range cur.SubCmds {
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
		suggestion := formatCompletion(app, sub.Name, desc)
		result = append(result, suggestion)
	}

	printLines(app.completionCtx.out, result)
	return false
}

func (t *cmdTree) suggestCommandArgs(app *App, cm completionMethod) {
	result := []string{}
	if cm.foundCommand != nil {
		cmdOpts := newCmdOptions(cm.foundCommand.cmdOpts...)
		app.argsCtx = &compContextImpl{
			app:  app,
			args: cm.userArgs,
		}
		f := cmdOpts.argCompFunc
		if f != nil {
			res, _ := f(app.argsCtx)
			for _, item := range res {
				result = append(result, formatCompletion(app, item[0], item[1]))
			}
		}
		printLines(app.completionCtx.out, result)
	}
	// TODO: directive is cut, and not used | 2023-08-30
	// printLines(p.completionCtx.out, directive)
}

func (t *cmdTree) suggestFlags(app *App, cm completionMethod) bool {
	cmd := completedCommand(cm.userArgs, app.cmds)
	if cmd == nil || cmd.isGroup || cmd.isCompletion {
		return false
	}

	// Check that flag completion is enabled for the command.
	cmdOpts := newCmdOptions(cmd.cmdOpts...)
	isCmdFlagEnabled := app.EnableFlagCompletionForAllCommands || cmdOpts.enableFlagCompletion
	if !isCmdFlagEnabled {
		return true
	}

	// Parse flags for the command,
	// then transmit the executing to the parsing function.
	app.completionCtx.flagName = cm.flagName
	cmd.f()
	return true
}

func (p *App) continueFlagCompletion() {
	getUsage := func(f *_flag) string {
		if p.completionCtx.shell == "powershell" {
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
	leadingArgs := subSlice(p.completionCtx.userArgs, 0, -1)
	for _, arg := range leadingArgs {
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

	flagName := p.completionCtx.flagName
	flags := p.completionCtx.flags
	funcs := p.completionCtx.argCompFuncs

	result := make([]string, 0, 16)
	completionFunc := ""

	_, cleanFlagName := countFlagPrefixHyphen(flagName)

	for _, flag := range flags {
		usage := getUsage(flag)
		// flagShort := "--" + flag.short
		// flagLong := "--" + flag.name
		if flag.short != "" && flag.short != "-" && strings.HasPrefix(flag.short, cleanFlagName) && (flag.isCompositeType() || !isSeenFlag(flag)) {
			if flag.completionFunction != "" {
				if len(cleanFlagName) == len(flag.short) {
					completionFunc = flag.completionFunction
				}
			}

			suggestion := formatCompletion(p, "-"+flag.short, usage)
			if !slices.Contains(p.completionCtx.userArgs, "-"+flag.short) {
				result = append(result, suggestion)
			}
		}

		if flag.name != "" && flag.name != "--" && strings.HasPrefix(flag.name, cleanFlagName) && (flag.isCompositeType() || !isSeenFlag(flag)) {
			if flag.completionFunction != "" {
				if len(cleanFlagName) == len(flag.name) {
					completionFunc = flag.completionFunction
				}
			}
			suggestion := formatCompletion(p, "--"+flag.name, usage)
			if !slices.Contains(p.completionCtx.userArgs, "--"+flag.name) {
				result = append(result, suggestion)
			}
		}
	}

	// fmt.Println(result)
	if completionFunc != "" {
		if f, ok := funcs[completionFunc]; ok {
			res, _ := f(p.argsCtx)
			result = []string{}
			for _, item := range res {
				result = append(result, formatCompletion(p, item[0], item[1]))
			}
		} else {
			panic(fmt.Sprintf("mcli: flag argument completion called not passed function '%s'", completionFunc))
		}
	}

	// TODO: directive is cut, and not used | 2023-08-30
	// printLines(p.completionCtx.out, directive)
	printLines(p.completionCtx.out, result)
}

func countFlagPrefixHyphen(flagName string) (int, string) {
	n := 0
	for _, c := range flagName {
		if c == '-' {
			n++
			continue
		}
		break
	}
	return n, flagName[n:]
}

func printLines(w io.Writer, lines []string) {
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
}

func formatCompletion(app *App, opt string, desc string) string {
	if desc == "" {
		return opt
	}

	switch app.completionCtx.shell {
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
