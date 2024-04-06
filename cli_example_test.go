package mcli

import (
	"bytes"
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func mockOSArgs(mockArgs ...string) func() {
	old := os.Args
	os.Args = mockArgs
	return func() {
		os.Args = old
	}
}

type CommonIssueArgs struct {
	Repo string `cli:"-R, --repo, Select another repository using the '[HOST/]OWNER/REPO' format"`
}

func addTestGithubCliCommands() *bytes.Buffer {
	defaultApp.Options.HelpFooter = `
LEARN MORE
  Use 'gh <command> <subcommand> --help' for more information about a command.
  Read the manual at https://cli.github.com/manual
`

	AddGroup("auth", "Login, logout, and refresh your authentication",
		WithCategory("Core Commands"))
	Add("auth login", dummyCmd, "Authenticate with a GitHub host")
	Add("auth logout", dummyCmd, "Log out of a GitHub host")
	Add("auth refresh", dummyCmd, "Refresh stored authentication credentials")
	Add("auth setup-git", dummyCmd, "Configure git to use GitHub CLI as a credential helper")
	Add("auth status", dummyCmd, "View authentication status")

	Add("browse", githubCliBrowseCmd, "Open the repository in the browser",
		WithCategory("Core Commands"))

	AddGroup("issue", "Manage issues",
		WithCategory("Core Commands"))
	Add("issue close", dummyCmd, "Close issue")
	Add("issue comment", dummyCmd, "Create a new issue comment")
	Add("issue create", exampleGithubCliIssueCreate, "Create a new issue")
	Add("issue delete", dummyCmd, "Delete issue")
	Add("issue edit", dummyCmd, "Edit an issue")
	Add("issue list", dummyCmd, "List and filter issues in this repository")
	Add("issue reopen", dummyCmd, "Reopen issue")
	Add("issue status", dummyCmd, "Show status of relevant issues")
	Add("issue transfer", dummyCmd, "Transfer issue to another repository")
	Add("issue view", dummyCmd, "View an issue")

	AddGroup("codespace", "Connect to and manage your codespaces",
		WithCategory("Core Commands"))
	Add("codespace code", dummyCmd, "Open a codespace in Visual Studio Code")
	Add("codespace cp", dummyCmd, "Copy files between local and remote file systems")
	Add("codespace create", dummyCmd, "Create a codespace")
	Add("codespace delete", dummyCmd, "Delete a codespace")
	Add("codespace list", dummyCmd, "List your codespaces")
	Add("codespace logs", dummyCmd, "Access codespace logs")
	Add("codespace ports", dummyCmd, "List ports in a codespace")
	Add("codespace ssh", dummyCmd, "SSH into a codespace")
	Add("codespace stop", dummyCmd, "Stop a running codespace")

	AddGroup("gist", "Manage gists",
		WithCategory("Core Commands"))
	Add("gist clone", dummyCmd, "Clone a gist locally")
	Add("gist create", dummyCmd, "Create a new gist")
	Add("gist delete", dummyCmd, "Delete a gist")
	Add("gist edit", dummyCmd, "Edit one of your gists")
	Add("gist list", dummyCmd, "List your gists")
	Add("gist view", dummyCmd, "View a gist")

	AddGroup("pr", "Manage pull requests",
		WithCategory("Core Commands"))
	Add("pr checkout", dummyCmd, "Check out a pull request in git")
	Add("pr checks", dummyCmd, "Show CI status for a single pull request")
	Add("pr close", dummyCmd, "Close a pull request")
	Add("pr comment", dummyCmd, "Create a new pr comment")
	Add("pr create", dummyCmd, "Create a pull request")
	Add("pr diff", dummyCmd, "View changes in a pull request")
	Add("pr edit", dummyCmd, "Edit a pull request")
	Add("pr list", dummyCmd, "List and filter pull requests in this repository")
	Add("pr merge", dummyCmd, "Merge a pull request")
	Add("pr ready", dummyCmd, "Mark a pull request as ready for review")
	Add("pr reopen", dummyCmd, "Reopen a pull request")
	Add("pr review", dummyCmd, "Add a review to a pull request")
	Add("pr status", dummyCmd, "Show status of relevant pull requests")
	Add("pr view", dummyCmd, "View a pull request")

	AddGroup("release", "Manage GitHub releases",
		WithCategory("Core Commands"))
	Add("release create", dummyCmd, "Create a new release")
	Add("release delete", dummyCmd, "Delete a release")
	Add("release download", dummyCmd, "Download release assets")
	Add("release list", dummyCmd, "List releases in a repository")
	Add("release upload", dummyCmd, "Upload assets to a release")
	Add("release view", dummyCmd, "View information about a release")

	AddGroup("repo", "Create, clone, fork, and view repositories",
		WithCategory("Core Commands"))
	Add("repo archive", dummyCmd, "Archive a repository")
	Add("repo clone", dummyCmd, "Clone a repository locally")
	Add("repo create", dummyCmd, "Create a new repository")
	Add("repo delete", dummyCmd, "Delete a repository")
	Add("repo edit", dummyCmd, "Edit repository settings")
	Add("repo fork", dummyCmd, "Create a fork of a repository")
	Add("repo list", dummyCmd, "List repositories owned by user or organization")
	Add("repo rename", dummyCmd, "Rename a repository")
	Add("repo sync", dummyCmd, "Sync a repository")
	Add("repo view", dummyCmd, "View a repository")

	AddGroup("run", "View details about workflow runs",
		WithCategory("GitHub Actions Commands"))
	Add("run cancel", dummyCmd, "Cancel a workflow run")
	Add("run download", dummyCmd, "Download artifacts generated by a workflow run")
	Add("run list", dummyCmd, "List recent workflow runs")
	Add("run rerun", dummyCmd, "Rerun a failed run")
	Add("run view", dummyCmd, "View a summary of a workflow run")
	Add("run watch", dummyCmd, "Watch a run until it completes, showing its progress")

	AddGroup("workflow", "View details about GitHub Actions workflows",
		WithCategory("Additional Commands"))
	Add("workflow disable", dummyCmd, "Disable a workflow")
	Add("workflow enable", dummyCmd, "Enable a workflow")
	Add("workflow list", dummyCmd, "List workflows")
	Add("workflow run", dummyCmd, "Run a workflow by creating a workflow_dispatch event")
	Add("workflow view", dummyCmd, "View the summary of a workflow")

	AddGroup("alias", "Create command shortcuts",
		WithCategory("Additional Commands"))
	Add("alias delete", dummyCmd, "Delete an alias")
	Add("alias list", dummyCmd, "List your aliases")
	Add("alias set", dummyCmd, "Create a shortcut for a gh command")

	Add("api", dummyCmd, "Make an authenticated GitHub API request",
		WithCategory("Additional Commands"))

	AddGroup("config", "Manage configuration for gh")
	Add("config get", dummyCmd, "Print the value of a given configuration key")
	Add("config list", dummyCmd, "Print a list of configuration keys and values")
	Add("config set", dummyCmd, "Update configuration with a value for the given key")

	AddGroup("extension", "Manage gh extensions")
	Add("extension create", dummyCmd, "Create a new extension")
	Add("extension install", dummyCmd, "Install a gh extension from a repository")
	Add("extension list", dummyCmd, "List installed extension commands")
	Add("extension remove", dummyCmd, "Remove an installed extension")
	Add("extension upgrade", dummyCmd, "Upgrade installed extensions")

	AddGroup("gpg-key", "Manage GPG keys")
	Add("gpg-key add", dummyCmd, "Add a GPG key to your GitHub account")
	Add("gpg-key list", dummyCmd, "Lists GPG keys in your GitHub account")

	AddGroup("secret", "Manage GitHub secrets")
	Add("secret list", dummyCmd, "List secrets")
	Add("secret remove", dummyCmd, "Remove secrets")
	Add("secret set", dummyCmd, "Create or update secrets")

	AddGroup("ssh-key", "Manage SSH keys")
	Add("ssh-key add", dummyCmd, "Add an SSH key to your GitHub account")
	Add("ssh-key list", dummyCmd, "Lists SSH keys in your GitHub account")

	Add("actions", dummyCmd, "Learn about working with GitHub Actions",
		WithCategory("Help Topics"))

	// Enable the "help" command.
	AddHelp()
	AddCompletionWithName("completion")

	var buf bytes.Buffer
	defaultApp.getFlagSet().Init("", flag.ContinueOnError)
	defaultApp.getFlagSet().SetOutput(&buf)
	return &buf
}

func Test_githubCli_mainHelp(t *testing.T) {
	resetDefaultApp()
	buf := addTestGithubCliCommands()
	defer mockOSArgs("gh", "-h")()
	Run()

	want := strings.TrimSpace(`
Usage:
  gh <command> ...

Core Commands:
  auth          Login, logout, and refresh your authentication
  browse        Open the repository in the browser
  codespace     Connect to and manage your codespaces
  gist          Manage gists
  issue         Manage issues
  pr            Manage pull requests
  release       Manage GitHub releases
  repo          Create, clone, fork, and view repositories

GitHub Actions Commands:
  run           View details about workflow runs

Additional Commands:
  alias         Create command shortcuts
  api           Make an authenticated GitHub API request
  workflow      View details about GitHub Actions workflows

Help Topics:
  actions       Learn about working with GitHub Actions

Other Commands:
  completion    Generate shell completion scripts
  config        Manage configuration for gh
  extension     Manage gh extensions
  gpg-key       Manage GPG keys
  help          Help about any command
  secret        Manage GitHub secrets
  ssh-key       Manage SSH keys

LEARN MORE
  Use 'gh <command> <subcommand> --help' for more information about a command.
  Read the manual at https://cli.github.com/manual
`)
	assert.Equal(t, want, strings.TrimSpace(buf.String()))
}

func Test_githubCli_mainHelp_keepCmdOrder(t *testing.T) {
	resetDefaultApp()
	defaultApp.Options.KeepCommandOrder = true
	buf := addTestGithubCliCommands()
	defer mockOSArgs("gh", "-h")()
	Run()

	want := strings.TrimSpace(`
Usage:
  gh <command> ...

Core Commands:
  auth          Login, logout, and refresh your authentication
  browse        Open the repository in the browser
  issue         Manage issues
  codespace     Connect to and manage your codespaces
  gist          Manage gists
  pr            Manage pull requests
  release       Manage GitHub releases
  repo          Create, clone, fork, and view repositories

GitHub Actions Commands:
  run           View details about workflow runs

Additional Commands:
  workflow      View details about GitHub Actions workflows
  alias         Create command shortcuts
  api           Make an authenticated GitHub API request

Help Topics:
  actions       Learn about working with GitHub Actions

Other Commands:
  config        Manage configuration for gh
  extension     Manage gh extensions
  gpg-key       Manage GPG keys
  secret        Manage GitHub secrets
  ssh-key       Manage SSH keys
  help          Help about any command
  completion    Generate shell completion scripts

LEARN MORE
  Use 'gh <command> <subcommand> --help' for more information about a command.
  Read the manual at https://cli.github.com/manual
`)
	assert.Equal(t, want, strings.TrimSpace(buf.String()))
}

func Test_githubCli_issueHelp(t *testing.T) {
	resetDefaultApp()
	buf := addTestGithubCliCommands()
	defer mockOSArgs("gh", "issue")()
	Run()

	want := strings.TrimSpace(`
Manage issues

Usage:
  gh issue <command> ...

Commands:
  close       Close issue
  comment     Create a new issue comment
  create      Create a new issue
  delete      Delete issue
  edit        Edit an issue
  list        List and filter issues in this repository
  reopen      Reopen issue
  status      Show status of relevant issues
  transfer    Transfer issue to another repository
  view        View an issue

LEARN MORE
  Use 'gh <command> <subcommand> --help' for more information about a command.
  Read the manual at https://cli.github.com/manual
`)
	assert.Equal(t, want, strings.TrimSpace(buf.String()))
}

/*
Open the GitHub repository in the web browser.

USAGE
  gh browse [<number> | <path>] [flags]

FLAGS
  -b, --branch string            Select another branch by passing in the branch name
  -c, --commit                   Open the last commit
  -n, --no-browser               Print destination URL instead of opening the browser
  -p, --projects                 Open repository projects
  -R, --repo [HOST/]OWNER/REPO   Select another repository using the [HOST/]OWNER/REPO format
  -s, --settings                 Open repository settings
  -w, --wiki                     Open repository wiki

INHERITED FLAGS
  --help   Show help for command

ARGUMENTS
  A browser location can be specified using arguments in the following format:
  - by number for issue or pull request, e.g. "123"; or
  - by path for opening folders and files, e.g. "cmd/gh/main.go"

EXAMPLES
  $ gh browse
  #=> Open the home page of the current repository

  $ gh browse 217
  #=> Open issue or pull request 217

  $ gh browse --settings
  #=> Open repository settings

  $ gh browse main.go:312
  #=> Open main.go at line 312

  $ gh browse main.go --branch main
  #=> Open main.go in the main branch

ENVIRONMENT VARIABLES
  To configure a web browser other than the default, use the BROWSER environment variable.

LEARN MORE
  Use 'gh <command> <subcommand> --help' for more information about a command.
  Read the manual at https://cli.github.com/manual

*/

type githubCliBrowseArgs struct {
	Branch    string `cli:"-b, --branch, Select another branch by passing in the branch name"`
	Commit    bool   `cli:"-c, --commit, Open the last commit"`
	NoBrowser bool   `cli:"-n, --no-browser, Print destination URL instead of opening the browser"`
	Projects  bool   `cli:"-p, --projects, Open repository projects"`
	Repo      string `cli:"-R, --repo, Select another repository using the '[HOST/]OWNER/REPO' format"`
	Settings  bool   `cli:"-s, --settings, Open repository settings"`
	Wiki      bool   `cli:"-w, --wiki, Open repository wiki"`

	Location string `cli:"location, A browser location can be specified using arguments in the following format:\n- by number for issue or pull request, e.g. \"123\"; or\n- by path for opening folders and files, e.g. \"cmd/gh/main.go\""`
}

var githubCliBrowseCmd = NewCommand(exampleGithubCliBrowse,
	WithErrorHandling(flag.ContinueOnError),
	WithExamples(`
		$ gh browse
		#=> Open the home page of the current repository

		$ gh browse 217
		#=> Open issue or pull request 217

		$ gh browse --settings
		#=> Open repository settings

		$ gh browse main.go:312
		#=> Open main.go at line 312

		$ gh browse main.go --branch main
		#=> Open main.go in the main branch`))

func exampleGithubCliBrowse(ctx *Context, args *githubCliBrowseArgs) {
	if err := ctx.ArgsError(); err != nil && err != flag.ErrHelp {
		panic(err)
	}
}

func Test_githubCli_browseHelp(t *testing.T) {
	resetDefaultApp()
	buf := addTestGithubCliCommands()
	defer mockOSArgs("gh", "browse", "-h")()
	Run()

	want := strings.TrimSpace(`
Open the repository in the browser

Usage:
  gh browse [flags] [location]

Flags:
  -b, --branch <string>    Select another branch by passing in the branch name
  -c, --commit             Open the last commit
  -n, --no-browser         Print destination URL instead of opening the browser
  -p, --projects           Open repository projects
  -R, --repo <[HOST/]OWNER/REPO>
                           Select another repository using the [HOST/]OWNER/REPO format
  -s, --settings           Open repository settings
  -w, --wiki               Open repository wiki

Arguments:
  location <string>    A browser location can be specified using arguments in the following format:
                       - by number for issue or pull request, e.g. "123"; or
                       - by path for opening folders and files, e.g. "cmd/gh/main.go"

Examples:
  $ gh browse
  #=> Open the home page of the current repository

  $ gh browse 217
  #=> Open issue or pull request 217

  $ gh browse --settings
  #=> Open repository settings

  $ gh browse main.go:312
  #=> Open main.go at line 312

  $ gh browse main.go --branch main
  #=> Open main.go in the main branch

LEARN MORE
  Use 'gh <command> <subcommand> --help' for more information about a command.
  Read the manual at https://cli.github.com/manual
`)
	assert.Equal(t, want, strings.TrimSpace(buf.String()))
}

/*
Create a new issue

USAGE
  gh issue create [flags]

FLAGS
  -a, --assignee login   Assign people by their login. Use "@me" to self-assign.
  -b, --body string      Supply a body. Will prompt for one otherwise.
  -F, --body-file file   Read body text from file (use "-" to read from standard input)
  -l, --label name       Add labels by name
  -m, --milestone name   Add the issue to a milestone by name
  -p, --project name     Add the issue to projects by name
      --recover string   Recover input from a failed run of create
  -t, --title string     Supply a title. Will prompt for one otherwise.
  -w, --web              Open the browser to create an issue

INHERITED FLAGS
      --help                     Show help for command
  -R, --repo [HOST/]OWNER/REPO   Select another repository using the [HOST/]OWNER/REPO format

EXAMPLES
  $ gh issue create --title "I found a bug" --body "Nothing works"
  $ gh issue create --label "bug,help wanted"
  $ gh issue create --label bug --label "help wanted"
  $ gh issue create --assignee monalisa,hubot
  $ gh issue create --assignee "@me"
  $ gh issue create --project "Roadmap"

LEARN MORE
  Use 'gh <command> <subcommand> --help' for more information about a command.
  Read the manual at https://cli.github.com/manual

*/

func exampleGithubCliIssueCreate() {
	var args struct {
		Assignee  string `cli:"-a, --assignee    Assign people by their 'login'. Use \"@me\" to self-assign."`
		Body      string `cli:"-b, --body        Supply a body. Will prompt for one otherwise."`
		BodyFile  string `cli:"-F, --body-file   Read body text from 'file' (use \"-\" to read from standard input)"`
		Label     string `cli:"-l, --label       Add labels by 'name'"`
		Milestone string `cli:"-m, --milestone   Add the issue to a milestone by 'name'"`
		Project   string `cli:"-p, --project     Add the issue to projects by 'name'"`
		Recover   string `cli:"    --recover     Recover input from a failed run of create"`
		Title     string `cli:"-t, --title       Supply a title. Will prompt for one otherwise."`
		Web       bool   `cli:"-w, --web         Open the browser to create an issue"`
		CommonIssueArgs
	}
	_, err := Parse(&args, WithErrorHandling(flag.ContinueOnError),
		WithExamples(`
  $ gh issue create --title "I found a bug" --body "Nothing works"
  $ gh issue create --label "bug,help wanted"
  $ gh issue create --label bug --label "help wanted"
  $ gh issue create --assignee monalisa,hubot
  $ gh issue create --assignee "@me"
  $ gh issue create --project "Roadmap"
`))
	if err != nil && err != flag.ErrHelp {
		panic(err)
	}
}

var exampleGithubCliIssueCreateHelp = strings.TrimSpace(`
Create a new issue

Usage:
  gh issue create [flags]

Flags:
  -a, --assignee <login>    Assign people by their login. Use "@me" to self-assign.
  -b, --body <string>       Supply a body. Will prompt for one otherwise.
  -F, --body-file <file>    Read body text from file (use "-" to read from standard input)
  -l, --label <name>        Add labels by name
  -m, --milestone <name>    Add the issue to a milestone by name
  -p, --project <name>      Add the issue to projects by name
      --recover <string>    Recover input from a failed run of create
  -R, --repo <[HOST/]OWNER/REPO>
                            Select another repository using the [HOST/]OWNER/REPO format
  -t, --title <string>      Supply a title. Will prompt for one otherwise.
  -w, --web                 Open the browser to create an issue

Examples:
  $ gh issue create --title "I found a bug" --body "Nothing works"
  $ gh issue create --label "bug,help wanted"
  $ gh issue create --label bug --label "help wanted"
  $ gh issue create --assignee monalisa,hubot
  $ gh issue create --assignee "@me"
  $ gh issue create --project "Roadmap"

LEARN MORE
  Use 'gh <command> <subcommand> --help' for more information about a command.
  Read the manual at https://cli.github.com/manual
`)

func Test_githubCli_issueCreate_helpFlag(t *testing.T) {
	resetDefaultApp()
	buf := addTestGithubCliCommands()
	defer mockOSArgs("gh", "issue", "create", "-h")()
	Run()

	want := exampleGithubCliIssueCreateHelp
	assert.Equal(t, want, strings.TrimSpace(buf.String()))
}

func Test_githubCli_issueCreate_helpCommand(t *testing.T) {
	resetDefaultApp()
	buf := addTestGithubCliCommands()
	defer mockOSArgs("gh", "help", "issue", "create")()
	Run()

	want := exampleGithubCliIssueCreateHelp
	assert.Equal(t, want, strings.TrimSpace(buf.String()))
}
