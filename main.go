// Goversion is a convenience tool that allows using any Go version as the main one.
// It also provides basic versions management: installing, listing and removing.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
)

var Version = "dev" // injected at build time.

func main() {
	if err := run(); err != nil {
		var exitErr *exec.ExitError

		switch {
		case errors.Is(err, flag.ErrHelp):
			fmt.Fprintf(output, "%s", usage)
			os.Exit(0)
		case errors.As(err, new(usageError)):
			fmt.Fprintf(output, "Error: %v\n\n%s", err, usage)
			os.Exit(2)
		case errors.As(err, &exitErr):
			code := exitErr.ExitCode()
			os.Exit(code)
		default:
			fmt.Fprintf(output, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}

func run() error {
	fset := flag.NewFlagSet("goversion", flag.ContinueOnError)
	fset.SetOutput(io.Discard)

	var printVersion bool
	fset.BoolVar(&printVersion, "v", false, "shorthand for -version")
	fset.BoolVar(&printVersion, "version", false, "print the version of goversion itself and quit")

	if err := fset.Parse(os.Args[1:]); err != nil {
		return usageError{err}
	}

	if printVersion {
		fmt.Fprintf(output, "goversion %s %s/%s\n", Version, runtime.GOOS, runtime.GOARCH)
		return nil
	}

	args := fset.Args()
	if len(args) == 0 {
		return usageError{errors.New("no command has been specified")}
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	gobinDir, ok := os.LookupEnv("GOBIN")
	if !ok {
		gobinDir = filepath.Join(home, "go", "bin")
		os.Setenv("GOBIN", gobinDir)
	}

	// TODO(junk1tm): rewrite when https://github.com/golang/go/issues/26520 is closed.
	sdkDir := filepath.Join(home, "sdk")

	// TODO(junk1tm): make sure it works on Windows
	// (see https://github.com/golang/go/issues/44279).
	gobin, sdk = dirFS(gobinDir), dirFS(sdkDir)

	switch cmd := args[0]; cmd {
	case "use":
		return use(ctx, args[1:])
	case "ls":
		return list(ctx, args[1:])
	case "rm":
		return remove(ctx, args[1:])
	default:
		return usageError{fmt.Errorf("unknown command %q", cmd)}
	}
}

var output io.Writer = os.Stdout

const usage = `Usage: goversion [flags] <command> [command flags]

Commands:

	use <version>        switch the current Go version (will be installed if not already exists)
	use main             switch to the main Go version

	ls                   print the list of installed Go versions
	    -a (-all)        print available versions from go.dev as well
	    -only=<prefix>   print only versions starting with this prefix
	    -only=latest     print only the latest patch for each minor version

	rm <version>         remove the specified Go version (both the binary and the SDK)

Flags:

	-h (-help)           print this message and quit
	-v (-version)        print the version of goversion itself and quit
`

type usageError struct{ err error }

func (e usageError) Error() string { return e.err.Error() }
func (e usageError) Unwrap() error { return e.err }
