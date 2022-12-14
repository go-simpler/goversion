package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// app holds values used by all the commands.
type app struct {
	path struct {
		sdk     string
		gobin   string
		symlink string
	}
	version struct {
		main      string
		current   string
		installed []string
	}
	client *http.Client
}

// newApp initializes and returns a new app.
func newApp(ctx context.Context) (*app, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	gobin, ok := os.LookupEnv("GOBIN")
	if !ok {
		gobin = filepath.Join(home, "go", "bin")
	}

	var app app
	app.path.sdk = filepath.Join(home, "sdk")
	app.path.gobin = gobin
	app.path.symlink = filepath.Join(gobin, "go")

	app.version.current, err = app.currentVersion(ctx)
	if err != nil {
		return nil, err
	}

	app.version.main, err = app.mainVersion(ctx)
	if err != nil {
		return nil, err
	}

	app.version.installed, err = app.installedVersions()
	if err != nil {
		return nil, err
	}

	app.client = &http.Client{Timeout: time.Minute}

	return &app, nil
}

// use switches the current Go version to the one specified.
// If it's not installed, use will install it and download its SDK first.
func (a *app) use(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return usageError{errors.New("no version has been specified")}
	}

	version := args[0]
	if version == "main" {
		version = a.version.main
	}

	if version == a.version.current {
		printf("%s is already in use\n", version)
		return nil
	}

	if version == a.version.main {
		// for switching to the main version simply removing the symlink is enough.
		if err := os.Remove(a.path.symlink); err != nil {
			return err
		}
		printf("Switched to %s (main)\n", version)
		return nil
	}

	if !a.installed(version) {
		printf("%s is not installed. Looking for it on go.dev ...\n", version)
		if err := a.install(ctx, version); err != nil {
			return err
		}
	}

	binary := filepath.Join(a.path.gobin, "go"+version)

	// it's possible that SDK download was canceled during initial installation,
	// so we need to ensure its presence even if $GOBIN/go<version> exists.
	if !a.downloaded(version) {
		printf("%s SDK is missing. Starting download ...\n", version)
		if err := command(ctx, binary, "download"); err != nil {
			return err
		}
	}

	if err := os.Remove(a.path.symlink); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Symlink(binary, a.path.symlink); err != nil {
		return err
	}

	printf("Switched to %s\n", version)
	return nil
}

// install installs the specified Go version and downloads its SDK.
func (a *app) install(ctx context.Context, version string) error {
	versions, err := a.allVersions(ctx)
	if err != nil {
		return err
	}

	exists := false
	for _, v := range versions {
		if v == version {
			exists = true
			break
		}
	}
	if !exists {
		return fmt.Errorf("malformed version %q", version)
	}

	url := fmt.Sprintf("golang.org/dl/go%s@latest", version)
	if err := command(ctx, "go", "install", url); err != nil {
		return err
	}

	binary := filepath.Join(a.path.gobin, "go"+version)
	if err := command(ctx, binary, "download"); err != nil {
		return err
	}

	return nil
}

// installed checks whether the specified Go version has been installed.
func (a *app) installed(version string) bool {
	for _, v := range a.version.installed {
		if v == version {
			return true
		}
	}
	return false
}

// downloaded checks whether the SDK of the specified Go version has been downloaded.
func (a *app) downloaded(version string) bool {
	// from https://github.com/golang/dl/blob/master/internal/version/version.go
	// .unpacked-success is a sentinel zero-byte file to indicate that the Go
	// version was downloaded and unpacked successfully.
	name := filepath.Join(a.path.sdk, "go"+version, ".unpacked-success")
	_, err := os.Stat(name)
	return err == nil
}

// list prints the list of installed Go versions, highlighting the current one.
// If the -all flag is provided, list prints available versions from go.dev as well.
func (a *app) list(ctx context.Context, args []string) error {
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

	printVersion := func(version string, installed bool) {
		if !strings.HasPrefix(version, only) {
			return
		}

		var extra string
		switch {
		case version == a.version.main:
			extra = " (main)"
		case !installed:
			extra = " (not installed)"
		case !a.downloaded(version):
			extra = " (missing SDK)"
		}

		prefix := " "
		if version == a.version.current {
			prefix = "*"
		}

		printf("%s %-10s%s\n", prefix, version, extra)
	}

	printVersion(a.version.main, true)

	for _, version := range a.version.installed {
		printVersion(version, true)
	}

	if printAll {
		all, err := a.allVersions(ctx)
		if err != nil {
			return err
		}

		printf("\n")
		for _, version := range all {
			if version == a.version.main || a.installed(version) {
				continue
			}
			printVersion(version, false)
		}
	}

	return nil
}

// remove removes the specified Go version (both the binary and the SDK).
// If this version is current, remove will switch to the main one first.
func (a *app) remove(_ context.Context, args []string) error {
	if len(args) == 0 {
		return usageError{errors.New("no version has been specified")}
	}

	version := args[0]
	if version == "main" {
		version = a.version.main
	}

	if version == a.version.main {
		return fmt.Errorf("unable to remove %s (main)", version)
	}

	if !a.installed(version) {
		return fmt.Errorf("%s is not installed", version)
	}

	if version == a.version.current {
		// switch to the main version first.
		if err := os.Remove(a.path.symlink); err != nil {
			return err
		}
		printf("Switched to %s (main)\n", a.version.main)
	}

	binary := filepath.Join(a.path.gobin, "go"+version)
	if err := os.Remove(binary); err != nil {
		return err
	}

	sdk := filepath.Join(a.path.sdk, "go"+version)
	if err := os.RemoveAll(sdk); err != nil {
		return err
	}

	printf("Removed %s\n", version)
	return nil
}

// currentVersion returns the current Go version.
func (a *app) currentVersion(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "go", "version")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return parseGoVersion(string(out))
}

// mainVersion returns the main Go version, the one that was used to install goversion itself.
func (a *app) mainVersion(ctx context.Context) (string, error) {
	if _, err := os.Stat(a.path.symlink); errors.Is(err, os.ErrNotExist) {
		// the main version is already in use.
		return a.version.current, nil
	}

	// to make exec.Command use the real go binary,
	// we need to temporarily remove $GOBIN from $PATH.
	realPath := os.Getenv("PATH")
	defer os.Setenv("PATH", realPath)

	var parts []string
	for _, part := range strings.Split(realPath, string(os.PathListSeparator)) {
		// $HOME/sdk/go<version>/bin is added by the go binary, filter it too.
		sdkBin := filepath.Join(a.path.sdk, "go"+a.version.current, "bin")
		if part != a.path.gobin && part != sdkBin {
			parts = append(parts, part)
		}
	}

	tempPath := strings.Join(parts, string(os.PathListSeparator))
	os.Setenv("PATH", tempPath)

	cmd := exec.CommandContext(ctx, "go", "version")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return parseGoVersion(string(out))
}

//nolint:gocritic // regexpSimplify: [0-9] reads better here than \d
var versionRE = regexp.MustCompile(`^go1(\.[1-9][0-9]*)?(\.[1-9][0-9]*)?((rc|beta)[1-9]+)?$`)

// installedVersions returns the list of installed Go versions.
func (a *app) installedVersions() ([]string, error) {
	entries, err := os.ReadDir(a.path.gobin)
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
func (a *app) allVersions(ctx context.Context) ([]string, error) {
	const url = "https://go.dev/dl/?mode=json&include=all"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := a.client.Do(req)
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

// parseGoVersion returns the semver part of a full Go version
// (e.g. `go version go1.18 darwin/arm64` -> `1.18`).
func parseGoVersion(s string) (string, error) {
	parts := strings.Split(s, " ")
	if len(parts) != 4 {
		return "", fmt.Errorf("unexpected format %q", s)
	}
	return strings.TrimPrefix(parts[2], "go"), nil
}

// command is a wrapper for exec.Command that redirects stdout/stderr.
func command(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
