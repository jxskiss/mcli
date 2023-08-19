package mcli

import (
	"fmt"
	"io"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
)

func newUsagePrinter(app *App) *usagePrinter {
	ctx := app.getParsingContext()
	out := ctx.getFlagSet().Output()
	return &usagePrinter{
		app: app,
		ctx: ctx,
		out: out,
	}
}

type usagePrinter struct {
	app *App
	ctx *parsingContext
	out io.Writer

	flagCount    int
	hasShortFlag bool

	subCmds        commands
	globalFlagHelp [][2]string
	cmdFlagHelp    [][2]string
	nonFlagHelp    [][2]string
}

func (p *usagePrinter) Do() {
	ctx := p.ctx
	out := p.out
	if ctx.opts.customUsage != nil {
		help := strings.TrimSpace(heredoc.Doc(ctx.opts.customUsage()))
		fmt.Fprintf(out, "%s\n\n", help)
		return
	}

	globalFlags := p.app.getGlobalFlags()
	if !ctx.parsed && globalFlags != nil {
		wrapArgs := &withGlobalFlagArgs{
			GlobalFlags: globalFlags,
		}
		err := ctx.parseTags(reflect.ValueOf(wrapArgs).Elem())
		if err != nil {
			return
		}
	}

	cmdName := ctx.name
	cmds := p.app.cmds
	p.subCmds = cmds.listSubCommandsToPrint(cmdName, ctx.showHidden)

	p.printUsageLine()
	p.printSubCommands()
	p.countFlags()
	p.splitAndFormatFlags()
	p.printCmdFlags()
	p.printArguments()
	p.printGlobalFlags()
	p.printExamples()
	p.printFooter()
}

func (p *usagePrinter) printUsageLine() {
	usage := ""
	ctx := p.ctx
	out := p.out
	cmd := ctx.cmd
	cmdName := ctx.name
	progName := getProgramName()
	appDesc := strings.TrimSpace(p.app.Description)

	if cmd != nil {
		cmdOpts := newCmdOptions(cmd.cmdOpts...)
		if cmd.isRoot {
			if appDesc != "" {
				usage += appDesc + "\n"
			}
		} else {
			if cmd.AliasOf != "" {
				usage += cmd.Description + "\n"
				cmd = p.app.cmdMap[cmd.AliasOf]
				cmdName = cmd.Name
			}
			if cmd.Description != "" {
				usage += cmd.Description + "\n"
			}
		}
		if cmdOpts.longDesc != "" {
			if usage != "" {
				usage += "\n"
			}
			usage += cmdOpts.longDesc + "\n"
		}
	} else if appDesc != "" {
		usage += appDesc + "\n"
	}
	if usage != "" {
		usage += "\n"
	}
	usage += "USAGE:\n  " + progName
	if cmd != nil && cmd.isRoot {
		usage += p.commandLineFlagAndSubCmdInfo("")
		if len(p.app.cmds) > 0 {
			usage += "\n  " + progName + " <command> [flags] ..."
		}
	} else {
		usage += p.commandLineFlagAndSubCmdInfo(cmdName)
	}
	fmt.Fprint(out, usage, "\n\n")
}

func (p *usagePrinter) commandLineFlagAndSubCmdInfo(cmdName string) string {
	ctx := p.ctx
	hasFlags := len(ctx.flags) > 0
	hasNonflags := len(ctx.nonflags) > 0
	hasSubCmds := len(p.subCmds) > 0

	usage := ""
	if cmdName != "" {
		usage += " " + cmdName
	}
	if hasFlags {
		usage += " [flags]"
	}
	if hasNonflags {
		for _, f := range ctx.nonflags {
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
	return usage
}

func (p *usagePrinter) printSubCommands() {
	ctx := p.ctx
	out := p.out
	if len(p.subCmds) > 0 {
		subCmds := p.subCmds
		showHidden := ctx.showHidden
		keepCmdOrder := p.app.Options.KeepCommandOrder
		printSubCommands(out, subCmds, showHidden, keepCmdOrder)
	}
}

func (p *usagePrinter) countFlags() {
	flags := p.ctx.flags
	showHidden := p.ctx.showHidden
	for _, f := range flags {
		if !f.hidden || showHidden {
			p.flagCount++
			p.hasShortFlag = p.hasShortFlag || f.short != ""
		}
	}
}

func (p *usagePrinter) splitAndFormatFlags() {
	flags := p.ctx.flags
	showHidden := p.ctx.showHidden
	hasShortFlag := p.hasShortFlag

	var globalFlagHelp [][2]string
	var cmdFlagHelp [][2]string
	var nonFlagHelp [][2]string
	if p.flagCount > 0 {
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
	for _, f := range p.ctx.nonflags {
		name, usage := f.getUsage(false)
		nonFlagHelp = append(nonFlagHelp, [2]string{name, usage})
	}
	p.globalFlagHelp = globalFlagHelp
	p.cmdFlagHelp = cmdFlagHelp
	p.nonFlagHelp = nonFlagHelp
}

func (p *usagePrinter) printCmdFlags() {
	out := p.out
	if len(p.cmdFlagHelp) > 0 {
		fmt.Fprint(out, "FLAGS:\n")
		printWithAlignment(out, p.cmdFlagHelp, 0)
		fmt.Fprint(out, "\n")
	}
}

func (p *usagePrinter) printArguments() {
	out := p.out
	if len(p.nonFlagHelp) > 0 {
		fmt.Fprint(out, "ARGUMENTS:\n")
		printWithAlignment(out, p.nonFlagHelp, 0)
		fmt.Fprint(out, "\n")
	}
}

func (p *usagePrinter) printGlobalFlags() {
	out := p.out
	if len(p.globalFlagHelp) > 0 {
		fmt.Fprint(out, "GLOBAL FLAGS:\n")
		printWithAlignment(out, p.globalFlagHelp, 0)
		fmt.Fprint(out, "\n")
	}
}

var blankLineRE = regexp.MustCompile(`\n\s+\n`)

func (p *usagePrinter) printExamples() {
	ctx := p.ctx
	out := p.out

	if ctx.opts.examples != "" {
		examples := strings.ReplaceAll(ctx.opts.examples, "\n", "\n  ")
		examples = blankLineRE.ReplaceAllString(examples, "\n\n")
		fmt.Fprint(out, "EXAMPLES:\n  ")
		fmt.Fprintf(out, "%s\n\n", examples)
	}
}

func (p *usagePrinter) printFooter() {
	ctx := p.ctx
	out := p.out
	if ctx.opts.helpFooter != nil {
		footer := strings.TrimSpace(ctx.opts.helpFooter())
		fmt.Fprintf(out, "%s\n\n", footer)
	} else if p.app.HelpFooter != "" {
		footer := strings.TrimSpace(p.app.HelpFooter)
		fmt.Fprintf(out, "%s\n\n", footer)
	}
}

func printSubCommands(out io.Writer, cmds commands, showHidden, keepCmdOrder bool) {
	if len(cmds) == 0 {
		return
	}
	if keepCmdOrder {
		sort.Slice(cmds, func(i, j int) bool {
			return cmds[i].idx < cmds[j].idx
		})
	}

	cmdGroups, hasCategories := cmds.groupByCategory()
	if hasCategories {
		printGroupedSubCommands(out, cmdGroups, showHidden, keepCmdOrder)
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
		if cmd.isCompletion && leafCmdName != cmd.Name {
			continue
		}
		name := strings.Repeat("  ", len(prefix)) + leafCmdName
		description := cmd.Description
		if cmd.Hidden {
			name += " (HIDDEN)"
		}
		cmdLines = append(cmdLines, [2]string{name, description})
		preName = cmd.Name
	}
	fmt.Fprint(out, "COMMANDS:\n")
	printWithAlignment(out, cmdLines, 0)
	fmt.Fprint(out, "\n")
}

func printGroupedSubCommands(out io.Writer, cmdGroups []*commandGroup, showHidden, keepCmdOrder bool) {
	type groupCmdLines struct {
		category string
		cmdLines [][2]string
	}

	if !keepCmdOrder {
		sort.Slice(cmdGroups, func(i, j int) bool {
			return cmdGroups[i].category < cmdGroups[j].category
		})
	}

	var groupLines []*groupCmdLines
	var cmdLines [][][2]string
	for _, grp := range cmdGroups {
		var grpLines [][2]string
		for _, cmd := range grp.commands {
			if cmd.Name == "" || (cmd.Hidden && !showHidden) || cmd.level > 1 {
				continue
			}
			name := "  " + cmd.Name
			description := cmd.Description
			if cmd.Hidden {
				name += " (HIDDEN)"
			}
			grpLines = append(grpLines, [2]string{name, description})
		}
		if len(grpLines) == 0 {
			continue
		}
		groupLines = append(groupLines, &groupCmdLines{
			category: grp.category,
			cmdLines: grpLines,
		})
		cmdLines = append(cmdLines, grpLines)
	}

	maxPrefixLen := calcMaxPrefixLen(cmdLines)
	for _, grp := range groupLines {
		fmt.Fprint(out, addTrailingColon(grp.category)+"\n")
		printWithAlignment(out, grp.cmdLines, maxPrefixLen)
		fmt.Fprint(out, "\n")
	}
}

func addTrailingColon(s string) string {
	if !strings.HasSuffix(s, ":") {
		s += ":"
	}
	return s
}

const (
	__MaxPrefixLen = 36
)

func printWithAlignment(out io.Writer, lines [][2]string, maxPrefixLen int) {
	if maxPrefixLen <= 0 {
		maxPrefixLen = calcMaxPrefixLen([][][2]string{lines})
	}
	padding := "\n" + strings.Repeat(" ", maxPrefixLen+4)
	for _, line := range lines {
		x, y := line[0], line[1]
		fmt.Fprint(out, x)
		if y != "" {
			if len(x) <= __MaxPrefixLen {
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

func calcMaxPrefixLen(lineGroups [][][2]string) int {
	maxPrefixLen := 0
	for _, lines := range lineGroups {
		for _, line := range lines {
			if n := len(line[0]); n > maxPrefixLen && n <= __MaxPrefixLen {
				maxPrefixLen = n
			}
		}
	}
	return maxPrefixLen
}
