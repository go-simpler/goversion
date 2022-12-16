// https://go.dev/doc/manage-install
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
	"runtime"
)

var Version = "dev" // injected at build time.

func main() {
	if err := run(); err != nil {
		var exitErr *exec.ExitError

		switch {
		case errors.Is(err, flag.ErrHelp):
			printf(usage)
			os.Exit(0)
		case errors.As(err, new(usageError)):
			printf("Error: %v\n\n%s", err, usage)
			os.Exit(2)
		case errors.As(err, &exitErr):
			code := exitErr.ExitCode()
			os.Exit(code)
		default:
			printf("Error: %v\n", err)
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
		printf("goversion %s %s/%s (built by %s)\n", Version, runtime.GOOS, runtime.GOARCH, runtime.Version())
		return nil
	}

	args := fset.Args()
	if len(args) == 0 {
		return usageError{errors.New("no command has been specified")}
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	app, err := newApp(ctx)
	if err != nil {
		return err
	}

	switch cmd := args[0]; cmd {
	case "use":
		return app.use(ctx, args[1:])
	case "ls":
		return app.list(ctx, args[1:])
	case "rm":
		return app.remove(ctx, args[1:])
	default:
		return usageError{fmt.Errorf("unknown command %q", cmd)}
	}
}

func printf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format, args...)
}

const usage = `Usage: goversion [flags] <command> [command flags]

Commands:

	use <version>        switch the current Go version (will be installed if not already exists)

	ls                   print the list of installed Go versions
	    -a (-all)        print available versions from go.dev as well
	    -only=<prefix>   print only versions starting with this prefix

	rm <version>         remove the specified Go version (both the binary and the SDK)

Flags:

	-h (-help)           print this message and quit
	-v (-version)        print the version of goversion itself and quit
`

type usageError struct{ err error }

func (e usageError) Error() string { return e.err.Error() }
func (e usageError) Unwrap() error { return e.err }
