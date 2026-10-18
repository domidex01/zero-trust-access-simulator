package main

import (
	"fmt"
	"io"
	"os"
)

const version = "0.1.0"

const usage = `usage: zt-sim <command> [flags]

commands:
  decide <signals.json>   evaluate a JSON array of signals, print decisions
  run <scenarios-dir>     run scenario packs, print summary
  policies                list loaded policies
  audit <audit.jsonl>     print + validate an audit log

global:
  --version               print version and exit
`

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 64
	}
	switch args[0] {
	case "--version":
		fmt.Fprintf(stdout, "zt-sim version %s\n", version)
		return 0
	case "decide", "run", "policies", "audit":
		fmt.Fprintf(stderr, "%s: not implemented yet\n", args[0])
		return 70
	default:
		fmt.Fprintf(stderr, "unknown subcommand %q\n", args[0])
		fmt.Fprint(stderr, usage)
		return 64
	}
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
