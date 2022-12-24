package main

import (
	"context"
	"fmt"
	"io/fs"
	"strings"
	"testing"

	"github.com/go-simpler/assert"
	. "github.com/go-simpler/assert/dotimport"
)

func Test_versionRE(t *testing.T) {
	test := func(s string, match bool) {
		t.Helper()
		got := versionRE.MatchString(s)
		assert.Equal[E](t, got, match)
	}

	test("tip", true)
	test("top", false)
	test("1", true)
	test("1.", false)
	test("1.18", true)
	test("1.18rc", false)
	test("1.18rc1", true)
	test("1.18alpha1", false)
	test("1.18beta1", true)
	test("1.18.", false)
	test("1.18.10", true)
	test("1.18.10.", false)
}

const mainVersion = "1.19"

var ctx = context.Background()

func Test_use(t *testing.T) {
	t.Run("install fresh version", func(t *testing.T) {
		var steps []string
		recordCommands(&steps)

		gobin = &spyFS{dir: "gobin", calls: &steps}
		sdk = &spyFS{dir: "sdk", calls: &steps}

		err := use(ctx, []string{"1.18"})
		assert.NoErr[F](t, err)
		assert.Equal[E](t, steps, []string{
			"command: go version",
			"call: gobin.Readlink(go)",
			"call: gobin.ReadDir(.)",
			"command: go install golang.org/dl/go1.18@latest",
			"call: sdk.Stat(go1.18/.unpacked-success)",
			"command: go1.18 download",
			"call: gobin.Remove(go)",
			"call: gobin.Symlink(go1.18, go)",
		})
	})

	t.Run("switch to the current version", func(t *testing.T) {
		var steps []string
		recordCommands(&steps)

		gobin = &spyFS{
			dir:   "gobin",
			link:  "/path/to/go1.18",
			files: []dirFile{"go1.18"},
			calls: &steps,
		}
		sdk = &spyFS{
			dir:   "sdk",
			files: []dirFile{"go1.18/.unpacked-success"},
			calls: &steps,
		}

		err := use(ctx, []string{"1.18"})
		assert.NoErr[F](t, err)
		assert.Equal[E](t, steps, []string{
			"command: go version",
			"call: gobin.Readlink(go)",
			"call: gobin.ReadDir(.)",
		})
	})

	t.Run("switch to the main version", func(t *testing.T) {
		var steps []string
		recordCommands(&steps)

		gobin = &spyFS{
			dir:   "gobin",
			link:  "/path/to/go1.18",
			files: []dirFile{"go1.18"},
			calls: &steps,
		}
		sdk = &spyFS{
			dir:   "sdk",
			files: []dirFile{"go1.18/.unpacked-success"},
			calls: &steps,
		}

		err := use(ctx, []string{"main"})
		assert.NoErr[F](t, err)
		assert.Equal[E](t, steps, []string{
			"command: go version",
			"call: gobin.Readlink(go)",
			"call: gobin.ReadDir(.)",
			"call: gobin.Remove(go)",
		})
	})
}

func recordCommands(commands *[]string) {
	command = func(ctx context.Context, name string, args ...string) error {
		c := strings.Join(append([]string{name}, args...), " ")
		*commands = append(*commands, "command: "+c)
		return nil
	}
	commandOutput = func(ctx context.Context, name string, args ...string) (string, error) {
		_ = command(ctx, name, args...)
		return fmt.Sprintf("go version go%s darwin/arm64", mainVersion), nil
	}
}

type spyFS struct {
	dir   string
	link  string
	files []dirFile
	calls *[]string
}

func (s *spyFS) Open(name string) (fs.File, error) {
	// won't be called because Stat is implemented.
	panic("unimplemented")
}

func (s *spyFS) Stat(name string) (fs.FileInfo, error) {
	*s.calls = append(*s.calls, fmt.Sprintf("call: %s.Stat(%s)", s.dir, name))
	for _, f := range s.files {
		if string(f) == name {
			return nil, nil
		}
	}
	return nil, fs.ErrNotExist
}

func (s *spyFS) Remove(name string) error {
	*s.calls = append(*s.calls, fmt.Sprintf("call: %s.Remove(%s)", s.dir, name))
	return nil
}

func (s *spyFS) RemoveAll(name string) error {
	*s.calls = append(*s.calls, fmt.Sprintf("call: %s.RemoveAll(%s)", s.dir, name))
	return nil
}

func (s *spyFS) Symlink(oldname, newname string) error {
	*s.calls = append(*s.calls, fmt.Sprintf("call: %s.Symlink(%s, %s)", s.dir, oldname, newname))
	return nil
}

func (s *spyFS) Readlink(name string) (string, error) {
	*s.calls = append(*s.calls, fmt.Sprintf("call: %s.Readlink(%s)", s.dir, name))
	if s.link == "" {
		return "", fs.ErrNotExist
	}
	return s.link, nil
}

func (s *spyFS) ReadDir(name string) ([]fs.DirEntry, error) {
	*s.calls = append(*s.calls, fmt.Sprintf("call: %s.ReadDir(%s)", s.dir, name))
	entries := make([]fs.DirEntry, len(s.files))
	for i, f := range s.files {
		entries[i] = f
	}
	return entries, nil
}

type dirFile string

func (f dirFile) Name() string               { return string(f) }
func (f dirFile) IsDir() bool                { return false }
func (f dirFile) Type() fs.FileMode          { panic("unimplemented") }
func (f dirFile) Info() (fs.FileInfo, error) { panic("unimplemented") }
