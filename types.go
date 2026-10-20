package schemalens

import (
	"github.com/mokevnin/schemalens/internal/dialect"
	"github.com/mokevnin/schemalens/internal/finding"
)

// Dialect identifies a database engine.
type Dialect string

// Supported dialects.
const (
	SQLite   Dialect = "sqlite"
	Postgres Dialect = "postgres"
	MySQL    Dialect = "mysql"
)

// Object names the database object a finding is about. Only the fields that
// apply are set; Schema and Table are always set, except for views and
// functions, which have no table.
type Object struct {
	Schema     string `json:"schema"`
	Table      string `json:"table,omitempty"`
	Column     string `json:"column,omitempty"`
	Index      string `json:"index,omitempty"`
	ForeignKey string `json:"foreign_key,omitempty"`
	Trigger    string `json:"trigger,omitempty"`
	View       string `json:"view,omitempty"`
	Function   string `json:"function,omitempty"`
}

// Position is a place in the model's source, supplied by adapters.
type Position struct {
	File string `json:"file"`
	Line int    `json:"line"`
}

// Finding is one disagreement between a model and the database, or one
// violated database rule. Rule is a stable identifier to branch on; Fix says
// how to repair the model.
type Finding struct {
	Rule    string    `json:"rule"`
	Object  Object    `json:"object"`
	Message string    `json:"message"`
	Fix     string    `json:"fix"`
	Source  *Position `json:"source,omitempty"`
}

// Exclusion silences the findings of one rule on one object. Object is
// "schema.table", optionally followed by ".column:NAME", ".index:NAME",
// ".foreign_key:NAME" or ".trigger:NAME"; views and functions are written
// "schema.view:NAME" and "schema.function:NAME". Reason is required.
type Exclusion struct {
	Rule   string `yaml:"rule" json:"rule"`
	Object string `yaml:"object" json:"object"`
	Reason string `yaml:"reason" json:"reason"`
}

func findingFromInternal(f finding.Finding) Finding {
	out := Finding{Rule: f.Rule, Object: Object(f.Object), Message: f.Message, Fix: f.Fix}
	if f.Source != nil {
		out.Source = &Position{File: f.Source.File, Line: f.Source.Line}
	}
	return out
}

func (f Finding) internal() finding.Finding {
	out := finding.Finding{Rule: f.Rule, Object: finding.Object(f.Object), Message: f.Message, Fix: f.Fix}
	if f.Source != nil {
		out.Source = &finding.Position{File: f.Source.File, Line: f.Source.Line}
	}
	return out
}

func dialectsFromInternal(ds []dialect.Dialect) []Dialect {
	out := make([]Dialect, len(ds))
	for i, d := range ds {
		out[i] = Dialect(d)
	}
	return out
}
