package main

import (
	"flag"
	"fmt"
	"io"
)

const (
	commandProfileList       = "list"
	commandProfileShow       = "show"
	commandProfileUse        = "use"
	commandProfileDelete     = "delete"
)

// writeProfileUsage prints profile subcommand usage.
func writeProfileUsage(output io.Writer) {
	_, _ = fmt.Fprint(output, `Usage:
  beebuzz profile [command]

Commands:
  list          List profiles
  show <name>   Show profile configuration (token masked)
  use <name>    Set the default profile
  delete <name>  Delete a profile

Run "beebuzz profile <command> --help" for command-specific help.
`)
}

// writeProfileListUsage prints profile list subcommand usage.
func writeProfileListUsage(output io.Writer) {
	_, _ = fmt.Fprint(output, `Usage:
  beebuzz profile list

Lists all available profiles.
`)
}

// writeProfileShowUsage prints profile show subcommand usage.
func writeProfileShowUsage(output io.Writer) {
	_, _ = fmt.Fprint(output, `Usage:
  beebuzz profile show <name>

Shows configuration for a profile. The API token is masked for security.

Arguments:
  name    Profile name to show
`)
}

// writeProfileUseUsage prints profile use subcommand usage.
func writeProfileUseUsage(output io.Writer) {
	_, _ = fmt.Fprint(output, `Usage:
  beebuzz profile use <name>

Sets the default profile. The profile must exist.

Arguments:
  name    Profile name to set as default
`)
}

// writeProfileDeleteUsage prints profile delete subcommand usage.
func writeProfileDeleteUsage(output io.Writer) {
	_, _ = fmt.Fprint(output, `Usage:
  beebuzz profile delete <name>

Deletes a profile. Cannot delete the current default profile.

Arguments:
  name    Profile name to delete
`)
}

// runProfileCommand dispatches profile subcommands.
func (a *App) runProfileCommand(args []string) error {
	if len(args) == 0 {
		return a.runProfileShowDefault()
	}

	switch args[0] {
	case commandProfileList:
		return a.runProfileList(args[1:])
	case commandProfileShow:
		return a.runProfileShow(args[1:])
	case commandProfileUse:
		return a.runProfileUse(args[1:])
	case commandProfileDelete:
		return a.runProfileDelete(args[1:])
	case "--help", "-h":
		writeProfileUsage(a.stderr)
		return flag.ErrHelp
	default:
		return fmt.Errorf("%w: unknown profile command: %s", ErrUsage, args[0])
	}
}

// runProfileList executes "beebuzz profile list".
func (a *App) runProfileList(args []string) error {
	flagSet := flag.NewFlagSet(commandProfileList, flag.ContinueOnError)
	flagSet.SetOutput(a.stderr)
	flagSet.Usage = func() {
		writeProfileListUsage(flagSet.Output())
	}

	if err := flagSet.Parse(args); err != nil {
		return err
	}
	if len(flagSet.Args()) != 0 {
		return fmt.Errorf("%w: %s does not accept positional arguments", ErrUsage, commandProfileList)
	}

	profiles, err := a.profileService.list()
	if err != nil {
		return err
	}

	defaultProfile, err := a.profileService.resolveDefaultProfileName()
	if err != nil {
		return err
	}

	for _, p := range profiles {
		if p == defaultProfile {
			_, _ = fmt.Fprintf(a.stdout, "%s (default)\n", p)
		} else {
			_, _ = fmt.Fprintln(a.stdout, p)
		}
	}

	return nil
}

// runProfileShow executes "beebuzz profile show <name>".
func (a *App) runProfileShow(args []string) error {
	flagSet := flag.NewFlagSet(commandProfileShow, flag.ContinueOnError)
	flagSet.SetOutput(a.stderr)
	flagSet.Usage = func() {
		writeProfileShowUsage(flagSet.Output())
	}

	if err := flagSet.Parse(args); err != nil {
		return err
	}
	if len(flagSet.Args()) != 1 {
		return fmt.Errorf("%w: usage: beebuzz profile show <name>", ErrUsage)
	}

	name := flagSet.Args()[0]

	config, err := a.profileService.show(name)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(a.stdout, "Profile: %s\n", name)
	_, _ = fmt.Fprintf(a.stdout, "API URL: %s\n", config.APIURL)
	_, _ = fmt.Fprintf(a.stdout, "API Token: %s\n", config.APIToken)
	_, _ = fmt.Fprintf(a.stdout, "Device Keys: %d\n", len(config.DeviceKeys))

	return nil
}

// runProfileUse executes "beebuzz profile use <name>".
func (a *App) runProfileUse(args []string) error {
	flagSet := flag.NewFlagSet(commandProfileUse, flag.ContinueOnError)
	flagSet.SetOutput(a.stderr)
	flagSet.Usage = func() {
		writeProfileUseUsage(flagSet.Output())
	}

	if err := flagSet.Parse(args); err != nil {
		return err
	}
	if len(flagSet.Args()) != 1 {
		return fmt.Errorf("%w: usage: beebuzz profile %s <name>", ErrUsage, commandProfileUse)
	}

	name := flagSet.Args()[0]

	if err := a.profileService.setDefault(name); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(a.stdout, "profile %q is now the default\n", name)
	return nil
}

// runProfileDelete executes "beebuzz profile delete <name>".
func (a *App) runProfileDelete(args []string) error {
	flagSet := flag.NewFlagSet(commandProfileDelete, flag.ContinueOnError)
	flagSet.SetOutput(a.stderr)
	flagSet.Usage = func() {
		writeProfileDeleteUsage(flagSet.Output())
	}

	if err := flagSet.Parse(args); err != nil {
		return err
	}
	if len(flagSet.Args()) != 1 {
		return fmt.Errorf("%w: usage: beebuzz profile delete <name>", ErrUsage)
	}

	name := flagSet.Args()[0]

	if err := a.profileService.delete(name); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(a.stdout, "deleted profile %q\n", name)
	return nil
}

// runProfileShowDefault shows current default profile.
func (a *App) runProfileShowDefault() error {
	defaultProfile, err := a.profileService.resolveDefaultProfileName()
	if err != nil {
		return err
	}

	profiles, err := a.profileService.list()
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(a.stdout, "Default profile: %s\n\nAvailable profiles:\n", defaultProfile)

	for _, p := range profiles {
		if p == defaultProfile {
			_, _ = fmt.Fprintf(a.stdout, "  %s (default)\n", p)
		} else {
			_, _ = fmt.Fprintln(a.stdout, "  "+p)
		}
	}

	return nil
}
