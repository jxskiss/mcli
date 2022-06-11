# mcli

[![GoDoc](https://img.shields.io/badge/api-Godoc-blue.svg)][godoc]
[![Go Report Card](https://goreportcard.com/badge/github.com/jxskiss/mcli)][goreport]
[![Issues](https://img.shields.io/github/issues/jxskiss/mcli.svg)][issues]
[![GitHub release](http://img.shields.io/github/release/jxskiss/mcli.svg)][release]
[![MIT License](http://img.shields.io/badge/license-MIT-blue.svg)][license]

[godoc]: https://pkg.go.dev/github.com/jxskiss/mcli
[goreport]: https://goreportcard.com/report/github.com/jxskiss/mcli
[issues]: https://github.com/jxskiss/mcli/issues
[release]: https://github.com/jxskiss/mcli/releases
[license]: https://github.com/jxskiss/mcli/blob/master/LICENSE


`mcli` is a minimal but powerful cli library for Go.
`m` stands for minimal and magic.

It is extremely easy to use, it makes you like writing cli programs in Go.

Disclaimer: the original idea is borrowed from [shafreeck/cortana](https://github.com/shafreeck/cortana),
which is licensed under the Apache License 2.0.

## Features

1. Extremely easy to use, simple but powerful API to define commands, flags and arguments.
2. Add arbitrary level sub-command with single line code. 
3. Define command flags and arguments inside the command processor using struct tag.
4. Define global flags apply to all commands.
5. Read environment variables for flags and arguments.
6. Set default value for flags and arguments.
7. Work with slice, map out of box, of course the bool, string, integer, unsigned integer,
   float, duration types are also supported.
8. Automatic help for commands, flags and arguments.
9. Mark commands, flags as hidden, hidden commands and flags won't be showed in help,
   except that when a special flag "--mcli-show-hidden" is provided.
10. Mark flags, arguments as required, it reports error when not given.
11. Mark flags as deprecated.
12. Automatic suggestions like git.
13. Compatible with the standard library's flag.FlagSet.

## Usage

Use in main function:

```go
func main() {
    var args struct {
        Name string `cli:"-n, --name, Who do you want to say to" default:"tom"`

        // This argument is required.
        Text string `cli:"#R, text, The 'message' you want to send"`
    }
    mcli.Parse(&args)
    fmt.Printf("Say to %s: %s\n", args.Name, args.Text)
}
```

```shell
$ go run say.go
argument is required but not given: text
USAGE:
  say [flags] <text>

FLAGS:
  -n, --name string    Who do you want to say to (default "tom")

ARGUMENTS:
  text message (REQUIRED)    The message you want to send

exit status 2

$ go run say.go hello
Say to tom: hello
```

Use sub-commands:

```go
func main() {
    mcli.Add("cmd1", runCmd1, "An awesome command cmd1")

    mcli.AddGroup("cmd2", "This is a command group called cmd2")
    mcli.Add("cmd2 sub1", runCmd2Sub1, "Do something with cmd2 sub1")
    mcli.Add("cmd2 sub2", runCmd2Sub2, "Brief description about cmd2 sub2")

    // A sub-command can also be registered without registering the group.
    mcli.Add("group3 sub1 subsub1", runGroup3Sub1Subsub1, "Blah blah Blah")

    // This is a hidden command, it won't be showed in help,
    // except that when flag "--mcli-show-hidden" is given.
    mcli.AddHiden("secret-cmd", secretCmd, "An secret command won't be showed in help")

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
        CommonIssueArgs
    }
    mcli.Parse(&args)

    // Do something
}
```

Also, there are some sophisticated examples:

* [github-cli](./examples/github-cli/main.go) mimics Github's cli command `gh`
* [lego](./examples/lego/main.go) mimics Lego's command `lego`

## API

- `Add` adds a command.
- `AddHidden` adds a hidden command.
- `AddGroup` adds a group explicitly. A group is a common prefix for some commands.
  It's not required to add group before adding sub commands, but user can use this function
  to add a description to a group, which will be showed in help.
- `AddHelp` enables the "help" command.
- `Parse` parses the command line for flags and arguments.
- `Run` runs the program, it will parse the command line, search for a registered command and run it.
- `PrintHelp` prints usage doc of the current command to stderr.
- `SetGlobalFlags` sets global flags, global flags are available to all commands.
- `KeepCommandOrder` makes `Parse` to print commands in the order of adding the commands.

### Custom parsing options

- `WithArgs` tells `Parse` to parse from the given args, instead of parsing from the command line arguments.
- `WithErrorHandling` tells `Parse` to use the given ErrorHandling.
  By default, it exits the program when an error happens.
- `WithName` specifies the command name to use when printing usage doc.
- `DisableGlobalFlags` tells `Parse` to don't parse and print global flags in help.

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
 */
CliTag       <-  ( Modifiers ',' Space? )? Name ( ( ',' | Space ) Description )?
Modifiers    <-  '#' [DHR]+
Name         <-  ( ShortName LongName? ) | LongName
Description  <-  ( ![\r\n] . )*

/* env tag, optional.
 * Multiple environment names can be specified, the first non-empty value
 * will be used.
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

* D - marks a flag or argument as deprecated, "DEPRECATED" will be showed in help
* R - marks a flag or argument as required, "REQUIRED" will be showed in help
* H - marks a flag as hidden, see below for more about hidden flags

Hidden flags won't be showed in help, except that when a special flag
"--mcli-show-hidden" is provided.

Modifier `H` shall not be used for an argument, else it panics.
An argument must be showed in help to tell user how to use the program
correctly.

Some modifiers cannot be used together, else it panics, e.g.

* H & R - a required flag must appear in help to tell user to set it
* D & R - a required flag must not be deprecated, it does not make sense,
  but makes user confused

## Compatibility with package `flag`

`Parse` returns a `*flag.FlagSet` if success, all defined flags are available
with the flag set, including both short and long names.

Note that there is a little difference with flag package, while the flag
package requires command line flags must present before arguments, and
arguments can be accessed using flag.Arg(i), this library doesn't require
that, the order that user pass flags and arguments doesn't matter.
Arguments should be defined in the struct given to Parse, command line
arguments will be set to the struct. As a bonus, you can use slice and
map arguments, it just works.

When command line arguments are given before flags, calling FlagSet.Arg(i)
won't get the expected arguments.

## Changelog

### v0.2.0 @ 2022-06-11

New features:

- support alternative 'mcli' tag
- support global flags
- support keep command order in help
- improve examples and docs

### v0.1.1 @ 2022-03-17

Initial public release.
