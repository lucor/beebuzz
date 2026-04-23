package main

import (
	"context"
	"flag"
	"fmt"
	"io"
)

// runConnectCommand parses and executes the connect command.
func (a *App) runConnectCommand(args []string) error {
	flagSet := flag.NewFlagSet(commandConnect, flag.ContinueOnError)
	flagSet.SetOutput(a.stderr)
	flagSet.Usage = func() {
		writeConnectUsage(flagSet.Output())
	}

	apiURL := flagSet.String("api-url", "", "BeeBuzz API URL, for example https://api.beebuzz.app")
	apiToken := flagSet.String("api-token", "", "BeeBuzz API token")
	profile := flagSet.String("profile", "", "Profile name")

	if err := flagSet.Parse(args); err != nil {
		return err
	}
	if len(flagSet.Args()) != 0 {
		return fmt.Errorf("%w: %s does not accept positional arguments", ErrUsage, commandConnect)
	}

	return a.runConnect(*apiURL, *apiToken, *profile)
}

// runConnect collects config values, verifies them against the server, and saves the config file.
func (a *App) runConnect(apiURL, apiToken, profile string) error {
	resolvedProfile, err := a.profileService.resolveProfile(profile)
	if err != nil {
		return err
	}

	config, err := buildSetupConfig(a.stdin, a.stdout, a.profileService.setupDefaults(), apiURL, apiToken)
	if err != nil {
		return err
	}
	resolvedAPIURL, deviceKeys, err := resolveAPIURL(context.Background(), a.httpClient, config.APIURL, config.APIToken)
	if err != nil {
		return err
	}
	config.APIURL = resolvedAPIURL
	config.DeviceKeys = deviceKeys
	config.Profile = resolvedProfile

	if err := a.profileService.saveResolvedConfig(config); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(a.stdout, "connected profile %q to %s\n", resolvedProfile, config.APIURL); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	return nil
}

// writeConnectUsage prints connect subcommand usage.
func writeConnectUsage(output io.Writer) {
	_, _ = fmt.Fprint(output, `Usage:
  beebuzz connect [--api-url URL] [--api-token TOKEN] [--profile NAME]

Connect or update this CLI to BeeBuzz. It can refresh the token and paired device keys for an existing profile.

Flags:
  --api-url string
        BeeBuzz API URL (default "https://api.beebuzz.app")
  --api-token string
        BeeBuzz API token
  --profile string
        Profile name (default "default")

If flags are omitted, connect prompts interactively.
`)
}
