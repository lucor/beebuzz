package main

import (
	"fmt"
	"io"
)

// writeUsage prints the top-level CLI usage text.
func writeUsage(output io.Writer) {
	_, _ = fmt.Fprint(output, `Usage:
  beebuzz <command> [flags] [arguments]

Core commands:
  connect  Connect this CLI to BeeBuzz
  send     Send a message

Advanced commands:
  keys     Sync paired device keys
  profile  Manage profiles
  version  Print version information

Run "beebuzz <command> --help" for command-specific help.

Environment:
  BEEBUZZ_API_URL
        API URL override for connect, keys, and send commands
  BEEBUZZ_API_TOKEN
        API token override for connect, keys, and send commands
`)
}

// writeVersionUsage prints version subcommand usage.
func writeVersionUsage(output io.Writer) {
	_, _ = fmt.Fprint(output, `Usage:
  beebuzz version
`)
}
