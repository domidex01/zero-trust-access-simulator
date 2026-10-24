package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"zt-sim/internal/engine"
	"zt-sim/internal/model"
	"zt-sim/internal/policies"
	"zt-sim/internal/report"
)

const version = "0.1.0"

const usage = `usage: zt-sim <command> [flags]

commands:
  decide <signals.json>   evaluate a JSON array of signals, print decisions
  decide --stdin          read one signal object from stdin
  run <scenarios-dir>     run scenario packs, print summary
  policies                list loaded policies
  audit <audit.jsonl>     print + validate an audit log

flags on decide:
  --format table|json     output format (default table)

global:
  --version               print version and exit
`

const decideUsage = "usage: zt-sim decide [--stdin] [--format table|json] <signals.json>"

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 64
	}
	switch args[0] {
	case "--version":
		fmt.Fprintf(stdout, "zt-sim version %s\n", version)
		return 0
	case "decide":
		return cmdDecide(args[1:], stdin, stdout, stderr)
	case "run", "policies", "audit":
		fmt.Fprintf(stderr, "%s: not implemented yet\n", args[0])
		return 70
	default:
		fmt.Fprintf(stderr, "unknown subcommand %q\n", args[0])
		fmt.Fprint(stderr, usage)
		return 64
	}
}

// cmdDecide evaluates one signal (--stdin) or a file's array of signals
// through the v1 pack and prints the decisions. Exit code = strictest
// action (§2: severity doubles as the exit code).
func cmdDecide(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("decide", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprintln(stderr, decideUsage) }
	useStdin := fs.Bool("stdin", false, "read one signal object from stdin")
	format := fs.String("format", "table", "output format: table|json")
	if err := fs.Parse(args); err != nil {
		return 64
	}
	rest := fs.Args()
	if *format != "table" && *format != "json" {
		fmt.Fprintln(stderr, decideUsage)
		return 64
	}
	if *useStdin {
		if len(rest) > 0 {
			fmt.Fprintln(stderr, decideUsage)
			return 64
		}
	} else if len(rest) != 1 {
		fmt.Fprintln(stderr, decideUsage)
		return 64
	}

	var signals []model.Signal
	if *useStdin {
		s, err := model.DecodeSignal(stdin)
		if err != nil {
			fmt.Fprintf(stderr, "decide: stdin: %v\n", err)
			return 65
		}
		signals = []model.Signal{s}
	} else {
		f, err := os.Open(rest[0])
		if err != nil {
			fmt.Fprintf(stderr, "decide: %v\n", err)
			return 65
		}
		defer f.Close()
		signals, err = model.DecodeSignals(f)
		if err != nil {
			fmt.Fprintf(stderr, "decide: %s: %v\n", rest[0], err)
			return 65
		}
	}

	e := engine.New(policies.Pack())
	rows := make([]report.Row, len(signals))
	decisions := make([]model.Decision, len(signals))
	strictest := 0
	for i, s := range signals {
		d := e.Evaluate(s)
		rows[i] = report.Row{Signal: s, Decision: d}
		decisions[i] = d
		strictest = max(strictest, int(d.Action))
	}

	var err error
	if *format == "json" {
		err = report.DecisionsJSON(stdout, decisions)
	} else {
		err = report.Decisions(stdout, rows)
	}
	if err != nil {
		fmt.Fprintf(stderr, "decide: %v\n", err)
		return 65
	}
	return strictest
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
