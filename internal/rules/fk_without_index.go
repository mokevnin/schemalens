package rules

import (
	"fmt"
	"slices"
	"strings"

	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlite"

	"github.com/mokevnin/schemalens/internal/dialect"
	"github.com/mokevnin/schemalens/internal/finding"
)

// FKWithoutIndex is the id of the rule below.
const FKWithoutIndex = "fk-without-index"

func init() {
	register(Rule{
		ID: FKWithoutIndex,
		// MySQL (InnoDB) creates an index for every foreign key itself when none
		// exists, so the rule has nothing to say there and does not run.
		Dialects: []dialect.Dialect{dialect.Postgres, dialect.SQLite},
		Explain: Explanation{
			Why:       "Deleting or updating a referenced row scans the referencing table for matching rows, and joins on the foreign key scan it too. Without an index on the referencing columns both degrade to full table scans and take long locks as the table grows.",
			Fix:       "Add an index to the model whose leading columns are the foreign key columns, then regenerate the schema.",
			Exception: "Tiny, static referencing tables, or a foreign key whose referenced rows are never updated or deleted and never joined through.",
		},
		Check: checkFKWithoutIndex,
	})
}

func checkFKWithoutIndex(in Input) []finding.Finding {
	var out []finding.Finding
	for _, s := range in.Realm.Schemas {
		for _, t := range s.Tables {
			for _, fk := range t.ForeignKeys {
				if len(fk.Columns) == 0 || fkIndexed(t, fk) {
					continue
				}
				names := make([]string, len(fk.Columns))
				for i, c := range fk.Columns {
					names[i] = fmt.Sprintf("%q", c.Name)
				}
				cols := strings.Join(names, ", ")
				plural := "column " + cols + " is"
				lead := "whose leading column is " + cols
				if len(names) > 1 {
					plural = "columns " + cols + " are"
					lead = "whose leading columns are " + cols
				}
				out = append(out, finding.Finding{
					Rule:    FKWithoutIndex,
					Object:  finding.Object{Schema: s.Name, Table: t.Name, ForeignKey: fk.Symbol},
					Message: fmt.Sprintf("foreign key %q on table %q has no supporting index: %s not indexed", fk.Symbol, t.Name, plural),
					Fix: fmt.Sprintf("Add an index on table %q %s (for example an index tag on the model field), then regenerate the schema.",
						t.Name, lead),
				})
			}
		}
	}
	return out
}

// fkIndexed reports whether the foreign key columns are the leading columns
// of the primary key or of a usable index. Column order inside the prefix does
// not matter: a B-tree serves equality lookups on any permutation of it.
func fkIndexed(t *schema.Table, fk *schema.ForeignKey) bool {
	if t.PrimaryKey != nil && coversPrefix(t.PrimaryKey, fk.Columns) {
		return true
	}
	for _, idx := range t.Indexes {
		if isPartial(idx) {
			continue // serves only the rows matching its predicate
		}
		if coversPrefix(idx, fk.Columns) {
			return true
		}
	}
	return false
}

func coversPrefix(idx *schema.Index, cols []*schema.Column) bool {
	if len(idx.Parts) < len(cols) {
		return false
	}
	names := make([]string, 0, len(cols))
	for _, p := range idx.Parts[:len(cols)] {
		if p.C == nil {
			return false // expression part
		}
		names = append(names, p.C.Name)
	}
	for _, c := range cols {
		if !slices.Contains(names, c.Name) {
			return false
		}
	}
	return true
}

func isPartial(idx *schema.Index) bool {
	for _, a := range idx.Attrs {
		switch p := a.(type) {
		case *postgres.IndexPredicate:
			return p.P != ""
		case *sqlite.IndexPredicate:
			return p.P != ""
		}
	}
	return false
}
