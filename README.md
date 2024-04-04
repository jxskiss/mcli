# mcli

[![GoDoc](https://img.shields.io/badge/api-Godoc-blue.svg)][godoc]
[![Go Report Card](https://goreportcard.com/badge/github.com/jxskiss/mcli)][goreport]
[![Coverage](https://codecov.io/gh/jxskiss/mcli/branch/main/graph/badge.svg)][codecov]
[![Issues](https://img.shields.io/github/issues/jxskiss/mcli.svg)][issues]
[![GitHub release](http://img.shields.io/github/release/jxskiss/mcli.svg)][release]
[![MIT License](http://img.shields.io/badge/license-MIT-blue.svg)][license]

[godoc]: https://pkg.go.dev/github.com/jxskiss/mcli

[goreport]: https://goreportcard.com/report/github.com/jxskiss/mcli

[codecov]: https://codecov.io/gh/jxskiss/mcli

[issues]: https://github.com/jxskiss/mcli/issues

[release]: https://github.com/jxskiss/mcli/releases

[license]: https://github.com/jxskiss/mcli/blob/master/LICENSE


`mcli` is a minimal but powerful cli library for Go.
`m` stands for minimal and magic.

It is extremely easy to use, it makes you love writing cli programs in Go.

Disclaimer: the original idea is inspired by [shafreeck/cortana](https://github.com/shafreeck/cortana),
which is licensed under the Apache License 2.0.

## Features

* Easy to use, dead simple yet very powerful API to define commands, flags and arguments.
* Add arbitrary nested sub-command with single line code.
* Group subcommands into different categories in help.
* Define command flags and arguments inside the command processor using struct tag.
* Define global flags apply to all commands, or share common flags between a group of commands.
* Read environment variables for flags and arguments.
* Set default value for flags and arguments.
* Work with time.Duration, slice, map out of box.
* Mark commands, flags as hidden, hidden commands and flags won't be showed in help,
  except that when a special flag `--mcli-show-hidden` is provided.
* Mark flags, arguments as required, report error when a required flag is not given.
* Mark flags as deprecated.
* Automatic suggestions like git.
* Automatic help generation for commands, flags and arguments.
* Automatic help flag recognition of `-h`, `--help`, etc.
* Automatic shell completion, it supports `bash`, `zsh`, `fish`, `powershell` for now.
* Compatible with the standard library's flag.FlagSet.
* Optional posix-style single token multiple options command line parsing.
* Alias command, so you can reorganize commands without breaking them.
* Flexibility to define your own usage messages.
* Minimal dependency.
* Makes you love writing cli programs in Go.

## Usage

Use in main function:

```go
func main() {
    var args struct {
        Name string `cli:"-n, --name, Who do you want to say to" default:"tom"`

        // This argument is required.
        Text string `cli:"#R, text, The 'message' you want to send"`

        // This argument reads environment variable and requires the variable must exist,
        // it doesn't accept input from command line.
        APIAccessKey string `cli:"#ER, The access key to your service provider" env:"MY_API_ACCESS_KEY"`
    }
    mcli.Parse(&args)
    fmt.Printf("Say to %s: %s\n", args.Name, args.Text)
}
```

```shell
$ go run say.go -h
Usage:
  say [flags] <text>

Flags:
  -n, --name <string>    Who do you want to say to
                         [default: "tom"]

Arguments:
  text <message> [REQUIRED]    The message you want to send

Environment Variables:
  - MY_API_ACCESS_KEY <string> [REQUIRED]
    The access key to your service provider

$ MY_API_ACCESS_KEY=xxxx go run say.go hello
Say to tom: hello
```

Use sub-commands:

```go
func main() {
    mcli.Add("cmd1", runCmd1, "An awesome command cmd1")

    mcli.AddGroup("cmd2", "This is a command group called cmd2")
    mcli.Add("cmd2 sub1", runCmd2Sub1, "Do something with cmd2 sub1")
    mcli.Add("cmd2 sub2", runCmd2Sub2, "Brief description about cmd2 sub2")

    // A sub-command can also be added without registering the group.
    mcli.Add("group3 sub1 subsub1", runGroup3Sub1Subsub1, "Blah blah Blah")

    // This is a hidden command, it won't be showed in help,
    // except that when flag "--mcli-show-hidden" is given.
    mcli.AddHidden("secret-cmd", secretCmd, "An secret command won't be showed in help")

    // Enable shell auto-completion, see `program completion -h` for help.
    mcli.AddCompletion()

    mcli.Run()
}

func runCmd1() {
    var args struct {
        Branch    string `cli:"-b, --branch, Select another branch by passing in the branch name"`
        Commit    bool   `cli:"-c, --commit, Open the last commit"`
        NoBrowser bool   `cli:"-n, --no-browser, Print destination URL instead of opening the browser"`
        Projects  bool   `cli:"-p, --projects, Open repository projects"`
        Repo      string `cli:"-R, --repo, Select another repository using the '[HOST/]OWNER/REPO' format"`
        Settings  bool   `cli:"-s, --settings, Open repository settings"`
        Wiki      bool   `cli:"-w, --wiki, Open repository wiki"`

        Location  string `cli:"location, A browser location can be specified using arguments in the following format:\n- by number for issue or pull request, e.g. \"123\"; or\n- by path for opening folders and files, e.g. \"cmd/gh/main.go\""`
    }
    mcli.Parse(&args)

    // Do something
}

type Cmd2CommonArgs struct {
    Repo string `cli:"-R, --repo, Select another repository using the '[HOST/]OWNER/REPO' format"`
}

func runCmd2Sub1() {
    // Note that the flag/argument description can be seperated either
    // by a comma or spaces, and can be mixed.
    var args struct {
        Body     string `cli:"-b, --body        Supply a body. Will prompt for one otherwise."`
        BodyFile string `cli:"-F, --body-file   Read body text from 'file' (use \"-\" to read from standard input)"`
        Editor   bool   `cli:"-e, --editor,     Add body using editor"`
        Web      bool   `cli:"-w, --web,        Add body in browser"`

        // Can embed other structs.
        Cmd2CommonArgs
    }
    mcli.Parse(&args)

    // Do something
}
```

Also, there are some sophisticated examples:

* [github-cli](./examples/github-cli/main.go) mimics Github's cli command `gh`
* [lego](./examples/lego/main.go) mimics Lego's command `lego`

## API

Use the default App:

- `SetOptions` updates options of the default application.
- `SetGlobalFlags` sets global flags, global flags are available to all commands.
- `Add` adds a command.
- `AddRoot` adds a root command. A root command is executed when no sub command is specified.
- `AddAlias` adds an alias name for a command.
- `AddHidden` adds a hidden command.
- `AddGroup` adds a group explicitly. A group is a common prefix for some commands.
  It's not required to add group before adding sub commands, but user can use this function
  to add a description to a group, which will be showed in help.
- `AddHelp` enables the "help" command.
- `AddCompletion` enables the "completion" command to generate autocomplete scripts.
- `Parse` parses the command line for flags and arguments.
- `Run` runs the program, it will parse the command line, search for a registered command and run it.
- `PrintHelp` prints usage doc of the current command to stderr.

Create a new App instance:

- `NewApp` creates a new cli applcation instance.

### Custom options

App:

- `App.Options` specifies optional options for an application.

CmdOpt:

- `WithCategory` groups commands into different categories in help.
- `WithLongDesc` specifies a long description of a command, which will be showed in the command's help.
- `EnableFlagCompletion` enables flag completion for a command.

ParseOpt:

- `WithArgs` tells `Parse` to parse from the given args, instead of parsing from the command line arguments.
- `WithErrorHandling` tells `Parse` to use the given ErrorHandling.
  By default, the program exits when an error happens.
- `WithName` specifies the command name to use when printing usage doc.
- `DisableGlobalFlags` tells `Parse` to don't parse and print global flags in help.
- `ReplaceUsage` tells `Parse` to use a custom usage function instead of the default.
- `WithExamples` specifies examples for a command. Examples will be showed after flags in the help.
- `WithFooter` adds a footer message after the default help,
  this option overrides the App's setting `Options.HelpFooter` for this parsing call.
- `WithArgCompFuncs` specifies functions to suggest flag values and positional arguments programmatically.

## Tag syntax

Struct tag is a powerful feature in Go, `mcli` uses struct tag to define flags and arguments.

* tag `cli` defines the name and description for flags and arguments
* tag `env` optionally tells Parse to lookup environment variables when user doesn't
  provide a value on the command line
* tag `default` optionally provides a default value to a flag or argument,
  which will be used when the value is not available from both command line and env

The syntax is

```text
/* cli tag, only Name is required.
 * Short name and long name are both optional, but at least one must be given.
 * See below for details about modifiers.
 * e.g.
 * - `cli:"-c, Open the last commit"`
 * - `cli:"#R, -b, --branch, Select another branch by passing in the branch name"`
 * - `cli:"--an-obvious-flag-dont-need-description"`
 * - `cli:"#ER, AWS Secret Access Key" env:"AWS_SECRET_ACCESS_KEY"`
 */
CliTag       <-  ( Modifiers ',' Space? )? Name ( ( ',' | Space ) Description )?
Modifiers    <-  '#' [DHRE]+
Name         <-  ( ShortName LongName? ) | LongName
Description  <-  ( ![\r\n] . )*

/* env tag, optional.
 * Multiple environment names can be specified, the first non-empty value takes effect.
 * e.g.
 * - `env:"SOME_ENV"`
 * - `env:"ANOTHER_ENV_1, ANOTHER_ENV_2"`
 */
EnvTag  <-  ( EnvName ',' Space? )* EnvName

/* default value tag, optional.
 * e.g.
 * - `default:"1.5s"` // duration
 * - `default:"true"` // bool
 */
DefaultValueTag  <-  ( ![\r\n] . )*
```

## Modifiers

Modifier represents an option to a flag, it sets the flag to be
deprecated, hidden, or required. In a `cli` tag, modifiers appears as
the first segment, starting with a `#` character.

Fow now the following modifiers are available:

* D - marks a flag or argument as deprecated, "DEPRECATED" will be showed in help.
* R - marks a flag or argument as required, "REQUIRED" will be showed in help.
* H - marks a flag as hidden, see below for more about hidden flags.
* E - marks an argument read from environment variables, but not command line,
      environment variables will be showed in a separate section in help.

Hidden flags won't be showed in help, except that when a special flag
"--mcli-show-hidden" is provided.

Modifier `H` shall not be used for an argument, else it panics.
An argument must be showed in help to tell user how to use the program
correctly.

Modifier `E` is useful when you want to read an environment variable,
but don't want user to provide from command line (e.g. password or other secrets).
Using together with `R` also ensures that the env variable must exist.

Some modifiers cannot be used together, else it panics, e.g.

* H & R - a required flag must appear in help to tell user to set it.
* D & R - a required flag must not be deprecated, it does not make sense,
  but makes user confused.

## Compatibility with package `flag`

`Parse` returns a `*flag.FlagSet` if success, all defined flags are available
with the flag set, including both short and long names.

Note that the package `flag` requires command line flags must present before
arguments, this package does not have this requirement.
Positional arguments can present either before flags or after flags,
even both before and after flags, in which case, the args will be reordered
and all arguments can be accessed by calling flagSet.Args() and flagSet.Arg(i).

If there is slice or map arguments, it will match all following arguments.

## Shell completion

`mcli` supports auto shell completion for `bash`, `zsh`, `fish`, and `powershell`.
Use `AddCompletion` to enable the feature, run `program help completion [bash|zsh|fish|powershell]`
for usage guide.

Also check `AddCompletion`, `EnableFlagCompletion`, and
`Options.EnableFlagCompletionForAllCommands` for detail docs about command flag completion.

User can use `WithArgCompFuncs` to specify functions to suggest flag values and
positional arguments programmatically, already provided flags and arguments
can be accessed in the functions.

## Changelog

See [CHANGELOG](./CHANGELOG.md) for detailed change history.
