// Package sarif renders findings as a SARIF 2.1.0 log so code-scanning
// systems can show them as annotations.
package sarif

import (
	"encoding/json"
	"io"
	"slices"
	"strings"

	"github.com/mokevnin/schemalens/internal/finding"
)

const (
	schemaURI = "https://json.schemastore.org/sarif-2.1.0.json"
	version   = "2.1.0"
	infoURI   = "https://github.com/mokevnin/schemalens"
)

type log struct {
	Schema  string `json:"$schema"`
	Version string `json:"version"`
	Runs    []run  `json:"runs"`
}

type run struct {
	Tool        tool         `json:"tool"`
	Invocations []invocation `json:"invocations,omitempty"`
	Results     []result     `json:"results"`
}

// invocation carries run-level problems that are not findings: configured
// exclusions that matched nothing.
type invocation struct {
	ExecutionSuccessful bool           `json:"executionSuccessful"`
	Notifications       []notification `json:"toolConfigurationNotifications"`
}

type notification struct {
	Level      string         `json:"level"`
	Message    text           `json:"message"`
	Descriptor descriptor     `json:"descriptor"`
	Props      map[string]any `json:"properties"`
}

// Unmatched is a configured exclusion that matched no finding.
type Unmatched struct {
	Rule, Object, Reason string
}

// unmatchedExclusionID is the notification descriptor of an unmatched exclusion.
const unmatchedExclusionID = "unmatched-exclusion"

type tool struct {
	Driver driver `json:"driver"`
}

type driver struct {
	Name           string       `json:"name"`
	InformationURI string       `json:"informationUri"`
	Rules          []descriptor `json:"rules"`
}

type descriptor struct {
	ID string `json:"id"`
}

type result struct {
	RuleID    string         `json:"ruleId"`
	RuleIndex int            `json:"ruleIndex"`
	Level     string         `json:"level"`
	Message   text           `json:"message"`
	Locations []location     `json:"locations"`
	Props     map[string]any `json:"properties,omitempty"`
}

type text struct {
	Text string `json:"text"`
}

type location struct {
	Physical *physical `json:"physicalLocation,omitempty"`
	Logical  []logical `json:"logicalLocations,omitempty"`
}

type physical struct {
	Artifact artifact `json:"artifactLocation"`
	Region   region   `json:"region"`
}

type artifact struct {
	URI string `json:"uri"`
}

type region struct {
	StartLine int `json:"startLine"`
}

type logical struct {
	FullyQualifiedName string `json:"fullyQualifiedName"`
	Kind               string `json:"kind"`
}

// Render writes findings as a SARIF log. Every finding is an error-level
// result (schemalens has no warning level). Findings keep their given order;
// rule descriptors are sorted by id, so output is deterministic. Unmatched
// exclusions, which make check fail, are reported in invocations[0] as
// configuration notifications with executionSuccessful false.
func Render(w io.Writer, findings []finding.Finding, unmatched []Unmatched) error {
	var ids []string
	for _, f := range findings {
		ids = append(ids, f.Rule)
	}
	slices.Sort(ids)
	ids = slices.Compact(ids)
	rules := make([]descriptor, len(ids))
	for i, id := range ids {
		rules[i] = descriptor{ID: id}
	}

	results := make([]result, 0, len(findings))
	for _, f := range findings {
		results = append(results, result{
			RuleID:    f.Rule,
			RuleIndex: slices.Index(ids, f.Rule),
			Level:     "error",
			Message:   text{Text: f.Message},
			Locations: []location{locate(f)},
			Props:     map[string]any{"fix": f.Fix},
		})
	}

	r := run{
		Tool:    tool{Driver: driver{Name: "schemalens", InformationURI: infoURI, Rules: rules}},
		Results: results,
	}
	if len(unmatched) > 0 {
		notes := make([]notification, 0, len(unmatched))
		for _, u := range unmatched {
			notes = append(notes, notification{
				Level: "error",
				Message: text{Text: "exclusion matches nothing: rule " + u.Rule + ", object " + u.Object +
					" (" + u.Reason + "); remove it or fix its object"},
				Descriptor: descriptor{ID: unmatchedExclusionID},
				Props:      map[string]any{"rule": u.Rule, "object": u.Object, "reason": u.Reason},
			})
		}
		r.Invocations = []invocation{{ExecutionSuccessful: false, Notifications: notes}}
	}
	doc := log{Schema: schemaURI, Version: version, Runs: []run{r}}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(doc)
}

// locate gives the model source position when known, and always names the
// database object as a logical location.
func locate(f finding.Finding) location {
	var loc location
	if f.Source != nil {
		loc.Physical = &physical{
			Artifact: artifact{URI: f.Source.File},
			Region:   region{StartLine: max(f.Source.Line, 1)},
		}
	}
	name, kind := []string{f.Object.Schema}, "table"
	if f.Object.Table != "" {
		name = append(name, f.Object.Table)
	}
	for _, part := range []struct{ kind, name string }{
		{"column", f.Object.Column}, {"index", f.Object.Index}, {"foreign key", f.Object.ForeignKey},
		{"trigger", f.Object.Trigger}, {"view", f.Object.View}, {"function", f.Object.Function},
	} {
		if part.name != "" {
			name, kind = append(name, part.name), part.kind
		}
	}
	loc.Logical = []logical{{FullyQualifiedName: strings.Join(name, "."), Kind: kind}}
	return loc
}
