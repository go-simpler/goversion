package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
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
	t.Run("install new version", func(t *testing.T) {
		var steps []string
		recordCommands(&steps)

		gobin = &spyFS{dir: "gobin", calls: &steps}
		sdk = &spyFS{dir: "sdk", calls: &steps}

		err := use(ctx, []string{"1.18"})
		assert.NoErr[F](t, err)
		assert.Equal[E](t, steps, []string{
			"exec: go version",                             // 1. read main version
			"call: gobin.Readlink(go)",                     // 2. read current version
			"call: gobin.ReadDir(.)",                       // 3. read installed versions
			"exec: go install golang.org/dl/go1.18@latest", // 4. install 1.18
			"call: sdk.Stat(go1.18/.unpacked-success)",     // 5. check 1.18 SDK
			"exec: go1.18 download",                        // 6. download 1.18 SDK
			"call: gobin.Remove(go)",                       // 7. remove previous symlink
			"call: gobin.Symlink(go1.18, go)",              // 8. create new symlink
		})
	})

	t.Run("switch to current version", func(t *testing.T) {
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

		var buf bytes.Buffer
		output = &buf

		err := use(ctx, []string{"1.18"})
		assert.NoErr[F](t, err)
		assert.Equal[E](t, buf.String(), "1.18 is already in use\n")
		assert.Equal[E](t, steps, []string{
			"exec: go version",         // 1. read main version
			"call: gobin.Readlink(go)", // 2. read current version
			"call: gobin.ReadDir(.)",   // 3. read installed versions
		})
	})

	t.Run("switch to main version", func(t *testing.T) {
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

		var buf bytes.Buffer
		output = &buf

		err := use(ctx, []string{"main"})
		assert.NoErr[F](t, err)
		assert.Equal[E](t, buf.String(), "Switched to 1.19 (main)\n")
		assert.Equal[E](t, steps, []string{
			"exec: go version",         // 1. read main version
			"call: gobin.Readlink(go)", // 2. read current version
			"call: gobin.ReadDir(.)",   // 3. read installed versions
			"call: gobin.Remove(go)",   // 4. remove symlink (switch to main)
		})
	})
}

func Test_list(t *testing.T) {
	t.Run("list local versions", func(t *testing.T) {
		var steps []string
		recordCommands(&steps)

		gobin = &spyFS{
			dir:   "gobin",
			link:  "/path/to/go1.18",
			files: []dirFile{"go1.17", "go1.18"},
			calls: &steps,
		}
		sdk = &spyFS{
			dir:   "sdk",
			files: []dirFile{"go1.18/.unpacked-success"}, // 1.17 SDK is missing.
			calls: &steps,
		}

		var buf bytes.Buffer
		output = &buf

		err := list(ctx, nil)
		assert.NoErr[F](t, err)
		assert.Equal[E](t, "\n"+buf.String(), `
  1.19       (main)
* 1.18      
  1.17       (missing SDK)
`)
		assert.Equal[E](t, steps, []string{
			"exec: go version",                         // 1. read main version
			"call: gobin.Readlink(go)",                 // 2. read current version
			"call: gobin.ReadDir(.)",                   // 3. read installed versions
			"call: sdk.Stat(go1.18/.unpacked-success)", // 4. check 1.18 SDK
			"call: sdk.Stat(go1.17/.unpacked-success)", // 5. check 1.17 SDK
		})
	})

	t.Run("list remote versions", func(t *testing.T) {
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

		var buf bytes.Buffer
		output = &buf

		httpClient = &httpSpy{
			requests: &steps,
			response: `[{"version":"1.19"},{"version":"1.18"},{"version":"1.17"}]`,
		}

		err := list(ctx, []string{"-all"})
		assert.NoErr[F](t, err)
		assert.Equal[E](t, "\n"+buf.String(), `
  tip        (not installed)
  1.19       (main)
* 1.18      
  1.17       (not installed)
`)
		assert.Equal[E](t, steps, []string{
			"exec: go version",                               // 1. read main version
			"call: gobin.Readlink(go)",                       // 2. read current version
			"call: gobin.ReadDir(.)",                         // 3. read installed versions
			"http: https://go.dev/dl/?mode=json&include=all", // 4. get remote versions
			"call: sdk.Stat(go1.18/.unpacked-success)",       // 5. check 1.18 SDK
		})
	})
}

func Test_remove(t *testing.T) {
	t.Run("remove existing version", func(t *testing.T) {
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

		err := remove(ctx, []string{"1.18"})
		assert.NoErr[F](t, err)
		assert.Equal[E](t, steps, []string{
			"exec: go version",            // 1. read main version
			"call: gobin.Readlink(go)",    // 2. read current version
			"call: gobin.ReadDir(.)",      // 3. read installed versions
			"call: gobin.Remove(go)",      // 4. remove symlink (switch to main)
			"call: gobin.Remove(go1.18)",  // 4. remove 1.18 binary
			"call: sdk.RemoveAll(go1.18)", // 5. remove 1.18 SDK
		})
	})
}

func recordCommands(commands *[]string) {
	command = func(ctx context.Context, name string, args ...string) error {
		c := strings.Join(append([]string{name}, args...), " ")
		*commands = append(*commands, "exec: "+c)
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

type httpSpy struct {
	requests *[]string
	response string
}

func (s *httpSpy) Do(req *http.Request) (*http.Response, error) {
	*s.requests = append(*s.requests, "http: "+req.URL.String())
	return &http.Response{Body: io.NopCloser(strings.NewReader(s.response))}, nil
}
