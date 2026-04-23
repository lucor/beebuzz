package main

import (
	"context"
	"flag"
	"fmt"
	"io"
)

// runKeysCommand parses and executes the keys subcommand.
func (a *App) runKeysCommand(args []string) error {
	flagSet := flag.NewFlagSet(commandKeys, flag.ContinueOnError)
	flagSet.SetOutput(a.stderr)
	flagSet.Usage = func() {
		writeKeysUsage(flagSet.Output())
	}

	apiURL := flagSet.String("api-url", "", "BeeBuzz API URL")
	profile := flagSet.String("profile", "", "Profile name")

	if err := flagSet.Parse(args); err != nil {
		return err
	}
	if len(flagSet.Args()) != 0 {
		return fmt.Errorf("%w: %s does not accept positional arguments", ErrUsage, commandKeys)
	}

	return a.runKeys(*apiURL, *profile)
}

// runKeys refreshes the cached device age keys from the BeeBuzz server.
func (a *App) runKeys(apiURL, profile string) error {
	config, err := a.profileService.resolveConfig(profile, apiURL)
	if err != nil {
		return err
	}
	previousKeys := append([]DeviceKey(nil), config.DeviceKeys...)

	if err := refreshKeys(context.Background(), a.httpClient, config); err != nil {
		return err
	}
	if err := a.profileService.saveResolvedConfig(config); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(a.stdout, "synced %d paired device keys\n", len(config.DeviceKeys)); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	if err := writeKeyRefreshSummary(a.stdout, previousKeys, config.DeviceKeys); err != nil {
		return err
	}

	return nil
}

// writeKeysUsage prints keys subcommand usage.
func writeKeysUsage(output io.Writer) {
	_, _ = fmt.Fprint(output, `Usage:
  beebuzz keys [--api-url URL] [--profile NAME]

Sync paired device keys from the BeeBuzz server.

Flags:
  --api-url string
        BeeBuzz API URL override
  --profile string
        Profile name
`)
}
