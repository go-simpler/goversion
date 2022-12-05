// https://go.dev/doc/manage-install
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
)

// a list of available Go versions from https://go.dev/dl
// TODO(junk1tm): fill the list
var availableVersions = map[string]struct{}{
	"1.17":   {},
	"1.18":   {},
	"1.19":   {},
	"1.19.1": {},
	"1.19.2": {},
	"1.19.3": {},
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("goversion: ")

	if err := run(); err != nil {
		log.Println(err)
		if errors.As(err, new(usageError)) {
			usage()
			os.Exit(2)
		}
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return usageError{errors.New("no command has been specified")}
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	app, err := newApp(ctx)
	if err != nil {
		return err
	}

	cmd, args := os.Args[1], os.Args[2:]

	switch cmd {
	case "use":
		return app.use(ctx, args)
	case "list", "ls":
		return app.list(ctx, args)
	case "remove", "rm":
		return app.remove(ctx, args)
	case "-h", "-help":
		usage()
		return nil
	default:
		return usageError{fmt.Errorf("unknown command %q", cmd)}
	}
}

// app holds values used by all the commands.
type app struct {
	path struct {
		sdk   string
		gobin string
	}
	version struct {
		main    string
		current string
	}
}

// newApp initializes and returns a new app.
func newApp(ctx context.Context) (*app, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	var app app
	app.path.sdk = filepath.Join(home, "sdk")
	app.path.gobin = filepath.Join(home, "go", "bin")

	app.version.current, err = currentVersion(ctx)
	if err != nil {
		return nil, err
	}

	app.version.main, err = app.mainVersion(ctx)
	if err != nil {
		return nil, err
	}

	return &app, nil
}

// use switches the current Go version to the one specified.
// If it's not installed, use will download it first.
func (a *app) use(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return usageError{errors.New("no version has been specified")}
	}

	version := args[0]
	if version == "main" {
		version = a.version.main
	}

	if _, ok := availableVersions[version]; !ok {
		return usageError{fmt.Errorf("invalid version %q", version)}
	}

	if version == a.version.current {
		log.Printf("%s is already in use", version)
		return nil
	}

	if version == a.version.main {
		// for switching to the main version simply removing the symlink is enough.
		symlink := filepath.Join(a.path.gobin, "go")
		if err := os.Remove(symlink); err != nil {
			return err
		}
		log.Printf("switched to %s", version)
		return nil
	}

	binary := filepath.Join(a.path.gobin, "go"+version)
	if notExist(binary) {
		url := fmt.Sprintf("golang.org/dl/go%s@latest", version)
		if err := command(ctx, "go", "install", url); err != nil {
			return err
		}
	}

	sdk := filepath.Join(a.path.sdk, "go"+version)
	if notExist(sdk) {
		if err := command(ctx, binary, "download"); err != nil {
			return err
		}
	}

	symlink := filepath.Join(a.path.gobin, "go")
	if err := os.Remove(symlink); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Symlink(binary, symlink); err != nil {
		return err
	}

	log.Printf("switched to %s", version)
	return nil
}

// list prints a list of installed Go versions, highlighting the current one.
func (a *app) list(_ context.Context, _ []string) error {
	// TODO(junk1tm): support -all
	// var all bool
	// fset := flag.NewFlagSet("list", flag.ContinueOnError)
	// fset.BoolVar(&all, "all", false, "print all available versions")
	// fset.BoolVar(&all, "a", false, "shorthand for -all")
	// if err := fset.Parse(args); err != nil {
	// 	return err
	// }

	entries, err := os.ReadDir(a.path.gobin)
	if err != nil {
		return err
	}

	var sb strings.Builder
	sb.WriteString("\n\n")

	prefixIf := func(cond bool) string {
		if cond {
			return "*"
		}
		return " "
	}

	fmt.Fprintf(&sb, "%s %-7s (main, installed)\n",
		prefixIf(a.version.main == a.version.current), a.version.main)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		version := entry.Name()
		version = strings.TrimPrefix(version, "go")
		if _, ok := availableVersions[version]; !ok {
			continue
		}

		sdk := filepath.Join(a.path.sdk, "go"+version)

		props := "installed"
		if notExist(sdk) {
			props = "not installed"
		}

		fmt.Fprintf(&sb, "%s %-7s (%s)\n",
			prefixIf(version == a.version.current), version, props)
	}

	log.Println(sb.String())
	return nil
}

// remove uninstalls the specified Go version.
// If this version is current, remove will switch to the main one first.
func (a *app) remove(_ context.Context, args []string) error {
	if len(args) == 0 {
		return usageError{errors.New("no version has been specified")}
	}

	version := args[0]
	if version == "main" {
		version = a.version.main
	}

	if _, ok := availableVersions[version]; !ok {
		return usageError{fmt.Errorf("invalid version %q", version)}
	}

	if version == a.version.main {
		return fmt.Errorf("unable to remove %s (main)", version)
	}

	if version == a.version.current {
		// switch to the main version first.
		symlink := filepath.Join(a.path.gobin, "go")
		if err := os.Remove(symlink); err != nil {
			return err
		}
		log.Printf("switched to %s (main)", a.version.main)
	}

	binary := filepath.Join(a.path.gobin, "go"+version)
	if err := os.Remove(binary); err != nil {
		return err
	}

	sdk := filepath.Join(a.path.sdk, "go"+version)
	if err := os.RemoveAll(sdk); err != nil {
		return err
	}

	log.Printf("removed %s", version)
	return nil
}

// command is a wrapper for exec.Command that redirects stdout/stderr to log.
func command(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = logWriter{}
	cmd.Stderr = logWriter{}

	log.Printf("running %s %s ...", name, strings.Join(args, " "))

	if err := cmd.Run(); err != nil {
		// var exitErr *exec.ExitError
		// if errors.As(err, &exitErr) {}
		return err
	}

	return nil
}

type logWriter struct{}

func (w logWriter) Write(p []byte) (int, error) {
	log.Print(string(p))
	return len(p), nil
}

// notExist returns true if the file does not exist.
func notExist(name string) bool {
	_, err := os.Stat(name)
	return errors.Is(err, os.ErrNotExist)
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

// currentVersion returns the current Go version.
func currentVersion(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "go", "version")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return parseGoVersion(string(out))
}

// mainVersion returns the main Go version, the one that was used to install goversion itself.
func (a *app) mainVersion(ctx context.Context) (string, error) {
	symlink := filepath.Join(a.path.gobin, "go")
	if notExist(symlink) {
		// the main version is already in use.
		return a.version.current, nil
	}

	// to make exec.Command use the real go binary,
	// we need to temporarily remove $GOBIN from $PATH.
	realPath := os.Getenv("PATH")
	defer os.Setenv("PATH", realPath)
	// log.Printf("real path: %s", realPath)

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
	// log.Printf("temp path: %s", tempPath)

	cmd := exec.CommandContext(ctx, "go", "version")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return parseGoVersion(string(out))
}

type usageError struct{ error }

func usage() {
	fmt.Printf("usage: TODO\n")
}
