package main

import (
	"errors"
	"fmt"
	"os"
	"time"
)

// version is set at build time via -ldflags. Falls back to "dev" for local builds.
var version = "dev"

const (
	httpTimeout   = 15 * time.Second
	defaultAPIURL = "https://api.beebuzz.app"
)

// ErrUsage indicates a CLI usage error (invalid args, unknown command).
var ErrUsage = errors.New("usage error")

// ErrUsage indicates a CLI usage error (invalid args, unknown command).
var ErrUnknownCmd = errors.New("unknown command")

// ErrNoArgs indicates no command was provided.
var ErrNoArgs = errors.New("no args")

// main is the BeeBuzz CLI entrypoint.
func main() {
	app := NewApp(os.Stdin, os.Stdout, os.Stderr)
	err := app.Run(os.Args[1:])
	if err == nil {
		return
	}
	if errors.Is(err, ErrNoArgs) {
		os.Exit(2)
	}
	fmt.Fprintln(os.Stderr, err)
	if errors.Is(err, ErrUnknownCmd) {
		os.Exit(2)
	}
	os.Exit(1)
}
