// Command sifty is the single-binary entry point for the Sifty backend: structured collection
// and fit assessment (Stage 1) and application-answer preparation (Stage 2). Browser automation
// and job-board scraping run in TypeScript workers the core invokes; see docs/worker-protocol.md.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/w3con/sifty/internal/version"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

const usage = `sifty — local-first job triage

Usage:
  sifty <command> [flags]

Commands:
  version        print the build version
  serve          start the HTTP API and UI server
  help           print this help

Global flags:
  --data-dir     the local data directory (overrides $SIFTY_DATA_DIR)

The data directory holds candidate evidence, remembered answers, feedback and the local
store. It is resolved from --data-dir, then $SIFTY_DATA_DIR, then config, then ./data.
`

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stdout, usage)
		return 0
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "version", "--version", "-v":
		fmt.Fprintln(stdout, version.Version)
		return 0
	case "help", "--help", "-h":
		fmt.Fprint(stdout, usage)
		return 0
	case "serve":
		return runServe(rest, stdout, stderr)
	case "selfcheck":
		return selfcheck(rest, stdout, stderr)
	default:
		// Commands that take flags share a common --data-dir; parse leniently so an unknown
		// command still reports clearly rather than panicking.
		fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
		fs.SetOutput(stderr)
		_ = fs.String("data-dir", "", "the local data directory")
		if err := fs.Parse(rest); err != nil {
			return 2
		}
		fmt.Fprintf(stderr, "sifty: unknown command %q\n\n%s", cmd, usage)
		return 2
	}
}
