// Package main implements the BeeBuzz HTTP API server.
package main

import (
	"fmt"
	"io"
	"os"
)

// commitHash is set at build time via -ldflags "-X main.commitHash=<commit-sha>".
var commitHash = "dev"

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "beebuzz server failed: %v\n", err)
		os.Exit(1)
	}
}

// run dispatches explicit server subcommands.
func run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		writeUsage(stderr)
		return fmt.Errorf("missing subcommand")
	}

	switch args[0] {
	case "serve":
		if len(args) != 1 {
			writeUsage(stderr)
			return fmt.Errorf("serve does not accept additional arguments")
		}
		return runServe()
	case "healthcheck":
		if len(args) != 1 {
			writeUsage(stderr)
			return fmt.Errorf("healthcheck does not accept additional arguments")
		}
		return runHealthcheck()
	case "vapid":
		if len(args) == 2 && args[1] == "generate" {
			return runGenerateVAPID(stdout)
		}
		writeUsage(stderr)
		return fmt.Errorf("unknown vapid subcommand")
	default:
		writeUsage(stderr)
		return fmt.Errorf("unknown subcommand: %s", args[0])
	}
}

func writeUsage(output io.Writer) {
	_, _ = fmt.Fprint(output, `Usage:
  beebuzz-server serve
  beebuzz-server healthcheck
  beebuzz-server vapid generate
`)
}
