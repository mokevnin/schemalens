package rules

import (
	"fmt"

	"ariga.io/atlas/sql/schema"

	"github.com/mokevnin/schemalens/internal/finding"
	"github.com/mokevnin/schemalens/internal/manifest"
)

// modelTable pairs a manifest table with the database table it maps to.
type modelTable struct {
	Schema string // resolved schema name
	Model  manifest.Table
	DB     *schema.Table
}

// modelTables returns the manifest tables that exist in the database, in
// manifest order. A manifest table with an empty schema maps to the
// database's only schema; with several schemas an empty name matches none.
// Tables absent from the database are skipped: this is not the rule that
// reports them. Rules that compare model with database start here, so schema
// matching is decided in one place.
func modelTables(in Input) []modelTable {
	if in.Manifest == nil || in.Realm == nil {
		return nil
	}
	var out []modelTable
	for _, mt := range in.Manifest.Tables {
		s := findSchema(in.Realm, mt.Schema)
		if s == nil {
			continue
		}
		if t, ok := s.Table(mt.Name); ok {
			out = append(out, modelTable{Schema: s.Name, Model: mt, DB: t})
		}
	}
	return out
}

func findSchema(r *schema.Realm, name string) *schema.Schema {
	if name == "" {
		if len(r.Schemas) == 1 {
			return r.Schemas[0]
		}
		return nil
	}
	for _, s := range r.Schemas {
		if s.Name == name {
			return s
		}
	}
	return nil
}

// sourceOf returns the column's source position, falling back to the
// table's, or nil when the manifest has neither.
func sourceOf(t manifest.Table, c manifest.Column) *finding.Position {
	if c.Source != nil {
		return c.Source
	}
	return t.Source
}

// Unmapped returns a human-readable description of each manifest table the
// model rules cannot compare because it has no database table: the table does
// not exist, or its schema is omitted while the database has several. Model
// rules skip these tables, so callers warn about them instead of letting the
// comparison pass vacuously.
func Unmapped(m *manifest.Manifest, r *schema.Realm) []string {
	if m == nil || r == nil {
		return nil
	}
	var out []string
	for _, mt := range m.Tables {
		s := findSchema(r, mt.Schema)
		switch {
		case s == nil && mt.Schema == "":
			out = append(out, fmt.Sprintf("manifest table %q has no schema and the database has %d schemas; set the table's schema in the manifest, so it is not compared", mt.Name, len(r.Schemas)))
		case s == nil:
			out = append(out, fmt.Sprintf("manifest table %q.%q: the database has no schema %q; the table is not compared", mt.Schema, mt.Name, mt.Schema))
		default:
			if _, ok := s.Table(mt.Name); !ok {
				out = append(out, fmt.Sprintf("manifest table %q does not exist in schema %q of the database; the table is not compared", mt.Name, s.Name))
			}
		}
	}
	return out
}
