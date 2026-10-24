package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"zt-sim/internal/engine"
	"zt-sim/internal/model"
	"zt-sim/internal/policies"
	"zt-sim/internal/report"
	"zt-sim/internal/scenario"
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
	case "run":
		return cmdRun(args[1:], stdout, stderr)
	case "policies":
		return cmdPolicies(args[1:], stdout, stderr)
	case "audit":
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
	// every usage rule in one guard: bad format, extra positional, or
	// stdin xor file violated
	switch {
	case *format != "table" && *format != "json",
		len(rest) > 1,
		*useStdin && len(rest) > 0,
		!*useStdin && len(rest) != 1:
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
	strictest := model.Allow
	for i, s := range signals {
		d := e.Evaluate(s)
		rows[i] = report.Row{Signal: s, Decision: d}
		decisions[i] = d
		strictest = max(strictest, d.Action)
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
	return int(strictest) // §2: the severity doubles as the exit code
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

const runUsage = "usage: zt-sim run <scenarios-dir>"

// cmdRun loads every *.json pack in dir, checks each case's expectation
// against the real pack, prints per-case lines and the summary table.
// Exit 0 all pass, 1 any failure, 65 unreadable/invalid pack.
func cmdRun(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, runUsage)
		return 64
	}
	dir := args[0]
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(stderr, "run: %v\n", err)
		return 65
	}
	var packs []scenario.Pack
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		p, err := scenario.Load(filepath.Join(dir, e.Name()))
		if err != nil {
			fmt.Fprintf(stderr, "run: %v\n", err)
			return 65
		}
		packs = append(packs, p)
	}
	if len(packs) == 0 {
		fmt.Fprintf(stderr, "run: no scenario packs in %s\n", dir)
		return 65
	}

	eng := engine.New(policies.Pack())
	var results []report.PackResult
	failed := false
	for _, p := range packs {
		res := report.PackResult{Name: p.Name, Strictest: model.Allow}
		for _, c := range p.Cases {
			d := eng.Evaluate(c.Signal)
			res.Cases++
			res.Strictest = max(res.Strictest, d.Action)
			if err := c.Check(d); err != nil {
				failed = true
				res.Fail++
				fmt.Fprintf(stdout, "FAIL %s/%s: %v\n", p.Name, c.Name, err)
				continue
			}
			res.Pass++
			fmt.Fprintf(stdout, "PASS %s/%s\n", p.Name, c.Name)
		}
		results = append(results, res)
	}
	if err := report.PackSummary(stdout, results); err != nil {
		fmt.Fprintf(stderr, "run: %v\n", err)
		return 65
	}
	if failed {
		return 1
	}
	return 0
}

const policiesUsage = "usage: zt-sim policies"

// cmdPolicies lists the v1 pack: ID, priority, pillar, action, note (§5).
func cmdPolicies(args []string, stdout, stderr io.Writer) int {
	if len(args) != 0 {
		fmt.Fprintln(stderr, policiesUsage)
		return 64
	}
	if err := report.Policies(stdout, policies.Pack()); err != nil {
		fmt.Fprintf(stderr, "policies: %v\n", err)
		return 65
	}
	return 0
}
