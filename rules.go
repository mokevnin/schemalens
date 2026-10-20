package schemalens

import (
	"slices"

	"github.com/mokevnin/schemalens/internal/rules"
)

// RuleInfo describes one rule.
type RuleInfo struct {
	// ID is the stable, kebab-case identifier used in findings, Config and
	// exclusions.
	ID string
	// Dialects lists the engines the rule applies to, sorted.
	Dialects []Dialect
	// DefaultOn is false for rules that run only when a Config enables them.
	DefaultOn bool
}

// Explanation is the text behind `schemalens explain <rule>`.
type Explanation struct {
	RuleInfo
	Why       string // why the rule exists
	Fix       string // how to fix a violation at the model level
	Exception string // when an exclusion is legitimate
}

func infoOf(r rules.Rule) RuleInfo {
	ds := dialectsFromInternal(r.Dialects)
	slices.Sort(ds)
	return RuleInfo{ID: r.ID, Dialects: ds, DefaultOn: !r.DefaultOff}
}

// Rules lists every rule, sorted by ID.
func Rules() []RuleInfo {
	all := rules.All()
	out := make([]RuleInfo, len(all))
	for i, r := range all {
		out[i] = infoOf(r)
	}
	return out
}

// Explain returns the explanation of the rule with the given ID.
func Explain(id string) (Explanation, bool) {
	r, ok := rules.Lookup(id)
	if !ok {
		return Explanation{}, false
	}
	return Explanation{RuleInfo: infoOf(r), Why: r.Explain.Why, Fix: r.Explain.Fix, Exception: r.Explain.Exception}, true
}
