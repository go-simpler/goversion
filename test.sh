#!/bin/sh

set -euf

os="$1"
exe=""
if [ "$os" = "windows-latest" ]; then
    exe=".exe"
fi

go build -o goversion"$exe"
./goversion"$exe" use 1.18
which -a go
"$HOME/go/bin/go" version
