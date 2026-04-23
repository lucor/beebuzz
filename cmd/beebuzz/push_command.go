package main

import (
	"context"
	"flag"
	"fmt"
	"io"

	"lucor.dev/beebuzz/internal/push"
)

// runSendCommand parses and executes the send command.
func (a *App) runSendCommand(args []string) error {
	flagSet := flag.NewFlagSet(commandSend, flag.ContinueOnError)
	flagSet.SetOutput(a.stderr)
	flagSet.Usage = func() {
		writeSendUsage(flagSet.Output())
	}

	var topic, priority, attachmentPath, apiURL, profile string
	flagSet.StringVar(&topic, "topic", push.DefaultTopicName, "Notification topic")
	flagSet.StringVar(&topic, "t", push.DefaultTopicName, "Notification topic")
	flagSet.StringVar(&priority, "priority", push.PriorityNormal, "Push priority: normal or high")
	flagSet.StringVar(&priority, "p", push.PriorityNormal, "Push priority: normal or high")
	flagSet.StringVar(&attachmentPath, "attachment", "", "Attachment file path")
	flagSet.StringVar(&attachmentPath, "a", "", "Attachment file path")
	flagSet.StringVar(&apiURL, "api-url", "", "BeeBuzz API URL override")
	flagSet.StringVar(&profile, "profile", "", "Profile name")

	if err := flagSet.Parse(args); err != nil {
		return err
	}

	pushInput, err := resolvePushInput(flagSet.Args(), a.stdin, topic, priority, attachmentPath, apiURL)
	if err != nil {
		return err
	}
	pushInput.Profile = profile

	return a.runSend(pushInput)
}

// runSend loads config, pushes an E2E notification, and persists updated device keys.
func (a *App) runSend(input PushInput) error {
	config, err := a.profileService.resolveConfig(input.Profile, input.APIURL)
	if err != nil {
		return err
	}
	previousKeys := append([]DeviceKey(nil), config.DeviceKeys...)

	response, err := pushNotification(context.Background(), a.httpClient, config, input)
	if err != nil {
		return err
	}
	config.DeviceKeys = response.DeviceKeys
	if err := a.profileService.saveResolvedConfig(config); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(a.stdout, formatSendSummary(response)); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	if err := writeKeyRefreshSummary(a.stdout, previousKeys, response.DeviceKeys); err != nil {
		return err
	}

	return nil
}

func formatSendSummary(response *PushResponse) string {
	if response == nil {
		return "sent message"
	}

	if response.TotalCount == response.SentCount && response.FailedCount == 0 {
		return fmt.Sprintf("sent message to %d device%s", response.TotalCount, pluralSuffix(response.TotalCount))
	}

	return fmt.Sprintf(
		"sent message to %d of %d device%s (%d failed)",
		response.SentCount,
		response.TotalCount,
		pluralSuffix(response.TotalCount),
		response.FailedCount,
	)
}

func pluralSuffix(count int) string {
	if count == 1 {
		return ""
	}

	return "s"
}

// writeSendUsage prints send subcommand usage.
func writeSendUsage(output io.Writer) {
	_, _ = fmt.Fprint(output, `Usage:
  beebuzz send [flags] <title> [body]

Send a message to BeeBuzz.

Flags:
  -t, --topic string
        Notification topic (default "general")
  -p, --priority string
        Push priority: normal or high (default "normal")
  -a, --attachment string
        Attachment file path
  --api-url string
        BeeBuzz API URL override
  --profile string
        Profile name

Notes:
  - Flags must appear before positional arguments.
  - Use -- to separate flags from titles starting with a dash.
  - If body is omitted and stdin is piped, stdin is used as the body.
  - Title is limited to 64 characters.
  - Body is limited to 256 characters.
  - Use attachments for longer content.
`)
}
