// Package app implements the core logic of the goversion tool.
package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"go-simpler.org/goversion/fsx"
)

type App struct {
	GoBin, SDK fsx.FS
	Output     io.Writer
	RunCmd     func(ctx context.Context, name string, args ...string) error
	RunCmdOut  func(ctx context.Context, name string, args ...string) (string, error)
	Requester  interface {
		Do(*http.Request) (*http.Response, error)
	}
}

func (a *App) Use(ctx context.Context, version string) error {
	local, err := a.localVersions(ctx)
	if err != nil {
		return err
	}

	if version == "main" {
		version = local.main
	}

	if !isValid(version) {
		return fmt.Errorf("malformed version %q", version)
	}

	switch version {
	case local.current:
		fmt.Fprintf(a.Output, "%s is already in use\n", version)
		return nil
	case local.main:
		if err := a.GoBin.Remove("go" + exe()); err != nil {
			return err
		}
		fmt.Fprintf(a.Output, "Switched to %s (main)\n", version)
		return nil
	}

	initial := false
	if !slices.Contains(local.list, version) {
		initial = true
		fmt.Fprintf(a.Output, "%s is not installed. Looking for it on go.dev ...\n", version)
		url := fmt.Sprintf("golang.org/dl/go%s@latest", version)
		if err := a.RunCmd(ctx, "go"+exe(), "install", url); err != nil {
			return err
		}
	}

	// it's possible that SDK download was canceled during initial installation,
	// so we need to ensure its presence even if the go<version> binary exists.
	if !a.downloaded(version) {
		if !initial {
			// this message doesn't make sense during initial installation.
			fmt.Fprintf(a.Output, "%s SDK is missing. Starting download ...\n", version)
		}
		if err := a.RunCmd(ctx, "go"+version+exe(), "download"); err != nil {
			return err
		}
	}

	if err := a.GoBin.Remove("go" + exe()); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := a.GoBin.Symlink("go"+version+exe(), "go"+exe()); err != nil {
		return err
	}

	fmt.Fprintf(a.Output, "Switched to %s\n", version)
	return nil
}

func (a *App) List(ctx context.Context, printAll bool, printOnly string) error {
	local, err := a.localVersions(ctx)
	if err != nil {
		return err
	}

	versions := local.list
	if printAll {
		if versions, err = a.remoteVersions(ctx); err != nil {
			return err
		}
	}

	if printOnly == "latest" {
		printOnly = ""
		versions = latestPatches(versions)
	}

	var maxLen int
	for _, version := range versions {
		maxLen = max(maxLen, len(version))
	}

	for _, version := range versions {
		if !strings.HasPrefix(version, printOnly) {
			continue
		}

		var extra string
		switch {
		case version == local.main:
			extra = " (main)"
		case !slices.Contains(local.list, version):
			extra = " (not installed)"
		case !a.downloaded(version):
			extra = " (missing SDK)"
		}

		prefix := " "
		if version == local.current {
			prefix = "*"
		}

		fmt.Fprintf(a.Output, "%s %-*s%s\n", prefix, maxLen, version, extra)
	}

	return nil
}

func (a *App) Remove(ctx context.Context, version string) error {
	local, err := a.localVersions(ctx)
	if err != nil {
		return err
	}

	if version == "main" {
		version = local.main
	}

	if !isValid(version) {
		return fmt.Errorf("malformed version %q", version)
	}

	if !slices.Contains(local.list, version) {
		return fmt.Errorf("%s is not installed", version)
	}

	switch version {
	case local.main:
		return fmt.Errorf("unable to remove %s (main)", version)
	case local.current:
		if err := a.GoBin.Remove("go" + exe()); err != nil {
			return err
		}
		fmt.Fprintf(a.Output, "Switched to %s (main)\n", local.main)
	}

	if err := a.GoBin.Remove("go" + version + exe()); err != nil {
		return err
	}
	if err := a.SDK.RemoveAll("go" + version); err != nil {
		return err
	}

	fmt.Fprintf(a.Output, "Removed %s\n", version)
	return nil
}

func (a *App) downloaded(version string) bool {
	// from https://github.com/golang/dl/blob/master/internal/version/version.go:
	// .unpacked-success is a sentinel zero-byte file to indicate that the Go version was downloaded and unpacked successfully.
	name := "go" + version + "/.unpacked-success"
	if version == "tip" {
		name = "gotip/bin/go" // https://github.com/golang/dl/blob/master/internal/version/gotip.go#L45
	}
	_, err := fs.Stat(a.SDK, name)
	return err == nil
}

type local struct {
	main    string
	current string
	list    []string // includes both main and current.
}

func (a *App) localVersions(ctx context.Context) (*local, error) {
	currPath := os.Getenv("PATH")
	defer os.Setenv("PATH", currPath)

	// temporarily remove $GOBIN from $PATH to force [exec.Command] to use the main go binary.
	tempPath := cutFromPath(currPath, os.Getenv("GOBIN"))
	os.Setenv("PATH", tempPath)

	output, err := a.RunCmdOut(ctx, "go"+exe(), "version")
	if err != nil {
		return nil, err
	}

	var main string
	if _, err := fmt.Fscanf(strings.NewReader(output), "go version go%s", &main); err != nil {
		return nil, fmt.Errorf("unexpected format %q", output)
	}

	var current string
	switch link, err := a.GoBin.Readlink("go" + exe()); {
	case errors.Is(err, fs.ErrNotExist):
		current = main
	case err == nil:
		current = strings.TrimPrefix(filepath.Base(link), "go") // TODO: windows: trim .exe?
	default:
		return nil, err
	}

	entries, err := fs.ReadDir(a.GoBin, ".")
	if err != nil {
		return nil, err
	}

	list := []string{main}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		// TODO: windows: trim .exe?
		version := strings.TrimPrefix(entry.Name(), "go")
		if isValid(version) {
			list = append(list, version)
		}
	}

	sort.Slice(list, func(i, j int) bool {
		return versionLess(list[i], list[j])
	})

	return &local{
		main:    main,
		current: current,
		list:    list,
	}, nil
}

func (a *App) remoteVersions(ctx context.Context) ([]string, error) {
	// sorted by version, from newest to oldest.
	const url = "https://go.dev/dl/?mode=json&include=all"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := a.Requester.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var list []struct {
		Version string `json:"version"`
		Stable  bool   `json:"stable"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, err
	}

	versions := make([]string, len(list)+1)
	versions[0] = "tip"
	for i, v := range list {
		versions[i+1] = strings.TrimPrefix(v.Version, "go")
	}

	return versions, nil
}
