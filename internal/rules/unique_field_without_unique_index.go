package rules

import (
	"fmt"
	"slices"
	"strings"

	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/schema"

	"github.com/mokevnin/schemalens/internal/dialect"
	"github.com/mokevnin/schemalens/internal/finding"
	"github.com/mokevnin/schemalens/internal/manifest"
)

// UniqueFieldWithoutUniqueIndex is the id of the rule below.
const UniqueFieldWithoutUniqueIndex = "unique-field-without-unique-index"

func init() {
	register(Rule{
		ID:       UniqueFieldWithoutUniqueIndex,
		Dialects: dialect.All,
		Explain: Explanation{
			Why:       "The model promises that a field, or a combination of fields, never repeats, and the application relies on that when it looks a row up by it. Only a unique index or primary key makes the database refuse duplicates; a check in application code loses races between concurrent writers, and a plain index does not prevent anything.",
			Fix:       "Keep the uniqueness in the model (a unique field, or a composite unique index over the fields) and regenerate the migration. Deduplicate existing rows first, or the unique index cannot be created.",
			Exception: "Uniqueness that is deliberately enforced elsewhere, for example by a partial or expression index the model cannot express, or soft-deleted rows that may repeat; write the reason down.",
		},
		Check: checkUniqueFieldWithoutUniqueIndex,
	})
}

// modelUnique is one uniqueness promise of the model.
type modelUnique struct {
	Columns []string
	Source  *finding.Position
}

// modelUniques lists the uniqueness a table promises, without duplicates: a
// column marked unique and an identical entry in Uniques count once.
func modelUniques(t manifest.Table) []modelUnique {
	var out []modelUnique
	seen := map[string]bool{}
	add := func(cols []string, src *finding.Position) {
		key := strings.Join(nameSet(cols), "\x00")
		if len(cols) == 0 || seen[key] {
			return
		}
		seen[key] = true
		out = append(out, modelUnique{Columns: cols, Source: src})
	}
	for _, c := range t.Columns {
		if c.Unique {
			add([]string{c.Name}, sourceOf(t, c))
		}
	}
	for _, u := range t.Uniques {
		add(u.Columns, t.Source)
	}
	return out
}

func checkUniqueFieldWithoutUniqueIndex(in Input) []finding.Finding {
	var out []finding.Finding
	for _, mt := range modelTables(in) {
		for _, u := range modelUniques(mt.Model) {
			if !uniquelyIndexed(mt.DB, u.Columns) {
				out = append(out, uniqueFinding(mt, u))
			}
		}
	}
	return out
}

func uniqueFinding(mt modelTable, u modelUnique) finding.Finding {
	obj := finding.Object{Schema: mt.Schema, Table: mt.DB.Name, Column: strings.Join(u.Columns, ",")}
	if len(u.Columns) == 1 {
		return finding.Finding{
			Rule:    UniqueFieldWithoutUniqueIndex,
			Object:  obj,
			Message: fmt.Sprintf("model field for column %q of table %q is unique, but no unique index or primary key covers the column", u.Columns[0], mt.DB.Name),
			Fix: fmt.Sprintf("Add a unique index on column %q of table %q: keep the field unique in the model, "+
				"then regenerate the migration and apply it (remove duplicate rows first).", u.Columns[0], mt.DB.Name),
			Source: u.Source,
		}
	}
	return finding.Finding{
		Rule:    UniqueFieldWithoutUniqueIndex,
		Object:  obj,
		Message: fmt.Sprintf("model declares columns %s of table %q unique together, but no unique index or primary key covers them", quoteList(u.Columns), mt.DB.Name),
		Fix: fmt.Sprintf("Add a unique index on columns %s of table %q: keep the composite uniqueness in the model, "+
			"then regenerate the migration and apply it (remove duplicate rows first).", quoteList(u.Columns), mt.DB.Name),
		Source: u.Source,
	}
}

// uniquelyIndexed reports whether the primary key or a unique index consists
// only of columns from cols. Such a key already forbids duplicates of the
// whole combination, so a narrower unique index counts as enforcement.
// Partial, expression and prefix indexes do not.
func uniquelyIndexed(t *schema.Table, cols []string) bool {
	if t.PrimaryKey != nil && withinColumns(t.PrimaryKey, cols) {
		return true
	}
	for _, idx := range t.Indexes {
		if idx.Unique && !isPartial(idx) && withinColumns(idx, cols) {
			return true
		}
	}
	return false
}

func withinColumns(idx *schema.Index, cols []string) bool {
	if len(idx.Parts) == 0 {
		return false
	}
	for _, p := range idx.Parts {
		if p.C == nil || !slices.Contains(cols, p.C.Name) || hasPrefixLength(p) {
			return false
		}
	}
	return true
}

// hasPrefixLength reports whether a MySQL index part indexes only a prefix of
// the column, which does not make the column unique.
func hasPrefixLength(p *schema.IndexPart) bool {
	for _, a := range p.Attrs {
		if sp, ok := a.(*mysql.SubPart); ok && sp.Len > 0 {
			return true
		}
	}
	return false
}
