package app_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"slices"
	"strings"
	"testing"

	"go-simpler.org/assert"
	. "go-simpler.org/assert/EF"
	"go-simpler.org/goversion/app"
)

func TestApp_Use(t *testing.T) {
	t.Run("switch to new version", func(t *testing.T) {
		var steps []string

		app := app.App{
			GoBin:  spyFS{dir: "bin", calls: &steps},
			SDK:    spyFS{dir: "sdk", calls: &steps},
			Output: io.Discard,
		}
		recordCmds(&app, &steps, "go version go1.20")

		err := app.Use(context.Background(), "1.18")
		assert.NoErr[F](t, err)
		assert.Equal[E](t, steps, []string{
			`exec: go version`,                             // 1. read main version
			`call: bin.Readlink("go")`,                     // 2. read current version
			`call: bin.ReadDir(".")`,                       // 3. read installed versions
			`exec: go install golang.org/dl/go1.18@latest`, // 4. install 1.18 binary
			`call: sdk.Stat("go1.18/.unpacked-success")`,   // 5. check 1.18 SDK
			`exec: go1.18 download`,                        // 6. download 1.18 SDK
			`call: bin.Remove("go")`,                       // 7. remove old symlink
			`call: bin.Symlink("go1.18", "go")`,            // 8. create new symlink
		})
	})

	t.Run("switch to current version", func(t *testing.T) {
		var steps []string
		var buf bytes.Buffer

		app := app.App{
			GoBin: spyFS{
				dir:   "bin",
				link:  "/path/to/go1.18",
				files: []string{"go1.18"},
				calls: &steps,
			},
			SDK: spyFS{
				dir:   "sdk",
				files: []string{"go1.18/.unpacked-success"},
				calls: &steps,
			},
			Output: &buf,
		}
		recordCmds(&app, &steps, "go version go1.20")

		err := app.Use(context.Background(), "1.18")
		assert.NoErr[F](t, err)
		assert.Equal[E](t, buf.String(), "1.18 is already in use\n")
		assert.Equal[E](t, steps, []string{
			`exec: go version`,         // 1. read main version
			`call: bin.Readlink("go")`, // 2. read current version
			`call: bin.ReadDir(".")`,   // 3. read installed versions
		})
	})

	t.Run("switch to main version", func(t *testing.T) {
		var steps []string
		var buf bytes.Buffer

		app := app.App{
			GoBin: spyFS{
				dir:   "bin",
				link:  "/path/to/go1.18",
				files: []string{"go1.18"},
				calls: &steps,
			},
			SDK: spyFS{
				dir:   "sdk",
				files: []string{"go1.18/.unpacked-success"},
				calls: &steps,
			},
			Output: &buf,
		}
		recordCmds(&app, &steps, "go version go1.20")

		err := app.Use(context.Background(), "main")
		assert.NoErr[F](t, err)
		assert.Equal[E](t, buf.String(), "Switched to 1.20 (main)\n")
		assert.Equal[E](t, steps, []string{
			`exec: go version`,         // 1. read main version
			`call: bin.Readlink("go")`, // 2. read current version
			`call: bin.ReadDir(".")`,   // 3. read installed versions
			`call: bin.Remove("go")`,   // 4. remove symlink (switch to main)
		})
	})
}

func TestApp_List(t *testing.T) {
	t.Run("list local versions", func(t *testing.T) {
		var steps []string
		var buf bytes.Buffer

		app := app.App{
			GoBin: spyFS{
				dir:   "bin",
				link:  "/path/to/go1.18",
				files: []string{"go1.18", "go1.19"},
				calls: &steps,
			},
			SDK: spyFS{
				dir:   "sdk",
				files: []string{"go1.18/.unpacked-success"}, // 1.19 SDK is missing.
				calls: &steps,
			},
			Output: &buf,
		}
		recordCmds(&app, &steps, "go version go1.20")

		err := app.List(context.Background(), false, "")
		assert.NoErr[F](t, err)
		assert.Equal[E](t, "\n"+buf.String(), `
  1.20 (main)
  1.19 (missing SDK)
* 1.18
`)
		assert.Equal[E](t, steps, []string{
			`exec: go version`,                           // 1. read main version
			`call: bin.Readlink("go")`,                   // 2. read current version
			`call: bin.ReadDir(".")`,                     // 3. read installed versions
			`call: sdk.Stat("go1.19/.unpacked-success")`, // 4. check 1.19 SDK
			`call: sdk.Stat("go1.18/.unpacked-success")`, // 5. check 1.18 SDK
		})
	})

	t.Run("list remote versions", func(t *testing.T) {
		var steps []string
		var buf bytes.Buffer

		app := app.App{
			GoBin: spyFS{
				dir:   "bin",
				link:  "/path/to/go1.18",
				files: []string{"go1.18"},
				calls: &steps,
			},
			SDK: spyFS{
				dir:   "sdk",
				files: []string{"go1.18/.unpacked-success"},
				calls: &steps,
			},
			Output: &buf,
			Requester: httpSpy{
				requests: &steps,
				response: `[{"version":"1.20"},{"version":"1.19"},{"version":"1.18"}]`,
			},
		}
		recordCmds(&app, &steps, "go version go1.20")

		err := app.List(context.Background(), true, "")
		assert.NoErr[F](t, err)
		assert.Equal[E](t, "\n"+buf.String(), `
  tip  (not installed)
  1.20 (main)
  1.19 (not installed)
* 1.18
`)
		assert.Equal[E](t, steps, []string{
			`exec: go version`,                               // 1. read main version
			`call: bin.Readlink("go")`,                       // 2. read current version
			`call: bin.ReadDir(".")`,                         // 3. read installed versions
			`http: https://go.dev/dl/?mode=json&include=all`, // 4. get remote versions
			`call: sdk.Stat("go1.18/.unpacked-success")`,     // 5. check 1.18 SDK
		})
	})
}

func TestApp_Remove(t *testing.T) {
	t.Run("remove existing version", func(t *testing.T) {
		var steps []string

		app := app.App{
			GoBin: spyFS{
				dir:   "bin",
				link:  "/path/to/go1.18",
				files: []string{"go1.18"},
				calls: &steps,
			},
			SDK: spyFS{
				dir:   "sdk",
				files: []string{"go1.18/.unpacked-success"},
				calls: &steps,
			},
			Output: io.Discard,
		}
		recordCmds(&app, &steps, "go version go1.20")

		err := app.Remove(context.Background(), "1.18")
		assert.NoErr[F](t, err)
		assert.Equal[E](t, steps, []string{
			`exec: go version`,              // 1. read main version
			`call: bin.Readlink("go")`,      // 2. read current version
			`call: bin.ReadDir(".")`,        // 3. read installed versions
			`call: bin.Remove("go")`,        // 4. remove symlink (switch to main)
			`call: bin.Remove("go1.18")`,    // 4. remove 1.18 binary
			`call: sdk.RemoveAll("go1.18")`, // 5. remove 1.18 SDK
		})
	})

	t.Run("remove non-existing version", func(t *testing.T) {
		var steps []string

		app := app.App{
			GoBin: spyFS{
				dir:   "bin",
				link:  "/path/to/go1.18",
				files: []string{"go1.18"},
				calls: &steps,
			},
			SDK: spyFS{
				dir:   "sdk",
				files: []string{"go1.18/.unpacked-success"},
				calls: &steps,
			},
			Output: io.Discard,
		}
		recordCmds(&app, &steps, "go version go1.20")

		err := app.Remove(context.Background(), "1.19")
		assert.Equal[F](t, err.Error(), "1.19 is not installed")
		assert.Equal[E](t, steps, []string{
			`exec: go version`,         // 1. read main version
			`call: bin.Readlink("go")`, // 2. read current version
			`call: bin.ReadDir(".")`,   // 3. read installed versions
		})
	})
}

func recordCmds(app *app.App, cmds *[]string, cmdOut string) {
	app.RunCmd = func(ctx context.Context, name string, args ...string) error {
		*cmds = append(*cmds, fmt.Sprintf("exec: %s %s", name, strings.Join(args, " ")))
		return nil
	}
	app.RunCmdOut = func(ctx context.Context, name string, args ...string) (string, error) {
		*cmds = append(*cmds, fmt.Sprintf("exec: %s %s", name, strings.Join(args, " ")))
		return cmdOut, nil
	}
}

type spyFS struct {
	dir   string
	link  string
	files []string
	calls *[]string
}

func (s spyFS) Open(name string) (fs.File, error) { panic("unimplemented") }

func (s spyFS) Stat(name string) (fs.FileInfo, error) {
	*s.calls = append(*s.calls, fmt.Sprintf("call: %s.Stat(%q)", s.dir, name))
	if slices.Contains(s.files, name) {
		return nil, nil
	}
	return nil, fs.ErrNotExist
}

func (s spyFS) Remove(name string) error {
	*s.calls = append(*s.calls, fmt.Sprintf("call: %s.Remove(%q)", s.dir, name))
	return nil
}

func (s spyFS) RemoveAll(name string) error {
	*s.calls = append(*s.calls, fmt.Sprintf("call: %s.RemoveAll(%q)", s.dir, name))
	return nil
}

func (s spyFS) Symlink(oldname, newname string) error {
	*s.calls = append(*s.calls, fmt.Sprintf("call: %s.Symlink(%q, %q)", s.dir, oldname, newname))
	return nil
}

func (s spyFS) Readlink(name string) (string, error) {
	*s.calls = append(*s.calls, fmt.Sprintf("call: %s.Readlink(%q)", s.dir, name))
	if s.link == "" {
		return "", fs.ErrNotExist
	}
	return s.link, nil
}

func (s spyFS) ReadDir(name string) ([]fs.DirEntry, error) {
	*s.calls = append(*s.calls, fmt.Sprintf("call: %s.ReadDir(%q)", s.dir, name))
	entries := make([]fs.DirEntry, len(s.files))
	for i, f := range s.files {
		entries[i] = dirFile(f)
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

func (s httpSpy) Do(req *http.Request) (*http.Response, error) {
	*s.requests = append(*s.requests, "http: "+req.URL.String())
	return &http.Response{
		Body: io.NopCloser(strings.NewReader(s.response)),
	}, nil
}
