package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// use switches the current Go version to the one specified.
// If it's not installed, use will install it and download its SDK first.
func use(ctx context.Context, gobinFS, sdkFS fsx, args []string) error {
	if len(args) == 0 {
		return usageError{errors.New("no version has been specified")}
	}

	main, current, err := getVersions(ctx, gobinFS)
	if err != nil {
		return err
	}

	version := args[0]
	if version == "main" {
		version = main
	}

	if version == current {
		printf("%s is already in use\n", version)
		return nil
	}

	if version == main {
		// for switching to the main version simply removing the symlink is enough.
		if err := gobinFS.Remove("go"); err != nil {
			return err
		}
		printf("Switched to %s (main)\n", version)
		return nil
	}

	installed, err := installedVersions(gobinFS)
	if err != nil {
		return err
	}
	if !contains(installed, version) {
		printf("%s is not installed. Looking for it on go.dev ...\n", version)
		if err := install(ctx, version); err != nil {
			return err
		}
	}

	// it's possible that SDK download was canceled during initial installation,
	// so we need to ensure its presence even if $GOBIN/go<version> exists.
	if !downloaded(version, sdkFS) {
		printf("%s SDK is missing. Starting download ...\n", version)
		if err := command(ctx, "go/bin/go"+version, "download"); err != nil {
			return err
		}
	}

	if err := gobinFS.Remove("go"); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := gobinFS.Symlink("go"+version, "go"); err != nil {
		return err
	}

	printf("Switched to %s\n", version)
	return nil
}

// install installs the specified Go version and downloads its SDK.
func install(ctx context.Context, version string) error {
	all, err := allVersions(ctx)
	if err != nil {
		return err
	}
	if !contains(all, version) {
		return fmt.Errorf("malformed version %q", version)
	}

	url := fmt.Sprintf("golang.org/dl/go%s@latest", version)
	if err := command(ctx, "go", "install", url); err != nil {
		return err
	}

	if err := command(ctx, "go"+version, "download"); err != nil {
		return err
	}

	return nil
}

// downloaded checks whether the SDK of the specified Go version has been downloaded.
func downloaded(version string, sdkFS fs.FS) bool {
	// from https://github.com/golang/dl/blob/master/internal/version/version.go
	// .unpacked-success is a sentinel zero-byte file to indicate that the Go
	// version was downloaded and unpacked successfully.
	_, err := fs.Stat(sdkFS, "go"+version+"/.unpacked-success")
	return err == nil
}

// list prints the list of installed Go versions, highlighting the current one.
// If the -all flag is provided, list prints available versions from go.dev as well.
func list(ctx context.Context, gobinFS, sdkFS fs.FS, args []string) error {
	fset := flag.NewFlagSet("list", flag.ContinueOnError)
	fset.SetOutput(io.Discard)

	var printAll bool
	fset.BoolVar(&printAll, "a", false, "shorthand for -all")
	fset.BoolVar(&printAll, "all", false, "print available versions from go.dev as well")

	var only string
	fset.StringVar(&only, "only", "", "print only versions starting with this prefix")

	if err := fset.Parse(args); err != nil {
		return usageError{err}
	}

	main, current, err := getVersions(ctx, gobinFS)
	if err != nil {
		return err
	}

	installed, err := installedVersions(gobinFS)
	if err != nil {
		return err
	}

	printVersion := func(version string, installed bool) {
		if !strings.HasPrefix(version, only) {
			return
		}

		var extra string
		switch {
		case version == main:
			extra = " (main)"
		case !installed:
			extra = " (not installed)"
		case !downloaded(version, sdkFS):
			extra = " (missing SDK)"
		}

		prefix := " "
		if version == current {
			prefix = "*"
		}

		printf("%s %-10s%s\n", prefix, version, extra)
	}

	printVersion(main, true)

	for _, version := range installed {
		printVersion(version, true)
	}

	if printAll {
		all, err := allVersions(ctx)
		if err != nil {
			return err
		}

		printf("\n")
		for _, version := range all {
			if version == main || contains(installed, version) {
				continue
			}
			printVersion(version, false)
		}
	}

	return nil
}

// remove removes the specified Go version (both the binary and the SDK).
// If this version is current, remove will switch to the main one first.
func remove(ctx context.Context, gobinFS, sdkFS fsx, args []string) error {
	if len(args) == 0 {
		return usageError{errors.New("no version has been specified")}
	}

	main, current, err := getVersions(ctx, gobinFS)
	if err != nil {
		return err
	}

	version := args[0]
	if version == "main" {
		version = main
	}

	if version == main {
		return fmt.Errorf("unable to remove %s (main)", version)
	}

	installed, err := installedVersions(gobinFS)
	if err != nil {
		return err
	}
	if !contains(installed, version) {
		return fmt.Errorf("%s is not installed", version)
	}

	if version == current {
		// switch to the main version first.
		if err := gobinFS.Remove("go"); err != nil {
			return err
		}
		printf("Switched to %s (main)\n", main)
	}

	if err := gobinFS.Remove("go" + version); err != nil {
		return err
	}
	if err := sdkFS.RemoveAll("go" + version); err != nil {
		return err
	}

	printf("Removed %s\n", version)
	return nil
}

// getVersions returns the main and current Go versions.
// The main version is the one that was used to install goversion itself.
func getVersions(ctx context.Context, gobinFS fs.FS) (main, current string, err error) {
	getVersion := func(ctx context.Context) (string, error) {
		out, err := exec.CommandContext(ctx, "go", "version").Output()
		if err != nil {
			return "", err
		}
		// the format is `go version go1.18 darwin/arm64`, we want the semver part.
		parts := strings.Split(string(out), " ")
		if len(parts) != 4 {
			return "", fmt.Errorf("unexpected format %q", out)
		}
		return strings.TrimPrefix(parts[2], "go"), nil
	}

	current, err = getVersion(ctx)
	if err != nil {
		return "", "", err
	}

	if _, err := fs.Stat(gobinFS, "go"); errors.Is(err, fs.ErrNotExist) {
		// the main version is already in use.
		return current, current, nil
	}

	// to make exec.Command use the real go binary,
	// we need to temporarily remove $GOBIN and $GOROOT/bin from $PATH.
	realPath := os.Getenv("PATH")
	defer os.Setenv("PATH", realPath)

	var parts []string
	for _, part := range strings.Split(realPath, string(os.PathListSeparator)) {
		// $GOROOT/bin is added by the go<version> binary ($GOROOT is $HOME/sdk/go<version>).
		// see https://github.com/golang/dl/blob/master/internal/version/version.go#L64.
		gorootBin := filepath.Join(os.Getenv("GOROOT"), "bin")
		if part != os.Getenv("GOBIN") && part != gorootBin {
			parts = append(parts, part)
		}
	}

	tempPath := strings.Join(parts, string(os.PathListSeparator))
	os.Setenv("PATH", tempPath)

	main, err = getVersion(ctx)
	if err != nil {
		return "", "", err
	}

	return main, current, nil
}

//nolint:gocritic // regexpSimplify: [0-9] reads better here than \d
var versionRE = regexp.MustCompile(`^go1(\.[1-9][0-9]*)?(\.[1-9][0-9]*)?((rc|beta)[1-9]+)?$`)

// installedVersions returns the list of installed Go versions.
func installedVersions(gobinFS fs.FS) ([]string, error) {
	entries, err := fs.ReadDir(gobinFS, ".")
	if err != nil {
		return nil, err
	}

	var versions []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if name := entry.Name(); versionRE.MatchString(name) {
			versions = append(versions, strings.TrimPrefix(name, "go"))
		}
	}

	// TODO(junk1tm): fix the order of rc/beta versions
	// reverse to match the order of the go.dev version list (from newest to oldest).
	for i, j := 0, len(versions)-1; i < j; i, j = i+1, j-1 {
		versions[i], versions[j] = versions[j], versions[i]
	}

	return versions, nil
}

// allVersions returns the list of all Go versions from go.dev.
func allVersions(ctx context.Context) ([]string, error) {
	const url = "https://go.dev/dl/?mode=json&include=all"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: time.Minute}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// sorted by version, from newest to oldest.
	var list []struct {
		Version string `json:"version"`
		Stable  bool   `json:"stable"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, err
	}

	versions := make([]string, len(list))
	for i, v := range list {
		versions[i] = strings.TrimPrefix(v.Version, "go")
	}

	return versions, nil
}

// command is a wrapper for exec.Command that redirects stdout/stderr.
func command(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
