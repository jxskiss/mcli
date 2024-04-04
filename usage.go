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
	globalFlagHelp []usageItem
	cmdFlagHelp    []usageItem
	nonFlagHelp    []usageItem
	envVarsHelp    []usageItem
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
	p.printEnvVariables()
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
	usage += "Usage:\n  " + progName
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
		parentCmdName := ctx.name
		subCmds := p.subCmds
		showHidden := ctx.showHidden
		keepCmdOrder := p.app.Options.KeepCommandOrder
		p.__printSubCommands(out, subCmds, parentCmdName, showHidden, keepCmdOrder)
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

	var (
		globalFlagHelp []usageItem
		cmdFlagHelp    []usageItem
		nonFlagHelp    []usageItem
		envVarsHelp    []usageItem
	)
	if p.flagCount > 0 {
		for _, f := range flags {
			if f.hidden && !showHidden {
				continue
			}
			usage := f.getUsage(hasShortFlag)
			if f.isGlobal {
				globalFlagHelp = append(globalFlagHelp, usage)
			} else {
				cmdFlagHelp = append(cmdFlagHelp, usage)
			}
		}
	}
	for _, f := range p.ctx.nonflags {
		usage := f.getUsage(false)
		nonFlagHelp = append(nonFlagHelp, usage)
	}
	for _, f := range p.ctx.envVars {
		usage := f.getUsage(false)
		envVarsHelp = append(envVarsHelp, usage)
	}
	p.globalFlagHelp = globalFlagHelp
	p.cmdFlagHelp = cmdFlagHelp
	p.nonFlagHelp = nonFlagHelp
	p.envVarsHelp = envVarsHelp
}

func (p *usagePrinter) printCmdFlags() {
	out := p.out
	if len(p.cmdFlagHelp) > 0 {
		fmt.Fprint(out, "Flags:\n")
		printWithAlignment(out, p.cmdFlagHelp, 0)
		fmt.Fprint(out, "\n")
	}
}

func (p *usagePrinter) printArguments() {
	out := p.out
	if len(p.nonFlagHelp) > 0 {
		fmt.Fprint(out, "Arguments:\n")
		printWithAlignment(out, p.nonFlagHelp, 0)
		fmt.Fprint(out, "\n")
	}
}

func (p *usagePrinter) printGlobalFlags() {
	out := p.out
	if len(p.globalFlagHelp) > 0 {
		fmt.Fprint(out, "Global Flags:\n")
		printWithAlignment(out, p.globalFlagHelp, 0)
		fmt.Fprint(out, "\n")
	}
}

func (p *usagePrinter) printEnvVariables() {
	out := p.out
	padding := "    "
	if len(p.envVarsHelp) > 0 {
		fmt.Fprint(out, "Environment Variables:\n")
		for _, line := range p.envVarsHelp {
			x, y := line.prefix, line.description
			fmt.Fprintf(out, "%s\n", x)
			if y != "" {
				fmt.Fprintf(out, "%s%s\n", padding, strings.ReplaceAll(y, "\n", "\n"+padding))
			}
		}
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
		fmt.Fprint(out, "Examples:\n  ")
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

func (p *usagePrinter) __printSubCommands(out io.Writer, cmds commands, parentCmdName string, showHidden, keepCmdOrder bool) {
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
		p.__printGroupedSubCommands(out, cmdGroups, showHidden)
		return
	}

	var cmdLines []usageItem
	prefix := []string{""}
	preName := ""
	for _, cmd := range cmds {
		cmdName := trimPrefix(cmd.Name, parentCmdName)
		if cmdName == "" || (cmd.Hidden && !showHidden) {
			continue
		}
		if preName != "" && cmdName != preName {
			if strings.HasPrefix(cmdName, preName) {
				prefix = append(prefix, preName)
			} else {
				for i := len(prefix) - 1; i > 0; i-- {
					if !strings.HasPrefix(cmdName, prefix[i]) {
						prefix = prefix[:i]
					}
				}
			}
		}
		leafCmdName := trimPrefix(cmdName, prefix[len(prefix)-1])
		if cmd.isCompletion && leafCmdName != cmdName {
			continue
		}
		name := strings.Repeat("  ", len(prefix)) + leafCmdName
		description := cmd.Description
		if cmd.Hidden {
			name += " (HIDDEN)"
		}
		cmdLines = append(cmdLines, usageItem{
			prefix:      name,
			description: description,
		})
		preName = cmdName
	}
	fmt.Fprint(out, "Commands:\n")
	printWithAlignment(out, cmdLines, 0)
	fmt.Fprint(out, "\n")
}

func (p *usagePrinter) __printGroupedSubCommands(out io.Writer, cmdGroups []*categoryCommands, showHidden bool) {
	type groupCmdLines struct {
		category string
		cmdLines []usageItem
	}

	sort.Slice(cmdGroups, func(i, j int) bool {
		idx1 := p.app.categoryIdx[cmdGroups[i].category]
		idx2 := p.app.categoryIdx[cmdGroups[j].category]
		if idx1 > 0 && idx2 > 0 {
			return idx1 < idx2
		}
		return idx1 > 0
	})

	var groupLines []*groupCmdLines
	var cmdLines [][]usageItem
	for _, grp := range cmdGroups {
		var grpLines []usageItem
		for _, cmd := range grp.commands {
			cmdName := cmd.Name
			if cmdName == "" || (cmd.Hidden && !showHidden) || cmd.level > 1 {
				continue
			}
			name := "  " + cmdName
			description := cmd.Description
			if cmd.Hidden {
				name += " (HIDDEN)"
			}
			grpLines = append(grpLines, usageItem{
				prefix:      name,
				description: description,
			})
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
	__MaxPrefixLen = 30
	__MinPrefixLen = 6
)

func printWithAlignment(out io.Writer, lines []usageItem, maxPrefixLen int) {
	if maxPrefixLen <= 0 {
		maxPrefixLen = calcMaxPrefixLen([][]usageItem{lines})
	}
	padding := strings.Repeat(" ", maxPrefixLen+4)
	newlineWithPadding := "\n" + padding
	for _, line := range lines {
		x, y := line.prefix, line.description
		fmt.Fprint(out, x)
		if y != "" {
			if len(x) <= maxPrefixLen {
				fmt.Fprint(out, strings.Repeat(" ", maxPrefixLen+4-len(x)))
				fmt.Fprint(out, strings.ReplaceAll(y, "\n", newlineWithPadding))
			} else {
				fmt.Fprint(out, newlineWithPadding)
				fmt.Fprint(out, strings.ReplaceAll(y, "\n", newlineWithPadding))
			}
		}
		fmt.Fprint(out, "\n")
		for _, a := range line.appendixes {
			fmt.Fprintf(out, "%s%s\n", padding, a)
		}
	}
}

func calcMaxPrefixLen(lineGroups [][]usageItem) int {
	maxPrefixLen := 0
	for _, lines := range lineGroups {
		for _, line := range lines {
			if n := len(line.prefix); n > maxPrefixLen && n <= __MaxPrefixLen {
				maxPrefixLen = n
			}
		}
	}
	if maxPrefixLen < __MinPrefixLen {
		maxPrefixLen = __MinPrefixLen
	}
	return maxPrefixLen
}
