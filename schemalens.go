package schemalens

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/mokevnin/schemalens/internal/config"
	"github.com/mokevnin/schemalens/internal/finding"
	"github.com/mokevnin/schemalens/internal/inspect"
	"github.com/mokevnin/schemalens/internal/manifest"
	"github.com/mokevnin/schemalens/internal/rules"
	"github.com/mokevnin/schemalens/internal/sarif"
)

// Manifest is a validated model manifest. Obtain one with ParseManifest.
type Manifest struct {
	m *manifest.Manifest
}

// ParseManifest decodes and validates a manifest document (JSON), exactly as
// `schemalens check --manifest` does: the version is checked first, then the
// document is validated against the schema printed by ManifestSchema.
func ParseManifest(data []byte) (*Manifest, error) {
	m, err := manifest.Parse(data)
	if err != nil {
		return nil, err
	}
	return &Manifest{m: m}, nil
}

// ManifestSchema returns the JSON Schema of the manifest document.
func ManifestSchema() []byte { return manifest.Schema() }

// Options is the input of Check.
type Options struct {
	// DSN names the database: sqlite://<path>, postgres://user:pass@host/db or
	// mysql://user:pass@host/db. Required. SQLite files are opened read-only.
	DSN string
	// Manifest is the model as a manifest document (JSON), validated like
	// ParseManifest. Optional: without a model only database rules run.
	Manifest []byte
	// Model is an already parsed manifest, an alternative to Manifest. Setting
	// both is an error.
	Model *Manifest
	// Config tunes rules and exclusions. Optional: nil means the zero Config.
	// Check never reads schemalens.yml by itself; use LoadConfig.
	Config *Config
}

// Report is the result of Check.
type Report struct {
	// Dialect is the engine of the inspected database.
	Dialect Dialect
	// Findings are the violations that no exclusion silenced, in
	// deterministic order. It is never nil.
	Findings []Finding
	// UnmatchedExclusions are configured exclusions that matched no finding.
	// They fail the report so stale exclusions do not accumulate.
	UnmatchedExclusions []Exclusion
	// Warnings are problems with the input that are not findings, such as a
	// manifest table missing from the database.
	Warnings []string
	// SkippedRules are the IDs of rules that do not apply to Dialect.
	SkippedRules []string
}

// Passed reports whether the check is clean: no findings and no unmatched
// exclusions. The CLI exits 0 exactly when Passed is true and 1 otherwise.
func (r *Report) Passed() bool {
	return len(r.Findings) == 0 && len(r.UnmatchedExclusions) == 0
}

// Check inspects the database, runs the enabled rules and applies the
// configured exclusions. An error means the check could not run (bad DSN,
// invalid manifest or configuration, unreadable database); findings are never
// errors. On error the Report is nil.
func Check(ctx context.Context, opts Options) (*Report, error) {
	var model *manifest.Manifest
	switch {
	case opts.Model != nil && len(opts.Manifest) > 0:
		return nil, errors.New("options: Model and Manifest are mutually exclusive")
	case opts.Model != nil:
		model = opts.Model.m
	case len(opts.Manifest) > 0:
		var err error
		if model, err = manifest.Parse(opts.Manifest); err != nil {
			return nil, fmt.Errorf("manifest: %w", err)
		}
	}
	cfg, err := opts.Config.internal()
	if err != nil {
		return nil, err
	}
	res, err := inspect.Inspect(ctx, opts.DSN)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	rep := &Report{Dialect: Dialect(res.Dialect), Warnings: rules.Unmapped(model, res.Realm)}
	for _, r := range rules.Skipped(res.Dialect) {
		rep.SkippedRules = append(rep.SkippedRules, r.ID)
	}
	found, err := rules.RunEnabled(ctx, rules.Input{
		Dialect: res.Dialect, Realm: res.Realm, Manifest: model, DB: res.DB, RuleOptions: cfg.Options,
	}, cfg.Enabled)
	if err != nil {
		return nil, err
	}
	kept, unmatched := cfg.Apply(found)
	rep.Findings = make([]Finding, len(kept))
	for i, f := range kept {
		rep.Findings[i] = findingFromInternal(f)
	}
	for _, e := range unmatched {
		rep.UnmatchedExclusions = append(rep.UnmatchedExclusions, Exclusion(e))
	}
	return rep, nil
}

// WriteJSON writes the report as the JSON document `schemalens check` prints:
// "findings" and, when present, "unmatched_exclusions".
func (r *Report) WriteJSON(w io.Writer) error {
	findings := r.Findings
	if findings == nil {
		findings = []Finding{}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(struct {
		Findings            []Finding   `json:"findings"`
		UnmatchedExclusions []Exclusion `json:"unmatched_exclusions,omitempty"`
	}{findings, r.UnmatchedExclusions})
}

// WriteSARIF writes the report as the SARIF 2.1.0 log that
// `schemalens check --format sarif` prints.
func (r *Report) WriteSARIF(w io.Writer) error {
	fs := make([]finding.Finding, len(r.Findings))
	for i, f := range r.Findings {
		fs[i] = f.internal()
	}
	um := make([]sarif.Unmatched, len(r.UnmatchedExclusions))
	for i, e := range r.UnmatchedExclusions {
		um[i] = sarif.Unmatched{Rule: e.Rule, Object: e.Object, Reason: e.Reason}
	}
	return sarif.Render(w, fs, um)
}

// BaselineOptions is the input of Baseline.
type BaselineOptions struct {
	// DSN names the database, as in Options.
	DSN string
	// Config is the current configuration YAML, or empty for none. Existing
	// exclusions, reasons and comments are preserved.
	Config []byte
}

// Baseline records the current findings as exclusions with the reason
// "baseline" and returns the updated configuration YAML and how many
// exclusions it added. Findings that Config already excludes are left alone, so
// running it twice adds nothing, and the returned YAML is then Config
// unchanged. Baseline only returns the content; writing it is up to the
// caller. Model rules do not take part, because baselining has no manifest.
func Baseline(ctx context.Context, opts BaselineOptions) ([]byte, int, error) {
	cfg, err := config.Parse(opts.Config)
	if err != nil {
		return nil, 0, err
	}
	res, err := inspect.Inspect(ctx, opts.DSN)
	if err != nil {
		return nil, 0, err
	}
	defer res.Close()
	found, err := rules.RunEnabled(ctx, rules.Input{
		Dialect: res.Dialect, Realm: res.Realm, DB: res.DB, RuleOptions: cfg.Options,
	}, cfg.Enabled)
	if err != nil {
		return nil, 0, err
	}
	return config.AddBaseline(opts.Config, found)
}
