package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/mokevnin/schemalens/internal/rules"
)

type ruleListing struct {
	ID       string   `json:"id"`
	Dialects []string `json:"dialects"`
}

func runRules(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("rules", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "output format: text or json")
	if code, done := parseFlags(fs, args); done {
		return code
	}
	if *format != "text" && *format != "json" {
		_, _ = fmt.Fprintf(stderr, "schemalens rules: unknown --format %q (want text or json)\n", *format)
		return ExitUsage
	}

	listing := []ruleListing{}
	for _, r := range rules.All() {
		ds := make([]string, len(r.Dialects))
		for i, d := range r.Dialects {
			ds[i] = string(d)
		}
		slices.Sort(ds)
		listing = append(listing, ruleListing{ID: r.ID, Dialects: ds})
	}

	if *format == "json" {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(struct {
			Rules []ruleListing `json:"rules"`
		}{listing}); err != nil {
			_, _ = fmt.Fprintf(stderr, "schemalens rules: write output: %v\n", err)
			return ExitUsage
		}
		return ExitOK
	}
	width := 0
	for _, l := range listing {
		width = max(width, len(l.ID))
	}
	for _, l := range listing {
		_, _ = fmt.Fprintf(stdout, "%-*s  %s\n", width, l.ID, strings.Join(l.Dialects, ", "))
	}
	return ExitOK
}

func runExplain(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("explain", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if code, done := parseFlags(fs, args); done {
		return code
	}
	if fs.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, "usage: schemalens explain <rule>")
		return ExitUsage
	}
	id := fs.Arg(0)
	r, ok := rules.Lookup(id)
	if !ok {
		_, _ = fmt.Fprintf(stderr, "schemalens explain: unknown rule %q (run `schemalens rules` to list them)\n", id)
		return ExitUsage
	}
	_, _ = fmt.Fprintf(stdout, "%s\n\nWhy:\n  %s\n\nFix:\n  %s\n\nException:\n  %s\n",
		r.ID, r.Explain.Why, r.Explain.Fix, r.Explain.Exception)
	return ExitOK
}

// parseFlags parses fs; done is true when the command must return code now.
func parseFlags(fs *flag.FlagSet, args []string) (code int, done bool) {
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitOK, true
		}
		return ExitUsage, true
	}
	return 0, false
}
