# goversion

[![checks](https://github.com/tmzane/goversion/actions/workflows/checks.yml/badge.svg)](https://github.com/tmzane/goversion/actions/workflows/checks.yml)
[![pkg.go.dev](https://pkg.go.dev/badge/go.tmz.dev/goversion.svg)](https://pkg.go.dev/go.tmz.dev/goversion)
[![goreportcard](https://goreportcard.com/badge/go.tmz.dev/goversion)](https://goreportcard.com/report/go.tmz.dev/goversion)
[![codecov](https://codecov.io/gh/tmzane/goversion/branch/main/graph/badge.svg)](https://codecov.io/gh/tmzane/goversion)

Easily switch between multiple Go versions

## 📌 About

Go already supports [installing multiple versions][1] simultaneously as separate binaries,
e.g. `go` (the main version) and `go1.18` (an additional version).
It works just fine when interacting with `go <command>` directly,
but could be inconvenient when the command is wrapped with something like `Makefile` or shell scripts.
The `goversion` tool attempts to solve this by symlinking `go1.X.Y` to `go`,
so any additional Go version could be used as if it was the main one.

```shell
> go version
go version go1.19 darwin/arm64

> goversion use 1.18
1.18 is not installed. Looking for it on go.dev ...
# Downloading ...
Switched to 1.18

> go version
go version go1.18 darwin/arm64
```

## 🚀 Features

* Use any additional Go version as the main one
* List installed Go versions (and, optionally, all available versions)
* Remove an installed Go version with a single command

## ✏️ Pre-requirements

`$GOBIN` (usually `$HOME/go/bin`) must be in your `$PATH` and it must take precedence over the location of the main Go binary (e.g. `/usr/local/go/bin` or `/opt/homebrew/bin`).

## 📦 Install

### Go

```shell
go install go.tmz.dev/goversion@latest
```

### Brew

```shell
brew install tmzane/tap/goversion
```

### Manual

Download a prebuilt binary from the [Releases][2] page.

## 📋 Commands

### Use

Switches the current Go version (will be installed if not already exists).

```shell
> goversion use 1.18
Switched to 1.18
```

As a special case, the `main` string can be provided to quickly switch to the main version.

```shell
> goversion use main
Switched to 1.19 (main)
```

The `gotip` version can be used just like any other.

```shell
> goversion use tip
Switched to tip
```

To update it, first switch to a stable Go version and then run `gotip download`.

### List

Prints the list of installed Go versions.
The current version is marked with the `*` symbol.

```shell
> goversion ls
  1.19       (main)
* 1.18      
  1.17      
```

The `-a (-all)` flag can be provided to print available versions from `go.dev` as well.

```shell
> goversion ls -a
  1.19.4     (not installed)
  1.19.3     (not installed)
# ...
  1.19       (main)
# ...
  1.2.2      (not installed)
  1          (not installed)
```

The full list is quite long, to limit it the `-only=<prefix>` flag can be used.

```shell
> goversion ls -a -only=1.18
  1.18.9     (not installed)
  1.18.8     (not installed)
# ...
* 1.18      
# ...
  1.18beta2  (not installed)
  1.18beta1  (not installed)
```

If the `-only=latest` combination is provided, `ls` prints only the latest patch for each minor version.

```shell
> goversion ls -a -only=latest
  1.19.5     (not installed)
  1.18.10    (not installed)
# ...
  1.10.8     (not installed)
# ...
  1.2.2      (not installed)
  1          (not installed)
```

### Remove

Removes the specified Go version (both the binary and the SDK).

```shell
> goversion rm 1.18
Removed 1.18
```

[1]: https://go.dev/doc/manage-install
[2]: https://github.com/tmzane/goversion/releases
