package mcli

import (
	"embed"
	"fmt"
	"io"
	"os"
	"strings"
)

const completionFlag = "--mcli-generate-completion"

func hasCompletionFlag(args []string) (bool, []string) {
	var lastArg string
	if len(args) > 0 {
		lastArg = args[len(args)-1]
	}
	if lastArg == completionFlag {
		return true, args[:len(args)-1]
	}
	return false, args
}

func isFlagCompletion(args []string) (isFlag bool, flagName string, userArgs []string) {
	var lastArg string
	if len(args) > 0 {
		lastArg = args[len(args)-1]
	}
	if strings.HasPrefix(lastArg, "-") {
		return true, lastArg, args[:len(args)-1]
	}
	return false, "", args
}

func (p *App) doAutoCompletion(args []string) {
	tree := p.parseCompletionInfo()
	isFlag, flagName, userArgs := isFlagCompletion(args)
	if isFlag {
		tree.suggestFlags(p, userArgs, flagName)
	} else {
		tree.suggestCommands(p, userArgs)
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

func (t *cmdTree) suggestCommands(app *App, cmdNames []string) {
	cur := t
	i := 0
	for i < len(cmdNames)-1 {
		name := cmdNames[i]
		cur = cur.SubTree[name]
		if cur == nil || (cur.Cmd != nil && cur.Cmd.noCompletion) {
			return
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
	matchFunc := func(n *cmdTree) bool { return true }
	if !gotCmd {
		matchFunc = func(n *cmdTree) bool {
			return strings.HasPrefix(n.Name, lastWord)
		}
	}
	result := make([]string, 0, 16)
	for _, sub := range cur.SubCmds {
		if sub.Cmd == nil || sub.Cmd.noCompletion || !matchFunc(sub) {
			continue
		}
		desc := ""
		if app.completionCtx.isZsh {
			desc = sub.Cmd.Description
		}
		suggestion := formatCompletion(sub.Name, desc)
		result = append(result, suggestion)
	}
	printLines(os.Stdout, result)
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
	if cur.Cmd == nil || cur.Cmd.isGroup || cur.Cmd.noCompletion {
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
		if !p.completionCtx.isZsh {
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
	for _, arg := range p.completionCtx.userArgs {
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
	dashCount, flagName := countFlagPrefixDash(flagName)
	if dashCount == 1 {
		for _, flag := range flags {
			if flag.short != "" && strings.HasPrefix(flag.short, flagName) && !isSeenFlag(flag) {
				usage := getUsage(flag)
				suggestion := formatCompletion("-"+flag.short, usage)
				result = append(result, suggestion)
			} else if flag.name != "" && strings.HasPrefix(flag.name, flagName) && !isSeenFlag(flag) {
				usage := getUsage(flag)
				hint := formatCompletion("--"+flag.name, usage)
				result = append(result, hint)
			}
		}
	} else {
		for _, flag := range flags {
			if flag.name != "" && strings.HasPrefix(flag.name, flagName) && !isSeenFlag(flag) {
				suggestion := formatCompletion("--"+flag.name, getUsage(flag))
				result = append(result, suggestion)
			}
		}
	}
	printLines(os.Stdout, result)
}

func countFlagPrefixDash(flagName string) (int, string) {
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

func formatCompletion(opt, desc string) string {
	if desc == "" {
		return opt
	}
	return fmt.Sprintf("%s:%s", opt, desc)
}

func (p *App) addCompletionCommands(name string) {
	p.completionCmdName = name
	p.AddGroup(name, "Generate autocompletion script for the specified shell")
	p.addCommand(&Command{
		Name:         name + " bash",
		Description:  "Generate the autocompletion script for bash",
		f:            p.completionCmd("bash"),
		noCompletion: true,
	})
	p.addCommand(&Command{
		Name:         name + " zsh",
		Description:  "Generate the autocompletion script for zsh",
		f:            p.completionCmd("zsh"),
		noCompletion: true,
	})
	p.addCommand(&Command{
		Name:         name + " powershell",
		Description:  "Generate the autocompletion script for powershell",
		f:            p.completionCmd("powershell"),
		noCompletion: true,
	})
}

func (p *App) completionCmd(shellType string) func() {
	return func() {
		customUsage := p.completionUsage(shellType)
		p.parseArgs(nil, ReplaceUsage(customUsage))

		tplName := ""
		switch shellType {
		case "bash":
			tplName = "autocomplete/bash_autocomplete"
		case "zsh":
			tplName = "autocomplete/zsh_autocomplete"
		case "powershell":
			tplName = "autocomplete/powershell_autocomplete.ps1"
		default: // shall never happen
			panic(bugReportTip)
		}
		tplContent, err := autoCompleteTpl.ReadFile(tplName)
		if err != nil { // shall never happen
			panic(err)
		}
		fmt.Println(string(tplContent))
	}
}

// Templates forked from github.com/urfave/cli/v2/autocomplete.
//
//go:embed autocomplete
var autoCompleteTpl embed.FS

const bugReportTip = "bug: please report to https://github.com/jxskiss/mcli"
