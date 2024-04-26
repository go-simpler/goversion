// The goversion tool implements switching between multiple Go versions.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"time"

	"go-simpler.org/goversion/app"
	"go-simpler.org/goversion/fsx"
)

const usage = `Usage: goversion [flags] <command> [command flags]

Commands:
    use main              switch to the main Go version
    use <version>         switch to the specified Go version (will be installed if not exists)
    ls                    print the list of installed Go versions
        -a (-all)         print also available versions from go.dev
        -only=<prefix>    print only versions starting with the prefix
        -only=latest      print only the latest patch for each version
    rm <version>          remove the specified Go version (both binary and SDK)

Flags:
    -h (-help)            print this message and quit
    -v (-version)         print the version of goversion itself and quit
`

var version = "dev" // injected at build time.

func main() {
	if err := run(); err != nil {
		var exitErr *exec.ExitError

		switch {
		case errors.Is(err, flag.ErrHelp):
			fmt.Printf("%s", usage)
			os.Exit(0)
		case errors.As(err, &exitErr):
			code := exitErr.ExitCode()
			os.Exit(code)
		case errors.As(err, new(usageError)):
			fmt.Printf("Error: %v\n\n%s", err, usage)
			os.Exit(2)
		default:
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	}
}

func run() error {
	fset := flag.NewFlagSet("", flag.ContinueOnError)
	fset.SetOutput(io.Discard)

	var printVersion bool
	fset.BoolVar(&printVersion, "v", false, "")
	fset.BoolVar(&printVersion, "version", false, "")

	if err := fset.Parse(os.Args[1:]); err != nil {
		return usageError{err}
	}

	if printVersion {
		fmt.Printf("goversion version %s %s/%s\n", version, runtime.GOOS, runtime.GOARCH)
		return nil
	}

	args := fset.Args()
	if len(args) == 0 {
		return usageError{errors.New("no command has been specified")}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	gobin, ok := os.LookupEnv("GOBIN")
	if !ok {
		gobin = filepath.Join(home, "go", "bin")
		os.Setenv("GOBIN", gobin)
	}

	app := app.App{
		// TODO: make sure it works on Windows;
		// see https://github.com/golang/go/issues/44279 for details.
		GoBin:  fsx.DirFS(gobin),
		SDK:    fsx.DirFS(home, "sdk"), // TODO: update when https://github.com/golang/go/issues/26520 is closed.
		Output: os.Stdout,
		RunCmd: func(ctx context.Context, name string, args ...string) error {
			cmd := exec.CommandContext(ctx, name, args...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stdout
			return cmd.Run()
		},
		RunCmdOut: func(ctx context.Context, name string, args ...string) (string, error) {
			cmd := exec.CommandContext(ctx, name, args...)
			out, err := cmd.Output()
			return string(out), err
		},
		Requester: &http.Client{Timeout: time.Minute},
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	switch cmd, cmdArgs := args[0], args[1:]; cmd {
	case "use":
		if len(cmdArgs) == 0 {
			return usageError{errors.New("no version has been specified")}
		}
		return app.Use(ctx, cmdArgs[0])

	case "ls":
		fset := flag.NewFlagSet("", flag.ContinueOnError)
		fset.SetOutput(io.Discard)

		var printAll bool
		fset.BoolVar(&printAll, "a", false, "")
		fset.BoolVar(&printAll, "all", false, "")

		var printOnly string
		fset.StringVar(&printOnly, "only", "", "")

		if err := fset.Parse(cmdArgs); err != nil {
			return usageError{err}
		}
		return app.List(ctx, printAll, printOnly)

	case "rm":
		if len(cmdArgs) == 0 {
			return usageError{errors.New("no version has been specified")}
		}
		return app.Remove(ctx, cmdArgs[0])

	default:
		return usageError{fmt.Errorf("unknown command %q", cmd)}
	}
}

type usageError struct{ err error }

func (e usageError) Error() string { return e.err.Error() }
func (e usageError) Unwrap() error { return e.err }
