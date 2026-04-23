package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
)

const (
	commandConnect = "connect"
	commandKeys    = "keys"
	commandSend    = "send"
	commandProfile = "profile"
	commandVersion = "version"
)

// App wires CLI dependencies and command dispatch.
type App struct {
	stdin          io.Reader
	stdout         io.Writer
	stderr         io.Writer
	httpClient     *http.Client
	profileService *ProfileService
}

// NewApp creates an App with production dependencies.
func NewApp(stdin io.Reader, stdout, stderr io.Writer) *App {
	return &App{
		stdin:          stdin,
		stdout:         stdout,
		stderr:         stderr,
		httpClient:     &http.Client{Timeout: httpTimeout},
		profileService: defaultProfileService,
	}
}

// Run executes the CLI command selected by the first argument.
func (a *App) Run(args []string) error {
	if len(args) == 0 {
		writeUsage(a.stderr)
		return ErrNoArgs
	}

	switch args[0] {
	case "--help", "-h":
		writeUsage(a.stderr)
		return flag.ErrHelp
	case commandVersion:
		return a.runVersionCommand(args[1:])
	case commandConnect:
		return a.runConnectCommand(args[1:])
	case commandKeys:
		return a.runKeysCommand(args[1:])
	case commandSend:
		return a.runSendCommand(args[1:])
	case commandProfile:
		return a.runProfileCommand(args[1:])
	default:
		writeUsage(a.stderr)
		return fmt.Errorf("beebuzz %s: %w", args[0], ErrUnknownCmd)
	}
}

// runVersionCommand parses and executes the version subcommand.
func (a *App) runVersionCommand(args []string) error {
	flagSet := flag.NewFlagSet(commandVersion, flag.ContinueOnError)
	flagSet.SetOutput(a.stderr)
	flagSet.Usage = func() {
		writeVersionUsage(flagSet.Output())
	}

	if err := flagSet.Parse(args); err != nil {
		return err
	}
	if len(flagSet.Args()) != 0 {
		return fmt.Errorf("%w: %s does not accept positional arguments", ErrUsage, commandVersion)
	}

	_, err := fmt.Fprintf(a.stdout, "beebuzz %s\n", version)
	return err
}
