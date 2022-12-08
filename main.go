// https://go.dev/doc/manage-install
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
)

func main() {
	if err := run(); err != nil {
		if exitErr := (*exec.ExitError)(nil); errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		printf("Error: %v\n", err)
		if errors.As(err, new(usageError)) {
			printf(usage)
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
	case "ls":
		return app.list(ctx, args)
	case "rm":
		return app.remove(ctx, args)
	case "help":
		printf(usage)
		return nil
	default:
		return usageError{fmt.Errorf("unknown command %q", cmd)}
	}
}

const usage = `
Usage: goversion <command> [command flags]

Commands:

	use <version>   switch the current Go version (will be installed if not already exists)

	ls              print the list of installed Go versions
	    -a (-all)   print available versions from go.dev as well

	rm <version>    remove the specified Go version (both the binary and the SDK)

	help            print this message and quit
`

type usageError struct{ error }

func printf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format, args...)
}
