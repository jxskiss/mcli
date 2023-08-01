package mcli

import (
	"strings"
	"text/template"
)

const bashCompletionUsage = `
Generate the autocompletion script for the bash shell.

The script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

	PROG={{ .ProgramName }}; source <({{ .ProgramName }} {{ .CompletionCmdName }} bash)

To load completions for every new session, execute once:

#### Linux:

	PROG={{ .ProgramName }}; {{ .ProgramName }} {{ .CompletionCmdName }} bash > /etc/bash_completion.d/{{ .ProgramName }}

#### macOS:

	PROG={{ .ProgramName }}; {{ .ProgramName }} {{ .CompletionCmdName }} bash > $(brew --prefix)/etc/bash_completion.d/{{ .ProgramName }}

You will need to start a new shell for this setup to take effect.

USAGE:
  {{ .ProgramName }} {{ .CompletionCmdName }} bash
`

const zshCompletionUsage = `
Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

	echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions in your current shell session:

	PROG={{ .ProgramName }}; source <({{ .ProgramName }} {{ .CompletionCmdName }} zsh)

To load completions for every new session, execute once:

#### Linux:

	PROG={{ .ProgramName }}; {{ .ProgramName }} {{ .CompletionCmdName }} zsh > "${fpath[1]}/_{{ .ProgramName }}"

#### macOS:

	PROG={{ .ProgramName }}; {{ .ProgramName }} {{ .CompletionCmdName }} zsh > $(brew --prefix)/share/zsh/site-functions/_{{ .ProgramName }}

You will need to start a new shell for this setup to take effect.

USAGE:
  {{ .ProgramName }} {{ .CompletionCmdName }} zsh
`

const powershellCompletionUsage = `
Generate the autocompletion script for powershell.

To load completions in your current shell session:

	{{ .ProgramName }} {{ .CompletionCmdName }} powershell | Out-String | Invoke-Expression

To load completions for every new session, add the output of the above command
to your powershell profile.

USAGE:
  {{ .ProgramName }} {{ .CompletionCmdName }} powershell
`

func (p *App) completionUsage(shellType string) func() string {
	return func() string {
		data := map[string]any{
			"ProgramName":       getProgramName(),
			"CompletionCmdName": p.completionCmdName,
		}
		var tplContent string
		switch shellType {
		case "bash":
			tplContent = bashCompletionUsage
		case "zsh":
			tplContent = zshCompletionUsage
		case "powershell":
			tplContent = powershellCompletionUsage
		}
		tpl := template.Must(template.New("").Parse(tplContent))
		builder := &strings.Builder{}
		err := tpl.Execute(builder, data)
		if err != nil {
			panic(bugReportTip)
		}
		return builder.String()
	}
}
