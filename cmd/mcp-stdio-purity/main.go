package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/kentomk/mcp-stdio-purity/internal/checker"
)

var version = "dev"

const helpText = `mcp-stdio-purity checks a real stdio MCP server for non-JSON-RPC stdout.

Usage:
  mcp-stdio-purity check [flags] -- COMMAND [ARG...]
  mcp-stdio-purity version
  mcp-stdio-purity help

Check flags:
  --format text|json
  --timeout DURATION
  --cleanup-grace DURATION
  --max-line-bytes N
  --max-stdout-bytes N
  --max-diagnostics N

Exit codes:
  0  stdout stayed JSON-RPC-pure
  1  one or more MSP001 purity violations were found
  2  invalid arguments or an operational failure
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 && (args[0] == "help" || args[0] == "--help" || args[0] == "-h") {
		fmt.Fprint(stdout, helpText)
		return 0
	}
	if len(args) == 1 && args[0] == "version" {
		fmt.Fprintln(stdout, version)
		return 0
	}
	if len(args) == 0 || args[0] != "check" {
		fmt.Fprintln(stderr, "usage: mcp-stdio-purity check [flags] -- COMMAND [ARG...]")
		return 2
	}
	if len(args) == 2 && (args[1] == "--help" || args[1] == "-h") {
		fmt.Fprint(stdout, helpText)
		return 0
	}

	flags := flag.NewFlagSet("check", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	format := flags.String("format", "text", "text or json")
	timeout := flags.Duration("timeout", 10*time.Second, "overall lifecycle timeout")
	cleanupGrace := flags.Duration("cleanup-grace", 250*time.Millisecond, "time to observe descendant cleanup output")
	maxLine := flags.Int("max-line-bytes", 1024*1024, "maximum stdout record size")
	maxTotal := flags.Int("max-stdout-bytes", 16*1024*1024, "maximum total stdout size")
	maxDiagnostics := flags.Int("max-diagnostics", 20, "maximum reported purity diagnostics")
	if err := flags.Parse(args[1:]); err != nil || (*format != "text" && *format != "json") {
		fmt.Fprintln(stderr, "invalid check arguments")
		return 2
	}

	config := checker.DefaultConfig(flags.Args())
	config.Timeout = *timeout
	config.CleanupGrace = *cleanupGrace
	config.MaxLineBytes = *maxLine
	config.MaxStdoutBytes = *maxTotal
	config.MaxDiagnostics = *maxDiagnostics
	config.Stderr = stderr
	config.ToolVersion = version
	report := checker.Run(context.Background(), config)
	if *format == "json" {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			fmt.Fprintln(stderr, "encode report")
			return 2
		}
	} else {
		fmt.Fprint(stdout, checker.FormatText(report))
	}
	return checker.ExitCode(report)
}
