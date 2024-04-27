#!/bin/sh

set -euf

os="$1"
exe=""
if [ "$os" = "windows-latest" ]; then
    exe=".exe"
fi

version="1.18"
go build -o goversion"$exe"
echo "Switching to $version"
./goversion"$exe" use "$version"
echo "Installed versions"
./goversion"$exe" ls
hash -r # refresh binary paths
go version | awk '{print $3}' > got
echo "go$version" > want
diff got want
