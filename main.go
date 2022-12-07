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
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("goversion: ")

	if err := run(); err != nil {
		var exitErr *exec.ExitError
		switch {
		case errors.As(err, &exitErr):
			code := exitErr.ExitCode()
			os.Exit(code)
		case errors.As(err, new(usageError)):
			log.Print(err)
			usage()
			os.Exit(2)
		default:
			log.Print(err)
			os.Exit(1)
		}
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

type usageError struct{ error }

func usage() {
	fmt.Printf("usage: TODO\n")
}
