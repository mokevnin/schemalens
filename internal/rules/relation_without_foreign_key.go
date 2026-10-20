package rules

import (
	"fmt"
	"slices"
	"strings"

	"ariga.io/atlas/sql/schema"

	"github.com/mokevnin/schemalens/internal/dialect"
	"github.com/mokevnin/schemalens/internal/finding"
	"github.com/mokevnin/schemalens/internal/manifest"
)

// RelationWithoutForeignKey is the id of the rule below.
const RelationWithoutForeignKey = "relation-without-foreign-key"

func init() {
	register(Rule{
		ID:       RelationWithoutForeignKey,
		Dialects: dialect.All,
		Explain: Explanation{
			Why:       "The model relates two tables, and queries and cascades assume the referenced row exists. Without a foreign key the database accepts references to rows that were never created or have been deleted, and those dangling ids surface later as missing data or failed joins. A many-to-many join table needs a foreign key to each side for the same reason.",
			Fix:       "Declare the relation in the model so the generated schema has a foreign key from the referencing columns (for a join table, one to each joined table), then regenerate the migration. Delete or repair orphaned rows first, or the constraint cannot be added.",
			Exception: "Relations deliberately kept without a constraint, for example across a sharded boundary, to a table in another database, or on a table that is bulk loaded; write the reason down.",
		},
		Check: checkRelationWithoutForeignKey,
	})
}

func checkRelationWithoutForeignKey(in Input) []finding.Finding {
	var out []finding.Finding
	for _, mt := range modelTables(in) {
		for _, rel := range mt.Model.Relations {
			if relationHasForeignKey(mt, rel) {
				continue
			}
			out = append(out, finding.Finding{
				Rule:   RelationWithoutForeignKey,
				Object: finding.Object{Schema: mt.Schema, Table: mt.DB.Name, Column: strings.Join(rel.Columns, ",")},
				Message: fmt.Sprintf("model relates %s of table %q to table %q, but no foreign key does that",
					columnsPhrase(rel.Columns), mt.DB.Name, rel.References.Table),
				Fix: fmt.Sprintf("Add a foreign key from %s of table %q to table %q: declare the relation in the model, "+
					"then regenerate the migration and apply it (repair orphaned rows first).",
					columnsPhrase(rel.Columns), mt.DB.Name, rel.References.Table),
				Source: mt.Model.Source,
			})
		}
		out = append(out, joinTableFindings(mt)...)
	}
	return out
}

func columnsPhrase(cols []string) string {
	if len(cols) == 1 {
		return fmt.Sprintf("column %q", cols[0])
	}
	return "columns " + quoteList(cols)
}

// refersTo reports whether fk references the table named by ref. An empty
// schema in the manifest matches any schema the foreign key points into.
func refersTo(fk *schema.ForeignKey, ref manifest.TableRef) bool {
	if fk.RefTable == nil || fk.RefTable.Name != ref.Table {
		return false
	}
	return ref.Schema == "" || fk.RefTable.Schema == nil || fk.RefTable.Schema.Name == ref.Schema
}

func fkColumnNames(cols []*schema.Column) []string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = c.Name
	}
	return out
}

// relationHasForeignKey reports whether a foreign key links exactly the
// relation's columns (in any order) to the referenced table; when the
// manifest names the referenced columns they must match pairwise.
func relationHasForeignKey(mt modelTable, rel manifest.Relation) bool {
	for _, fk := range mt.DB.ForeignKeys {
		if !refersTo(fk, rel.References) || !slices.Equal(nameSet(fkColumnNames(fk.Columns)), nameSet(rel.Columns)) {
			continue
		}
		if len(rel.RefColumns) == 0 {
			return true
		}
		if len(rel.RefColumns) != len(rel.Columns) || len(fk.RefColumns) != len(fk.Columns) {
			continue
		}
		pairs := map[string]string{}
		for i, c := range fk.Columns {
			pairs[c.Name] = fk.RefColumns[i].Name
		}
		match := true
		for i, c := range rel.Columns {
			if pairs[c] != rel.RefColumns[i] {
				match = false
			}
		}
		if match {
			return true
		}
	}
	return false
}

// joinTableFindings checks that a join table has a distinct foreign key to
// each table it joins; a table joined to itself needs two.
func joinTableFindings(mt modelTable) []finding.Finding {
	var out []finding.Finding
	used := map[*schema.ForeignKey]bool{}
	for _, ref := range mt.Model.JoinTableFor {
		var hit *schema.ForeignKey
		for _, fk := range mt.DB.ForeignKeys {
			if !used[fk] && refersTo(fk, ref) {
				hit = fk
				break
			}
		}
		if hit != nil {
			used[hit] = true
			continue
		}
		out = append(out, finding.Finding{
			Rule:   RelationWithoutForeignKey,
			Object: finding.Object{Schema: mt.Schema, Table: mt.DB.Name},
			Message: fmt.Sprintf("table %q is the join table for %s in the model, but has no foreign key to table %q",
				mt.DB.Name, joinedPhrase(mt.Model.JoinTableFor), ref.Table),
			Fix: fmt.Sprintf("Add a foreign key from table %q to table %q: declare the many-to-many relation in the model "+
				"so its join table references both sides, then regenerate the migration and apply it.", mt.DB.Name, ref.Table),
			Source: mt.Model.Source,
		})
	}
	return out
}

func joinedPhrase(refs []manifest.TableRef) string {
	names := make([]string, len(refs))
	for i, r := range refs {
		names[i] = r.Table
	}
	return quoteList(names)
}
