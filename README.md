# goversion

[![ci](https://github.com/junk1tm/goversion/actions/workflows/go.yml/badge.svg)](https://github.com/junk1tm/goversion/actions/workflows/go.yml)
[![docs](https://pkg.go.dev/badge/github.com/junk1tm/goversion.svg)](https://pkg.go.dev/github.com/junk1tm/goversion)
[![report](https://goreportcard.com/badge/github.com/junk1tm/goversion)](https://goreportcard.com/report/github.com/junk1tm/goversion)
[![codecov](https://codecov.io/gh/junk1tm/goversion/branch/main/graph/badge.svg)](https://codecov.io/gh/junk1tm/goversion)

Easily switch between multiple Go versions

## 📌 About

Go already supports [installing multiple versions][1] simultaneously as separate binaries,
e.g. `go` (the main version) and `go1.19` (an additional version).
It works just fine when interacting with `go <command>` directly,
but could be inconvenient when the command is wrapped with something like `Makefile` or shell scripts.
The `goversion` tool attempts to solve this by symlinking `go1.X.Y` to `go`,
so any additional Go version could be used as if it was the main one.

```shell
> go version
go version go1.18 darwin/arm64

> goversion use 1.19
1.19 is not installed. Looking for it on go.dev ...
# Downloading ...
Switched to 1.19

> go version
go version go1.19 darwin/arm64
```

## 🚀 Features

* Use any additional Go version as the main one
* List installed Go versions (and, optionally, all available versions)
* Remove an installed Go version with a single command

## ✏️ Pre-requirements

`$GOBIN` (usually `$HOME/go/bin`) must be in your `$PATH` and it must take precedence over the location of the main Go binary (e.g. `/opt/homebrew/bin`).

## 📦 Install

### Go

```shell
go install github.com/junk1tm/goversion@latest
```

### Brew

```shell
brew install junk1tm/tap/goversion
```

### Manual

Download a prebuilt binary from the [Releases][2] page.

[1]: https://go.dev/doc/manage-install
[2]: https://github.com/junk1tm/goversion/releases
