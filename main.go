// https://go.dev/doc/manage-install
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
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

type usageError struct{ error }

func usage() {
	fmt.Printf("usage: TODO\n")
}
