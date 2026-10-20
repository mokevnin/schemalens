// Package cli implements the schemalens command line. Run is the single entry
// point so tests drive exactly what a user or an agent would.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/mokevnin/schemalens/internal/config"
	"github.com/mokevnin/schemalens/internal/finding"
	"github.com/mokevnin/schemalens/internal/inspect"
	"github.com/mokevnin/schemalens/internal/manifest"
	"github.com/mokevnin/schemalens/internal/rules"
	"github.com/mokevnin/schemalens/internal/sarif"
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

// report is the JSON document printed by `check`.
type report struct {
	Findings []finding.Finding `json:"findings"`
	// UnmatchedExclusions are configured exclusions that matched no finding;
	// they fail the run so stale exclusions do not accumulate.
	UnmatchedExclusions []config.Exclusion `json:"unmatched_exclusions,omitempty"`
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
	cfg, err := config.Load(*configPath)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "schemalens check: %v\n", err)
		return ExitUsage
	}

	if *format != "json" && *format != "sarif" {
		_, _ = fmt.Fprintf(stderr, "schemalens check: unknown --format %q (want json or sarif)\n", *format)
		return ExitUsage
	}

	var model *manifest.Manifest
	var code int
	if *adapterName != "" {
		model, code = runAdapter(*adapterName, adapterArgs, stderr)
	} else {
		model, code = loadManifest(*manifestPath, stderr)
	}
	if code != ExitOK {
		return code
	}

	res, err := inspect.Inspect(context.Background(), *dsn)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "schemalens check: %v\n", err)
		return ExitUsage
	}
	for _, r := range rules.Skipped(res.Dialect) {
		_, _ = fmt.Fprintf(stderr, "schemalens check: rule %s skipped: does not apply to %s\n", r.ID, res.Dialect)
	}
	defer res.Close()
	findings, err := rules.RunEnabled(context.Background(), rules.Input{
		Dialect: res.Dialect, Realm: res.Realm, Manifest: model, DB: res.DB, RuleOptions: cfg.Options,
	}, cfg.Enabled)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "schemalens check: %v\n", err)
		return ExitUsage
	}
	findings, unmatched := cfg.Apply(findings)

	if *format == "sarif" {
		um := make([]sarif.Unmatched, len(unmatched))
		for i, e := range unmatched {
			um[i] = sarif.Unmatched{Rule: e.Rule, Object: e.Object, Reason: e.Reason}
		}
		err = sarif.Render(stdout, findings, um)
	} else {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		err = enc.Encode(report{Findings: findings, UnmatchedExclusions: unmatched})
	}
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "schemalens check: write output: %v\n", err)
		return ExitUsage
	}
	for _, e := range unmatched {
		_, _ = fmt.Fprintf(stderr, "schemalens check: exclusion matches nothing: rule %s, object %s (%s)\n", e.Rule, e.Object, e.Reason)
	}
	if len(findings) > 0 || len(unmatched) > 0 {
		return ExitFindings
	}
	return ExitOK
}
