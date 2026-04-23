package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

const (
	promptAPIToken = "API token: "
)

var (
	termIsTerminal  = term.IsTerminal
	termReadPassword = term.ReadPassword
)

// buildSetupConfig resolves setup values from flags or interactive prompts.
func buildSetupConfig(input io.Reader, output io.Writer, defaults setupDefaults, apiURL, apiToken string) (*Config, error) {
	resolvedAPIURL := strings.TrimSpace(apiURL)
	resolvedAPIToken := strings.TrimSpace(apiToken)

	reader := bufio.NewReader(input)

	if resolvedAPIURL == "" {
		value, err := promptValueWithDefault(input, reader, output, buildAPIURLPrompt(defaults.APIURL), defaults.APIURL, false)
		if err != nil {
			return nil, err
		}
		resolvedAPIURL = value
	}

	if resolvedAPIToken == "" {
		value, err := promptValueWithDefault(input, reader, output, buildAPITokenPrompt(defaults.APIToken), defaults.APIToken, true)
		if err != nil {
			return nil, err
		}
		resolvedAPIToken = value
	}

	config := &Config{
		APIURL:     resolvedAPIURL,
		APIToken:   resolvedAPIToken,
		DeviceKeys: []DeviceKey{},
	}
	config.Normalize()

	if config.APIURL == "" {
		return nil, fmt.Errorf("api_url is required")
	}
	if config.APIToken == "" {
		return nil, fmt.Errorf("api_token is required")
	}

	return config, nil
}

// setupDefaults stores the values shown as interactive setup defaults.
type setupDefaults struct {
	APIURL   string
	APIToken string
}

// buildAPIURLPrompt formats the API URL prompt with the current default.
func buildAPIURLPrompt(defaultValue string) string {
	return fmt.Sprintf("API URL [%s]: ", defaultValue)
}

// buildAPITokenPrompt formats the API token prompt and masks any existing value.
func buildAPITokenPrompt(defaultValue string) string {
	maskedToken := maskToken(defaultValue)
	if maskedToken == "" {
		return promptAPIToken
	}

	return fmt.Sprintf("API token [%s]: ", maskedToken)
}

// promptValue prints a prompt and reads a single line value.
func promptValue(input io.Reader, reader *bufio.Reader, output io.Writer, prompt string, masked bool) (string, error) {
	if reader == nil {
		return "", fmt.Errorf("reader is required")
	}
	if _, err := io.WriteString(output, prompt); err != nil {
		return "", fmt.Errorf("write prompt: %w", err)
	}

	if file, ok := input.(*os.File); ok && isTerminal(file) {
		value, err := readInteractivePrompt(file, output, masked)
		if err != nil {
			return "", err
		}

		return strings.TrimSpace(value), nil
	}

	value, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read prompt value: %w", err)
	}

	return strings.TrimSpace(value), nil
}

// promptValueWithDefault prints a prompt and reads a single line value, falling back to a default when empty.
func promptValueWithDefault(input io.Reader, reader *bufio.Reader, output io.Writer, prompt, defaultValue string, masked bool) (string, error) {
	value, err := promptValue(input, reader, output, prompt, masked)
	if err != nil {
		return "", err
	}
	if value == "" {
		return defaultValue, nil
	}

	return value, nil
}

func readInteractivePrompt(file *os.File, output io.Writer, masked bool) (string, error) {
	fd := int(file.Fd())
	if masked {
		value, err := termReadPassword(fd)
		if err != nil {
			return "", fmt.Errorf("read prompt value: %w", err)
		}
		if _, err := io.WriteString(output, "\n"); err != nil {
			return "", fmt.Errorf("write newline: %w", err)
		}

		return string(value), nil
	}

	reader := bufio.NewReader(file)
	value, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read prompt value: %w", err)
	}

	return value, nil
}

func isTerminal(file *os.File) bool {
	if file == nil {
		return false
	}

	return termIsTerminal(int(file.Fd()))
}
