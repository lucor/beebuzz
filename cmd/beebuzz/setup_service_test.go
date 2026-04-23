package main

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

func TestBuildSetupConfigUsesFlagsWithoutPrompting(t *testing.T) {
	output := &bytes.Buffer{}

	config, err := buildSetupConfig(
		bufio.NewReader(strings.NewReader("")),
		output,
		setupDefaults{APIURL: defaultAPIURL},
		"https://api.example.com/",
		" beebuzz_token ",
	)
	if err != nil {
		t.Fatalf("buildSetupConfig: %v", err)
	}

	if config.APIURL != "https://api.example.com" {
		t.Fatalf("APIURL: got %q, want %q", config.APIURL, "https://api.example.com")
	}
	if config.APIToken != "beebuzz_token" {
		t.Fatalf("APIToken: got %q, want %q", config.APIToken, "beebuzz_token")
	}
	if output.Len() != 0 {
		t.Fatalf("unexpected prompt output: %q", output.String())
	}
}

func TestBuildSetupConfigPromptsForMissingValues(t *testing.T) {
	input := "https://api.example.com/\nbeebuzz_token\n"
	output := &bytes.Buffer{}

	config, err := buildSetupConfig(
		bufio.NewReader(strings.NewReader(input)),
		output,
		setupDefaults{APIURL: defaultAPIURL},
		"",
		"",
	)
	if err != nil {
		t.Fatalf("buildSetupConfig: %v", err)
	}

	if config.APIURL != "https://api.example.com" {
		t.Fatalf("APIURL: got %q, want %q", config.APIURL, "https://api.example.com")
	}
	if config.APIToken != "beebuzz_token" {
		t.Fatalf("APIToken: got %q, want %q", config.APIToken, "beebuzz_token")
	}
	expectedPrompts := buildAPIURLPrompt(defaultAPIURL) + promptAPIToken
	if output.String() != expectedPrompts {
		t.Fatalf("prompts: got %q, want %q", output.String(), expectedPrompts)
	}
}

func TestBuildSetupConfigUsesDefaultDomainWhenPromptIsEmpty(t *testing.T) {
	input := "\nbeebuzz_token\n"

	config, err := buildSetupConfig(
		bufio.NewReader(strings.NewReader(input)),
		&bytes.Buffer{},
		setupDefaults{APIURL: defaultAPIURL},
		"",
		"",
	)
	if err != nil {
		t.Fatalf("buildSetupConfig: %v", err)
	}
	if config.APIURL != defaultAPIURL {
		t.Fatalf("APIURL: got %q, want %q", config.APIURL, defaultAPIURL)
	}
}

func TestBuildSetupConfigUsesProvidedDefaultsAsPromptDefaults(t *testing.T) {
	output := &bytes.Buffer{}
	config, err := buildSetupConfig(
		bufio.NewReader(strings.NewReader("\n\n")),
		output,
		setupDefaults{
			APIURL:   "https://api.example.com",
			APIToken: "beebuzz_token",
		},
		"",
		"",
	)
	if err != nil {
		t.Fatalf("buildSetupConfig: %v", err)
	}

	if config.APIURL != "https://api.example.com" {
		t.Fatalf("APIURL: got %q, want %q", config.APIURL, "https://api.example.com")
	}
	if config.APIToken != "beebuzz_token" {
		t.Fatalf("APIToken: got %q, want %q", config.APIToken, "beebuzz_token")
	}
	expectedPrompts := buildAPIURLPrompt("https://api.example.com") + buildAPITokenPrompt("beebuzz_token")
	if output.String() != expectedPrompts {
		t.Fatalf("prompts: got %q, want %q", output.String(), expectedPrompts)
	}
}

func TestBuildSetupConfigUsesMaskedTokenInPromptDefaults(t *testing.T) {
	output := &bytes.Buffer{}
	config, err := buildSetupConfig(
		bufio.NewReader(strings.NewReader("\n\n")),
		output,
		setupDefaults{
			APIURL:   "https://saved.example.com",
			APIToken: "beebuzz_saved_token",
		},
		"",
		"",
	)
	if err != nil {
		t.Fatalf("buildSetupConfig: %v", err)
	}

	if config.APIURL != "https://saved.example.com" {
		t.Fatalf("APIURL: got %q, want %q", config.APIURL, "https://saved.example.com")
	}
	if config.APIToken != "beebuzz_saved_token" {
		t.Fatalf("APIToken: got %q, want %q", config.APIToken, "beebuzz_saved_token")
	}

	expectedPrompts := buildAPIURLPrompt("https://saved.example.com") + buildAPITokenPrompt("beebuzz_saved_token")
	if output.String() != expectedPrompts {
		t.Fatalf("prompts: got %q, want %q", output.String(), expectedPrompts)
	}
}
