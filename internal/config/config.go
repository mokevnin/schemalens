// Package config reads schemalens.yml: which rules run, per-rule options and
// the exclusions that silence individual findings.
//
//	rules:
//	  <rule-id>:
//	    enabled: false        # default true
//	    options: {...}        # rule-specific, opaque to this package
//	exclusions:
//	  - rule: <rule-id>
//	    object: main.audit_log            # schema.table
//	    reason: append-only raw event log # required
//
// An exclusion object is "schema.table" optionally followed by
// ".column:NAME", ".index:NAME" or ".foreign_key:NAME" to name a sub-object.
// Matching is exact; there are no wildcards. Schema and table names must not
// contain dots.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/mokevnin/schemalens/internal/finding"
	"github.com/mokevnin/schemalens/internal/rules"
)

// DefaultFile is read from the working directory when --config is not given.
const DefaultFile = "schemalens.yml"

// RuleConfig is the configuration of one rule.
type RuleConfig struct {
	// Enabled is a pointer so an omitted key means "enabled".
	Enabled *bool `yaml:"enabled,omitempty"`
	// Options are rule-specific settings, interpreted by the rule itself.
	Options map[string]any `yaml:"options,omitempty"`
}

// Exclusion silences the findings of one rule on one object.
type Exclusion struct {
	Rule   string `yaml:"rule" json:"rule"`
	Object string `yaml:"object" json:"object"`
	Reason string `yaml:"reason" json:"reason"`
}

// Config is the parsed and validated configuration file.
type Config struct {
	Rules      map[string]RuleConfig `yaml:"rules,omitempty"`
	Exclusions []Exclusion           `yaml:"exclusions,omitempty"`
}

// Load reads and validates the file at path. An empty path means the default
// file in the working directory, which is optional: if it does not exist the
// zero Config (everything enabled, no exclusions) is returned. An explicitly
// named file must exist.
func Load(path string) (*Config, error) {
	explicit := path != ""
	if !explicit {
		path = DefaultFile
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if !explicit && errors.Is(err, fs.ErrNotExist) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("config: %w", err)
	}
	c, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("config %s: %w", path, err)
	}
	return c, nil
}

// Parse decodes and validates configuration data.
func Parse(data []byte) (*Config, error) {
	var c Config
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&c); err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if err := c.validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Config) validate() error {
	for id := range c.Rules {
		if _, ok := rules.Lookup(id); !ok {
			return fmt.Errorf("rules: unknown rule %q", id)
		}
	}
	for i, e := range c.Exclusions {
		where := fmt.Sprintf("exclusions[%d]", i)
		switch {
		case e.Rule == "":
			return fmt.Errorf("%s: rule is required", where)
		case e.Object == "":
			return fmt.Errorf("%s: object is required", where)
		case strings.TrimSpace(e.Reason) == "":
			return fmt.Errorf("%s (%s on %s): reason is required and must not be empty", where, e.Rule, e.Object)
		}
		if _, ok := rules.Lookup(e.Rule); !ok {
			return fmt.Errorf("%s: unknown rule %q", where, e.Rule)
		}
		if _, err := ParseObject(e.Object); err != nil {
			return fmt.Errorf("%s: %w", where, err)
		}
	}
	return nil
}

// Enabled reports whether the rule runs. Rules are enabled unless disabled.
func (c *Config) Enabled(ruleID string) bool {
	rc, ok := c.Rules[ruleID]
	return !ok || rc.Enabled == nil || *rc.Enabled
}

// Options returns the rule-specific options of a rule, or nil.
func (c *Config) Options(ruleID string) map[string]any {
	return c.Rules[ruleID].Options
}

// ParseObject turns the exclusion object syntax into a finding.Object.
func ParseObject(s string) (finding.Object, error) {
	parts := strings.SplitN(s, ".", 3)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return finding.Object{}, fmt.Errorf("object %q: want schema.table, optionally .column:NAME, .index:NAME or .foreign_key:NAME", s)
	}
	o := finding.Object{Schema: parts[0], Table: parts[1]}
	if len(parts) == 3 {
		kind, name, ok := strings.Cut(parts[2], ":")
		if !ok || name == "" {
			return finding.Object{}, fmt.Errorf("object %q: sub-object must be column:NAME, index:NAME or foreign_key:NAME", s)
		}
		switch kind {
		case "column":
			o.Column = name
		case "index":
			o.Index = name
		case "foreign_key":
			o.ForeignKey = name
		default:
			return finding.Object{}, fmt.Errorf("object %q: unknown sub-object kind %q", s, kind)
		}
	}
	return o, nil
}

// Apply removes the findings matched by an exclusion. It returns the kept
// findings and the exclusions that matched nothing. Exclusions of disabled
// rules are not reported as unmatched, because their rule never ran.
func (c *Config) Apply(findings []finding.Finding) (kept []finding.Finding, unmatched []Exclusion) {
	kept = []finding.Finding{}
	used := make([]bool, len(c.Exclusions))
	objects := make([]finding.Object, len(c.Exclusions))
	for i, e := range c.Exclusions {
		objects[i], _ = ParseObject(e.Object) // validated by Parse
	}
	for _, f := range findings {
		excluded := false
		for i, e := range c.Exclusions {
			if e.Rule == f.Rule && objects[i] == f.Object {
				used[i] = true
				excluded = true
			}
		}
		if !excluded {
			kept = append(kept, f)
		}
	}
	for i, e := range c.Exclusions {
		if !used[i] && c.Enabled(e.Rule) {
			unmatched = append(unmatched, e)
		}
	}
	return kept, unmatched
}
