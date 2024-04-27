#!/bin/sh

set -euf

os="$1"
if [ "$os" = "windows-latest" ]; then
    exe=".exe"
fi

go build -o goversion"$exe"
./goversion"$exe" use 1.18
go version