# Changelog

Notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.10.0] - 2026-01-28

- New: add new ParseOpt option `WithDefaults` to provide default values programmatically.
- New: add new ParseOpt option `WithEnums` to validate enum options.
- Change: improve help output formatting.

## [v0.9.3] - 2024-01-23

- New: support flag and arg completion for root command (thanks @akemrir)
- Fix: root command help

## [v0.9.2] - 2023-10-18

- New: support reading argument from only environment variables

## [v0.9.0] - 2023-10-10

- New: support pointer flags. #22
- New: support grouping command into different categories in help. #23
- New: support flag value and positional arg completion by function. #27 (thanks @akemrir)
- Fix: typo in README.md. #28 (thanks @maartenverheul)
- Change: use capitalized words instead of all uppercase in usage title. #25

## [v0.8.0] - 2023-08-01

- New: support shell auto-completion. #11
- New: support single quote name in flag usage. #13
- Fix unnecessary space in usage output

## [v0.7.0] - 2023-02-19

- New: support optional app description. #7
- New: support root command. #7
- Change: make app options be public accessible.
- Add more tests, increase coverage to 85%.

## [v0.6.0] - 2022-09-13

- Add coverage report and badge to readme. #2
- Fix suggestion not work in some cases. #3

## [v0.5.0] - 2022-06-22

- New: validate non-flag arguments for invalid usage.
- New: support value implementing encoding.TextUnmarshaler,
  allowing command-line flags and arguments to have types such as big.Int,
  netip.Addr, and time.Time.
- New: add type Context to allow using `func(*Context)` as command action,
  making it easier to use manually created App.
- Change: drop support for Go < 1.17.

## [v0.4.0] - 2022-06-18

- Fix: reflect.Pointer not exists when using with Go below 1.18.
- Fix: error handling for invalid command.
- New: add options `ReplaceUsage` and `WithFooter` to customize usage help.
- New: add option to allow parsing posix-style single token multiple options.
- New: support alias commands.
- Change: remove api `KeepCommandOrder`, replaced by `SetOptions`.
- Change: optimize help padding.
- Change: refactor code for better maintainability.

## [v0.2.1] - 2022-06-11

- Support alternative 'mcli' tag.
- Support global flags.
- Support keep command order in help.
- Improve compatibility with flag.FlagSet.
- Improve examples and docs.

## [v0.1.1] - 2022-03-17

Initial public release.
