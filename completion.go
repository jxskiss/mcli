package mcli

import (
	"embed"
	"fmt"
	"io"
	"strings"
	"text/template"
)

const completionFlag = "--mcli-generate-completion"

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

func isFlagCompletion(args []string) (isFlag bool, flagName string, userArgs []string) {
	var lastArg string
	if len(args) > 0 {
		lastArg = args[len(args)-1]
	}
	if strings.HasPrefix(lastArg, "-") {
		return true, lastArg, args[:len(args)-1]
	}

	// User has provided other flags, this completion request is for another flag.
	hasFlag := false
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			hasFlag = true
		}
	}
	return hasFlag, "", args
}

func (p *App) doAutoCompletion(args []string) {
	tree := p.parseCompletionInfo()
	isFlag, flagName, userArgs := isFlagCompletion(args)
	if isFlag {
		tree.suggestFlags(p, userArgs, flagName)
	} else {
		checkFlag := tree.suggestCommands(p, userArgs)
		if checkFlag {
			tree.suggestFlags(p, userArgs, "")
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

func (t *cmdTree) suggestCommands(app *App, cmdNames []string) (checkFlag bool) {
	cur := t
	i := 0
	for i < len(cmdNames)-1 {
		name := cmdNames[i]
		cur = cur.SubTree[name]
		if cur == nil || (cur.Cmd != nil && cur.Cmd.isCompletion) {
			return false
		}
		i++
	}
	gotCmd := true
	lastWord := ""
	if len(cmdNames) > 0 {
		lastWord = cmdNames[len(cmdNames)-1]
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

func (t *cmdTree) suggestFlags(app *App, userArgs []string, flagName string) {
	cmdNames := userArgs
	flagIdx := -1
	for i, arg := range userArgs {
		if strings.HasPrefix(arg, "-") {
			flagIdx = i
			break
		}
	}
	if flagIdx >= 0 {
		cmdNames = userArgs[:flagIdx]
	}
	cur := t
	for _, name := range cmdNames {
		cur = cur.SubTree[name]
		if cur == nil {
			return
		}
	}
	if cur.Cmd == nil || cur.Cmd.isGroup || cur.Cmd.isCompletion {
		return
	}

	// Check that flag completion is enabled for the command.
	cmdOpts := newCmdOptions(cur.Cmd.cmdOpts...)
	isCmdFlagEnabled := app.EnableFlagCompletionForAllCommands || cmdOpts.enableFlagCompletion
	if !isCmdFlagEnabled {
		return
	}

	// Parse flags for the command,
	// then transmit the executing to the parsing function.
	app.completionCtx.cmd = cur
	app.completionCtx.flagName = flagName
	cur.Cmd.f()
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

	result := make([]string, 0, 16)
	hyphenCount, flagName := countFlagPrefixHyphen(flagName)
	if hyphenCount <= 1 {
		for _, flag := range flags {
			if flag.short != "" && strings.HasPrefix(flag.short, flagName) &&
				(flag.isCompositeType() || !isSeenFlag(flag)) {
				usage := getUsage(flag)
				suggestion := formatCompletion(p, "-"+flag.short, usage)
				result = append(result, suggestion)
			} else if flag.name != "" && strings.HasPrefix(flag.name, flagName) &&
				(flag.isCompositeType() || !isSeenFlag(flag)) {
				usage := getUsage(flag)
				suggestion := formatCompletion(p, "--"+flag.name, usage)
				result = append(result, suggestion)
			}
		}
	} else {
		for _, flag := range flags {
			if flag.name != "" && strings.HasPrefix(flag.name, flagName) &&
				(flag.isCompositeType() || !isSeenFlag(flag)) {
				suggestion := formatCompletion(p, "--"+flag.name, getUsage(flag))
				result = append(result, suggestion)
			}
		}
	}
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
