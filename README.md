# goversion

[![checks](https://github.com/go-simpler/goversion/actions/workflows/checks.yml/badge.svg)](https://github.com/go-simpler/goversion/actions/workflows/checks.yml)
[![pkg.go.dev](https://pkg.go.dev/badge/go-simpler.org/goversion.svg)](https://pkg.go.dev/go-simpler.org/goversion)
[![goreportcard](https://goreportcard.com/badge/go-simpler.org/goversion)](https://goreportcard.com/report/go-simpler.org/goversion)
[![codecov](https://codecov.io/gh/go-simpler/goversion/branch/main/graph/badge.svg)](https://codecov.io/gh/go-simpler/goversion)

Easily switch between multiple Go versions.

## 📌 About

Go supports [installing multiple versions][1] simultaneously as separate binaries,
such as `go` (the main version) and `go1.18` (an add-on version).
This works fine when using `go <command>` directly,
but can be inconvenient when the command is hardcoded in a `Makefile` or a shell script.
The `goversion` tool solves this by symlinking `go1.X.Y` to `go`,
so that an add-on version can be used as the main one.

```shell
> go version
go version go1.20 darwin/arm64

> goversion use 1.18
1.18 is not installed. Looking for it on go.dev ...
# Downloading ...
Switched to 1.18

> go version
go version go1.18 darwin/arm64
```

## 🚀 Features

* Install and switch between multiple Go versions
* List installed Go versions (optionally, all available versions)
* Remove installed Go versions with a single command

## 📦 Install

First, add `$GOBIN` (usually `$HOME/go/bin`) to your `$PATH`.
Make sure it takes precedence over the location of the main `go` binary (e.g. `/usr/local/go/bin` or `/opt/homebrew/bin`).

Then install `goversion` with Go...

```shell
go install go-simpler.org/goversion@latest
```

...or download a prebuilt binary from the [Releases][2] page.

## 📋 Usage

### Use

Switches the current Go version (will be installed if not exists).

```shell
> goversion use 1.18
Switched to 1.18
```

The special [gotip][3] version can be used just like any other.

```shell
> goversion use tip
Switched to tip
```

To switch back to the main version, use the `main` string.

```shell
> goversion use main
Switched to 1.20 (main)
```

### List

Prints the list of installed Go versions.
The current version is marked with the `*` symbol.

```shell
> goversion ls
  1.20 (main)
* 1.18
```

The `-a (-all)` flag can be used to print also available versions from `go.dev`.

```shell
> goversion ls -all
  tip     (not installed)
  1.20.14 (not installed)
  1.20.13 (not installed)
# ...
  1.3rc1  (not installed)
  1.2.2   (not installed)
  1       (not installed)
```

The `-only=<prefix>` flag can be used to print only versions starting with the prefix.

```shell
> goversion ls -all -only=1.18
  1.18.10   (not installed)
  1.18.9    (not installed)
  1.18.8    (not installed)
# ...
  1.18rc1   (not installed)
  1.18beta2 (not installed)
  1.18beta1 (not installed)
```

If the `-only=latest` combination is given, `ls` prints only the latest patch for each version.

```shell
> goversion ls -all -only=latest
  tip     (not installed)
  1.20.14 (not installed)
  1.19.13 (not installed)
# ...
  1.3.3   (not installed)
  1.2.2   (not installed)
  1       (not installed)
```

### Remove

Removes the specified Go version (both binary and SDK).

```shell
> goversion rm 1.18
Removed 1.18
```

### Help

```shell
Usage: goversion [flags] <command> [command flags]

Commands:
    use main              switch to the main Go version
    use <version>         switch to the specified Go version (will be installed if not exists)
    ls                    print the list of installed Go versions
        -a (-all)         print also available versions from go.dev
        -only=<prefix>    print only versions starting with the prefix
        -only=latest      print only the latest patch for each version
    rm <version>          remove the specified Go version (both binary and SDK)

Flags:
    -h (-help)            print this message and quit
    -v (-version)         print the version of goversion itself and quit
```

[1]: https://go.dev/doc/manage-install
[2]: https://github.com/go-simpler/goversion/releases
[3]: https://pkg.go.dev/golang.org/dl/gotip
