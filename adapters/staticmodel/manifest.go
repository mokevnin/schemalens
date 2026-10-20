package staticmodel

import (
	"go/types"
	"strings"
)

// The manifest document, version 1. Mirrors `schemalens manifest-schema`;
// kept here so adapters do not depend on the core module.

// Manifest is what an adapter prints on stdout.
type Manifest struct {
	Version int     `json:"version"`
	Tables  []Table `json:"tables"`
}

// Table is one table the model maps to.
type Table struct {
	Schema     string     `json:"schema,omitempty"`
	Name       string     `json:"name"`
	Source     *Source    `json:"source,omitempty"`
	Columns    []Column   `json:"columns"`
	PrimaryKey []string   `json:"primary_key,omitempty"`
	Uniques    []Unique   `json:"uniques,omitempty"`
	Relations  []Relation `json:"relations,omitempty"`
}

// Unique is a composite unique constraint.
type Unique struct {
	Columns []string `json:"columns"`
}

// Relation is a foreign key from Columns to the referenced table.
type Relation struct {
	Columns    []string `json:"columns"`
	References TableRef `json:"references"`
}

// TableRef names a table.
type TableRef struct {
	Schema string `json:"schema,omitempty"`
	Table  string `json:"table"`
}

// Column is one column the model expects.
type Column struct {
	Name     string  `json:"name"`
	Required bool    `json:"required,omitempty"`
	Unique   bool    `json:"unique,omitempty"`
	MaxLen   int     `json:"max_length,omitempty"`
	Source   *Source `json:"source,omitempty"`
}

// Source is a place in the project's source, as the manifest spells it.
type Source struct {
	File string `json:"file"`
	Line int    `json:"line"`
}

// ToSource converts a position to the manifest form.
func ToSource(p Position) *Source { return &Source{File: p.File, Line: p.Line} }

// Ref returns the reference to the table.
func (t Table) Ref() TableRef { return TableRef{Schema: t.Schema, Table: t.Name} }

// SplitSchema sets Schema and Name from a possibly schema-qualified name
// ("billing.invoices").
func (t *Table) SplitSchema(name string) {
	if schema, rest, ok := strings.Cut(name, "."); ok {
		t.Schema, t.Name = schema, rest
		return
	}
	t.Name = name
}

// IsString reports whether t (through a pointer) is a Go string type.
func IsString(t types.Type) bool {
	basic, ok := Deref(t).Underlying().(*types.Basic)
	return ok && basic.Kind() == types.String
}
