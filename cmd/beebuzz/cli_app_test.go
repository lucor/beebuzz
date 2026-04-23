package main

import (
	"bytes"
	"errors"
	"flag"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestAppRunConnectRejectsPositionalArgs(t *testing.T) {
	app := NewApp(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})

	err := app.Run([]string{commandConnect, "hello"})
	if err == nil {
		t.Fatal("expected connect positional arguments error")
	}
	if !strings.Contains(err.Error(), "connect does not accept positional arguments") {
		t.Fatalf("error: got %q", err)
	}
}

func TestAppRunWithoutArgsReturnsHelpHint(t *testing.T) {
	originalPipedCheck := pipedInputCheck
	defer func() {
		pipedInputCheck = originalPipedCheck
	}()

	pipedInputCheck = func() bool { return false }

	app := NewApp(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	err := app.Run([]string{})
	if err == nil {
		t.Fatal("expected no-args error")
	}
	if !errors.Is(err, ErrNoArgs) {
		t.Fatalf("error: got %v, want ErrNoArgs", err)
	}
}

func TestAppRunConnectWithoutArgsDoesNotHitNoArgsFallback(t *testing.T) {
	originalPipedCheck := pipedInputCheck
	originalBasePath := configBasePath
	defer func() {
		pipedInputCheck = originalPipedCheck
		configBasePath = originalBasePath
	}()

	pipedInputCheck = func() bool { return false }
	configBasePath = func() (string, error) { return t.TempDir(), nil }

	app := NewApp(strings.NewReader("\nbeebuzz_token\n"), &bytes.Buffer{}, &bytes.Buffer{})
	app.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			body := io.NopCloser(strings.NewReader(`{"data":[]}`))
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       body,
				Header:     make(http.Header),
			}, nil
		}),
	}

	err := app.Run([]string{commandConnect})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestAppRunSendUsesConfigMissingConnectHint(t *testing.T) {
	originalPipedCheck := pipedInputCheck
	originalBasePath := configBasePath
	defer func() {
		pipedInputCheck = originalPipedCheck
		configBasePath = originalBasePath
	}()

	pipedInputCheck = func() bool { return false }
	configBasePath = func() (string, error) { return t.TempDir(), nil }

	app := NewApp(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	err := app.Run([]string{commandSend, "Build failed"})
	if err == nil {
		t.Fatal("expected config missing error")
	}
	if !strings.Contains(err.Error(), "run beebuzz connect --profile default") {
		t.Fatalf("error: got %q", err)
	}
}

func TestWriteUsageIncludesCoreCommands(t *testing.T) {
	output := &bytes.Buffer{}

	writeUsage(output)

	usage := output.String()
	for _, want := range []string{
		"connect", "send", "keys", "profile",
		envBeeBuzzAPIURL, envBeeBuzzAPIToken,
		`beebuzz <command> --help`,
		"Core commands:",
		"Advanced commands:",
	} {
		if !strings.Contains(usage, want) {
			t.Fatalf("top-level usage missing %q:\n%s", want, usage)
		}
	}
}

func TestWriteSendUsageIncludesSendDetails(t *testing.T) {
	output := &bytes.Buffer{}

	writeSendUsage(output)

	usage := output.String()
	for _, want := range []string{
		"beebuzz send",
		"Send a message to BeeBuzz.",
		"--topic string",
		"--priority string",
		"--attachment string",
		"--api-url string",
		"stdin is used as the body",
		"Title is limited to 64 characters.",
		"Body is limited to 256 characters.",
		"Use attachments for longer content.",
	} {
		if !strings.Contains(usage, want) {
			t.Fatalf("push usage missing %q:\n%s", want, usage)
		}
	}
}

func TestWriteConnectUsageIncludesConnectDetails(t *testing.T) {
	output := &bytes.Buffer{}

	writeConnectUsage(output)

	usage := output.String()
	for _, want := range []string{
		"beebuzz connect",
		"Connect or update this CLI to BeeBuzz.",
		"--api-url string",
		"--api-token string",
	} {
		if !strings.Contains(usage, want) {
			t.Fatalf("setup usage missing %q:\n%s", want, usage)
		}
	}
}

func TestWriteKeysUsageIncludesKeysDetails(t *testing.T) {
	output := &bytes.Buffer{}

	writeKeysUsage(output)

	usage := output.String()
	for _, want := range []string{
		"beebuzz keys",
		"--api-url string",
		"Sync paired device keys from the BeeBuzz server.",
	} {
		if !strings.Contains(usage, want) {
			t.Fatalf("keys usage missing %q:\n%s", want, usage)
		}
	}
}

func TestWriteProfileUsageIncludesUseDetails(t *testing.T) {
	output := &bytes.Buffer{}

	writeProfileUsage(output)

	usage := output.String()
	for _, want := range []string{
		"beebuzz profile [command]",
		"use <name>",
	} {
		if !strings.Contains(usage, want) {
			t.Fatalf("profile usage missing %q:\n%s", want, usage)
		}
	}
}

func TestWriteVersionUsageIncludesVersionDetails(t *testing.T) {
	output := &bytes.Buffer{}

	writeVersionUsage(output)

	usage := output.String()
	if !strings.Contains(usage, "beebuzz version") {
		t.Fatalf("version usage missing command:\n%s", usage)
	}
}

func TestAppRunHelpReturnsFlagHelp(t *testing.T) {
	stderr := &bytes.Buffer{}
	app := NewApp(strings.NewReader(""), &bytes.Buffer{}, stderr)
	err := app.Run([]string{"--help"})
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("error: got %v, want %v", err, flag.ErrHelp)
	}
	if !strings.Contains(stderr.String(), `beebuzz <command> --help`) {
		t.Fatalf("expected top-level usage, got:\n%s", stderr.String())
	}
}

func TestAppRunVersionPrintsVersion(t *testing.T) {
	stdout := &bytes.Buffer{}
	app := NewApp(strings.NewReader(""), stdout, &bytes.Buffer{})
	err := app.Run([]string{commandVersion})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := stdout.String(); got != "beebuzz "+version+"\n" {
		t.Fatalf("got %q, want %q", got, "beebuzz "+version+"\n")
	}
}

func TestAppRunVersionRejectsPositionalArgs(t *testing.T) {
	app := NewApp(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	err := app.Run([]string{commandVersion, "extra"})
	if err == nil {
		t.Fatal("expected positional args error")
	}
	if !strings.Contains(err.Error(), "version does not accept positional arguments") {
		t.Fatalf("error: got %q", err)
	}
}

func TestAppRunSubcommandHelpReturnsFlagHelp(t *testing.T) {
	for _, cmd := range []string{commandConnect, commandSend, "keys", "version"} {
		t.Run(cmd, func(t *testing.T) {
			stderr := &bytes.Buffer{}
			app := NewApp(strings.NewReader(""), &bytes.Buffer{}, stderr)
			err := app.Run([]string{cmd, "--help"})
			if !errors.Is(err, flag.ErrHelp) {
				t.Fatalf("error: got %v, want %v", err, flag.ErrHelp)
			}
			if !strings.Contains(stderr.String(), "beebuzz "+cmd) {
				t.Fatalf("expected %s-specific usage, got:\n%s", cmd, stderr.String())
			}
		})
	}
}

func TestAppRunSendDashDashSeparatesFlagsFromArgs(t *testing.T) {
	originalPipedCheck := pipedInputCheck
	originalBasePath := configBasePath
	defer func() {
		pipedInputCheck = originalPipedCheck
		configBasePath = originalBasePath
	}()

	pipedInputCheck = func() bool { return false }
	configBasePath = func() (string, error) { return t.TempDir(), nil }

	app := NewApp(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	err := app.Run([]string{commandSend, "--", "--looks-like-a-flag"})
	if err == nil {
		t.Fatal("expected config error, not a flag parse error")
	}
	if strings.Contains(err.Error(), "unknown flag") || strings.Contains(err.Error(), "flag provided but not defined") {
		t.Fatalf("-- did not terminate flag parsing: got %q", err)
	}
}

func TestAppRunProfileUseSetsDefault(t *testing.T) {
	originalBasePath := configBasePath
	defer func() {
		configBasePath = originalBasePath
	}()

	baseDir := t.TempDir()
	store := newProfileStore(func() (string, error) { return baseDir, nil })
	if err := store.saveConfig("work", &Config{APIURL: defaultAPIURL, APIToken: "beebuzz_token"}); err != nil {
		t.Fatalf("saveConfig: %v", err)
	}

	configBasePath = func() (string, error) { return baseDir, nil }

	stdout := &bytes.Buffer{}
	app := NewApp(strings.NewReader(""), stdout, &bytes.Buffer{})
	if err := app.Run([]string{"profile", commandProfileUse, "work"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := stdout.String(); got != "profile \"work\" is now the default\n" {
		t.Fatalf("stdout: got %q", got)
	}
}

func TestAppRunProfileUseHelpUsesFullPath(t *testing.T) {
	stderr := &bytes.Buffer{}
	app := NewApp(strings.NewReader(""), &bytes.Buffer{}, stderr)
	err := app.Run([]string{"profile", commandProfileUse, "--help"})
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("error: got %v, want %v", err, flag.ErrHelp)
	}
	if !strings.Contains(stderr.String(), "beebuzz profile "+commandProfileUse+" <name>") {
		t.Fatalf("expected profile use help path, got:\n%s", stderr.String())
	}
}
