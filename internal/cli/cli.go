// Package cli implements the schemalens command line. Run is the single entry
// point so tests drive exactly what a user or an agent would.
package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/mokevnin/schemalens"
)

// Exit codes. There is no warning level: any finding is a failure.
const (
	ExitOK       = 0 // no findings
	ExitFindings = 1 // at least one finding
	ExitUsage    = 2 // bad flag, bad configuration or unreadable database
)

// Run executes the CLI with args (without the program name) and returns the
// process exit code. Results go to stdout, diagnostics to stderr.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "usage: schemalens <command> [flags]\ncommands: check, rules, explain, baseline, manifest-schema")
		return ExitUsage
	}
	switch args[0] {
	case "check":
		return runCheck(args[1:], stdout, stderr)
	case "baseline":
		return runBaseline(args[1:], stdout, stderr)
	case "rules":
		return runRules(args[1:], stdout, stderr)
	case "explain":
		return runExplain(args[1:], stdout, stderr)
	case "manifest-schema":
		return runManifestSchema(stdout, stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "schemalens: unknown command %q\ncommands: check, rules, explain, baseline, manifest-schema\n", args[0])
		return ExitUsage
	}
}

func runCheck(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dsn := fs.String("dsn", "", "database to check, for example sqlite://path/to/db.sqlite")
	format := fs.String("format", "json", "output format: json or sarif")
	configPath := fs.String("config", "", "configuration file (default: schemalens.yml in the working directory, if present)")
	manifestPath := fs.String("manifest", "", "model manifest (JSON) to compare with the database")
	adapterName := fs.String("adapter", "", "run schemalens-adapter-<name> from PATH and use the manifest it prints")
	var adapterArgs stringList
	fs.Var(&adapterArgs, "adapter-arg", "argument passed to the adapter (repeatable)")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitOK
		}
		return ExitUsage
	}
	if *adapterName != "" && *manifestPath != "" {
		_, _ = fmt.Fprintln(stderr, "schemalens check: --adapter and --manifest are mutually exclusive")
		return ExitUsage
	}
	if len(adapterArgs) > 0 && *adapterName == "" {
		_, _ = fmt.Fprintln(stderr, "schemalens check: --adapter-arg needs --adapter")
		return ExitUsage
	}
	if *dsn == "" {
		_, _ = fmt.Fprintln(stderr, "schemalens check: --dsn is required")
		return ExitUsage
	}
	cfg, err := schemalens.LoadConfig(*configPath)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "schemalens check: %v\n", err)
		return ExitUsage
	}

	if *format != "json" && *format != "sarif" {
		_, _ = fmt.Fprintf(stderr, "schemalens check: unknown --format %q (want json or sarif)\n", *format)
		return ExitUsage
	}

	var model *schemalens.Manifest
	var code int
	if *adapterName != "" {
		model, code = runAdapter(*adapterName, adapterArgs, stderr)
	} else {
		model, code = loadManifest(*manifestPath, stderr)
	}
	if code != ExitOK {
		return code
	}

	rep, err := schemalens.Check(context.Background(), schemalens.Options{DSN: *dsn, Model: model, Config: cfg})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "schemalens check: %v\n", err)
		return ExitUsage
	}
	for _, id := range rep.SkippedRules {
		_, _ = fmt.Fprintf(stderr, "schemalens check: rule %s skipped: does not apply to %s\n", id, rep.Dialect)
	}
	for _, w := range rep.Warnings {
		_, _ = fmt.Fprintf(stderr, "schemalens check: warning: %s\n", w)
	}

	if *format == "sarif" {
		err = rep.WriteSARIF(stdout)
	} else {
		err = rep.WriteJSON(stdout)
	}
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "schemalens check: write output: %v\n", err)
		return ExitUsage
	}
	for _, e := range rep.UnmatchedExclusions {
		_, _ = fmt.Fprintf(stderr, "schemalens check: exclusion matches nothing: rule %s, object %s (%s)\n", e.Rule, e.Object, e.Reason)
	}
	if !rep.Passed() {
		return ExitFindings
	}
	return ExitOK
}
