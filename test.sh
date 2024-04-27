#!/bin/sh

set -euf

os="$1"
exe=""
if [ "$os" = "windows-latest" ]; then
    exe=".exe"
fi

dump_version() {
    file="$1"
    hash -r # refresh binary paths
    go version | awk '{print $3}' > "$file"
}

go build -o goversion"$exe"

version="1.18"
echo "go$version" > want
dump_version main

echo "Switching to $version"
./goversion"$exe" use "$version"
dump_version got
diff got want

echo "Installed versions"
./goversion"$exe" ls

echo "Removing $version"
./goversion"$exe" rm "$version"
dump_version got
diff got main
