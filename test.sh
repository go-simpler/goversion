#!/bin/sh

set -euf

os="$1"
exe=""
if [ "$os" = "windows-latest" ]; then
    exe=".exe"
fi

version="1.18"
go build -o goversion"$exe"
./goversion"$exe" use "$version"
hash -r # refresh binary paths
go version | awk '{print $3}' > got
echo "go$version" > want
diff got want
