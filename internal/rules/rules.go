// Package rules holds the rule registry and the rules themselves.
//
// A rule is written once against Atlas's schema model. Dialect differences
// live inside the rule, and every rule declares the dialects it applies to.
// To add a rule: create a file in this package that calls register from init.
package rules

import (
	"cmp"
	"context"
	"database/sql"
	"fmt"
	"slices"

	"ariga.io/atlas/sql/schema"

	"github.com/mokevnin/schemalens/internal/dialect"
	"github.com/mokevnin/schemalens/internal/finding"
	"github.com/mokevnin/schemalens/internal/manifest"
)

// Input is everything a rule may look at.
type Input struct {
	Dialect dialect.Dialect
	Realm   *schema.Realm
	// Manifest is the model, or nil when check runs without one. Rules that
	// compare model with database return nothing when it is nil.
	Manifest *manifest.Manifest
	// DB is a read-only handle for rules that query the database's own catalog
	// (see Rule.CheckDB). It is nil when no database is available.
	DB *sql.DB
	// RuleOptions looks up the configured options of a rule by id; nil means
	// no rule has options. The engine narrows it to Options before calling a
	// rule, so a rule reads only Options.
	RuleOptions func(ruleID string) map[string]any
	// Options are the options configured for the rule being checked, or nil.
	// Rules that need configuration must stay inactive when it is absent.
	Options map[string]any
}

// Explanation is the text behind `schemalens explain <rule>`.
type Explanation struct {
	Why       string // why the rule exists
	Fix       string // how to fix a violation at the model level
	Exception string // when an exclusion is legitimate
}

// Rule is one registered check.
type Rule struct {
	ID       string            // stable, kebab-case, never reused
	Dialects []dialect.Dialect // engines the rule applies to
	Explain  Explanation
	Check    func(Input) []finding.Finding
	// CheckDB is an alternative to Check for rules that query the database
	// catalog through Input.DB; it is skipped when Input.DB is nil.
	CheckDB func(context.Context, Input) ([]finding.Finding, error)
	// DefaultOff rules run only when the configuration enables them.
	DefaultOff bool
}

// AppliesTo reports whether the rule runs on d.
func (r Rule) AppliesTo(d dialect.Dialect) bool {
	return slices.Contains(r.Dialects, d)
}

var registry = map[string]Rule{}

func register(r Rule) {
	if _, dup := registry[r.ID]; dup {
		panic(fmt.Sprintf("rules: duplicate rule id %q", r.ID))
	}
	registry[r.ID] = r
}

// All returns every registered rule, sorted by id.
func All() []Rule {
	out := make([]Rule, 0, len(registry))
	for _, r := range registry {
		out = append(out, r)
	}
	slices.SortFunc(out, func(a, b Rule) int { return cmp.Compare(a.ID, b.ID) })
	return out
}

// Lookup returns the rule with the given id.
func Lookup(id string) (Rule, bool) {
	r, ok := registry[id]
	return r, ok
}

// Skipped returns the rules that do not apply to d, sorted by id. Callers
// report them so that a clean result is never mistaken for "checked".
func Skipped(d dialect.Dialect) []Rule {
	var out []Rule
	for _, r := range All() {
		if !r.AppliesTo(d) {
			out = append(out, r)
		}
	}
	return out
}

// Run executes every rule that applies to in.Dialect and returns the findings
// in deterministic order. Rules that need a database (CheckDB) are skipped when
// in.DB is nil; use RunEnabled to run them and see their errors.
func Run(in Input) []finding.Finding {
	out, _ := RunEnabled(context.Background(), in, nil)
	return out
}

// RunEnabled is Run restricted to the rules for which enabled returns true;
// a nil enabled runs every rule. It fails if a rule cannot query the database.
func RunEnabled(ctx context.Context, in Input, enabled func(ruleID string) bool) ([]finding.Finding, error) {
	out := []finding.Finding{}
	for _, r := range All() {
		if !r.AppliesTo(in.Dialect) || (enabled != nil && !enabled(r.ID)) {
			continue
		}
		ri := in
		if in.RuleOptions != nil {
			ri.Options = in.RuleOptions(r.ID)
		}
		if r.Check != nil {
			out = append(out, r.Check(ri)...)
		}
		if r.CheckDB != nil && in.DB != nil {
			fs, err := r.CheckDB(ctx, ri)
			if err != nil {
				return nil, fmt.Errorf("rule %s: %w", r.ID, err)
			}
			out = append(out, fs...)
		}
	}
	finding.Sort(out)
	return out, nil
}
