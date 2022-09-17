package mcli

import (
	"fmt"
	"io"
	"reflect"
	"strings"
)

func newUsagePrinter(app *App) *usagePrinter {
	ctx := app.getParsingContext()
	out := getFlagSetOutput(ctx.getFlagSet())
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
}

func (p *usagePrinter) Do() {
	ctx := p.ctx
	out := p.out
	if ctx.opts.customUsage != nil {
		help := strings.TrimSpace(ctx.opts.customUsage())
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
	p.splitFlags()
	p.printCmdFlags()
	p.printArguments()
	p.printGlobalFlags()
	p.printFooter()
}

func (p *usagePrinter) printUsageLine() {
	usage := ""
	ctx := p.ctx
	out := p.out
	cmd := ctx.cmd
	cmdName := ctx.name
	progName := getProgramName()
	if cmd != nil {
		if cmd.AliasOf != "" {
			usage += cmd.Description + "\n"
			cmd = p.app.cmdMap[cmd.AliasOf]
			cmdName = cmd.Name
		}
		if cmd.Description != "" {
			usage += cmd.Description + "\n"
		}
		if usage != "" {
			usage += "\n"
		}
	}
	usage += "USAGE:\n  " + progName
	if cmdName != "" {
		usage += " " + cmdName
	}

	hasFlags := len(ctx.flags) > 0
	hasNonflags := len(ctx.nonflags) > 0
	hasSubCmds := len(p.subCmds) > 0
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
	fmt.Fprint(out, usage, "\n\n")
}

func (p *usagePrinter) printSubCommands() {
	if len(p.subCmds) > 0 {
		ctx := p.ctx
		out := p.out
		subCmds := p.subCmds
		showHidden := ctx.showHidden
		keepCmdOrder := p.app.opts.KeepCommandOrder
		printSubCommands(out, subCmds, showHidden, keepCmdOrder)
		fmt.Fprint(out, "\n")
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

func (p *usagePrinter) splitFlags() {
	flags := p.ctx.flags
	showHidden := p.ctx.showHidden
	hasShortFlag := p.hasShortFlag

	var globalFlagHelp [][2]string
	var cmdFlagHelp [][2]string
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
	p.globalFlagHelp = globalFlagHelp
	p.cmdFlagHelp = cmdFlagHelp
}

func (p *usagePrinter) printCmdFlags() {
	if len(p.cmdFlagHelp) > 0 {
		out := p.out
		fmt.Fprint(out, "FLAGS:\n")
		printWithAlignment(out, p.cmdFlagHelp)
		fmt.Fprint(out, "\n")
	}
}

func (p *usagePrinter) printArguments() {
	out := p.out
	nonflags := p.ctx.nonflags
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
}

func (p *usagePrinter) printGlobalFlags() {
	if len(p.globalFlagHelp) > 0 {
		out := p.out
		fmt.Fprint(out, "GLOBAL FLAGS:\n")
		printWithAlignment(out, p.globalFlagHelp)
		fmt.Fprint(out, "\n")
	}
}

func (p *usagePrinter) printFooter() {
	ctx := p.ctx
	out := p.out
	if ctx.opts.helpFooter != nil {
		footer := strings.TrimSpace(ctx.opts.helpFooter())
		fmt.Fprintf(out, "%s\n\n", footer)
	}
}
