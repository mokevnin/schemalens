// Package finding defines the structured result of a rule violation.
package finding

import (
	"cmp"
	"slices"
)

// Object names the database object a finding is about. Only the fields that
// apply are set; Schema and Table are always set.
type Object struct {
	Schema     string `json:"schema"`
	Table      string `json:"table"`
	Column     string `json:"column,omitempty"`
	Index      string `json:"index,omitempty"`
	ForeignKey string `json:"foreign_key,omitempty"`
}

// Position is a place in the model's source, supplied by adapters.
type Position struct {
	File string `json:"file"`
	Line int    `json:"line"`
}

// Finding is one disagreement between a model and the database, or one
// violated database rule. Rule is a stable identifier agents can branch on.
type Finding struct {
	Rule    string    `json:"rule"`
	Object  Object    `json:"object"`
	Message string    `json:"message"`
	Fix     string    `json:"fix"`
	Source  *Position `json:"source,omitempty"`
}

// Sort orders findings deterministically: by rule, then by object, then by
// message. It sorts in place.
func Sort(fs []Finding) {
	slices.SortStableFunc(fs, func(a, b Finding) int {
		return cmp.Or(
			cmp.Compare(a.Rule, b.Rule),
			cmp.Compare(a.Object.Schema, b.Object.Schema),
			cmp.Compare(a.Object.Table, b.Object.Table),
			cmp.Compare(a.Object.Column, b.Object.Column),
			cmp.Compare(a.Object.Index, b.Object.Index),
			cmp.Compare(a.Object.ForeignKey, b.Object.ForeignKey),
			cmp.Compare(a.Message, b.Message),
		)
	})
}
