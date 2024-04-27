#!/bin/sh

set -euf

os="$1"
exe=""
if [ "$os" = "windows-latest" ]; then
    exe=".exe"
fi

go build -o goversion"$exe"
./goversion"$exe" use 1.18
./goversion"$exe" ls
go version | awk '{print $3}' > got
echo "go1.18" > want
diff got want
