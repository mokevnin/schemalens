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
			Why:       "The model relates two tables, and queries and cascades assume the referenced row exists. Without a foreign key the database accepts references to rows that were never created or have been deleted, and those dangling ids surface later as missing data or failed joins. A many-to-many join table needs a foreign key to each side for the same reason. A composite key that also carries a NOT NULL column, such as a tenant id leading (tenant_id, author_id) -> users (tenant_id, id), enforces the relation as well and counts; with that column nullable, rows holding NULL in it skip the check.",
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
			if f, ok := relationFinding(mt, rel); ok {
				out = append(out, f)
			}
		}
		out = append(out, joinTableFindings(mt)...)
	}
	return out
}

// relationFinding reports a relation no foreign key enforces. When a key does link the relation's
// columns but also holds a nullable column, the finding names that column: the key exists, and
// making the column NOT NULL is what lets it check every row.
func relationFinding(mt modelTable, rel manifest.Relation) (finding.Finding, bool) {
	var nullable []string
	for _, fk := range mt.DB.ForeignKeys {
		cover, ok := coverRelation(fk, rel)
		if !ok {
			continue
		}
		if len(cover.nullable) == 0 {
			return finding.Finding{}, false
		}
		if nullable == nil {
			nullable = cover.nullable
		}
	}
	f := finding.Finding{
		Rule:   RelationWithoutForeignKey,
		Object: finding.Object{Schema: mt.Schema, Table: mt.DB.Name, Column: strings.Join(rel.Columns, ",")},
		Message: fmt.Sprintf("model relates %s of table %q to table %q, but no foreign key does that",
			columnsPhrase(rel.Columns), mt.DB.Name, rel.References.Table),
		Fix: fmt.Sprintf("Add a foreign key from %s of table %q to table %q: declare the relation in the model, "+
			"then regenerate the migration and apply it (repair orphaned rows first).",
			columnsPhrase(rel.Columns), mt.DB.Name, rel.References.Table),
		Source: mt.Model.Source,
	}
	if nullable != nil {
		f.Message = fmt.Sprintf("model relates %s of table %q to table %q, but the foreign key that links them "+
			"also holds nullable %s, and a row with it NULL skips the check",
			columnsPhrase(rel.Columns), mt.DB.Name, rel.References.Table, columnsPhrase(nullable))
		f.Fix = fmt.Sprintf("Make %s of table %q NOT NULL so the foreign key checks every row: mark the field "+
			"required in the model, then regenerate the migration and apply it (fill the NULLs first).",
			columnsPhrase(nullable), mt.DB.Name)
	}
	return f, true
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

// coverage is how a foreign key covers a relation: nullable lists the key's columns outside the
// relation that may hold NULL.
type coverage struct {
	nullable []string
}

// coverRelation reports whether fk links the relation's columns to the referenced table and points
// each one at the column the relation references. A key of exactly the relation's columns may be
// wider than that: a tenant column leading it, as in (tenant_id, author_id) -> users (tenant_id, id),
// still enforces the relation, but only while that extra column is NOT NULL (MATCH SIMPLE checks
// nothing when any key column is NULL), so the extra nullable columns come back in the coverage.
//
// The relation's referenced columns are the ones the manifest names. When it names none, a key of
// exactly the relation's columns may point at any key of the table; a wider key must point them at
// the referenced table's primary key, in order, since nothing else tells which referenced column
// the relation's column stands for: in (tenant_id, author_id) -> users (id, tenant_id) author_id
// points at a primary key column too, and the key enforces nothing about authors.
func coverRelation(fk *schema.ForeignKey, rel manifest.Relation) (coverage, bool) {
	if !refersTo(fk, rel.References) || len(fk.RefColumns) != len(fk.Columns) {
		return coverage{}, false
	}
	refs := make(map[string]string, len(fk.Columns))
	var extra []*schema.Column
	for i, c := range fk.Columns {
		refs[c.Name] = fk.RefColumns[i].Name
		if !slices.Contains(rel.Columns, c.Name) {
			extra = append(extra, c)
		}
	}
	if len(fk.Columns)-len(extra) != len(rel.Columns) {
		return coverage{}, false
	}
	want := rel.RefColumns
	if len(want) == 0 && len(extra) > 0 {
		want = primaryKeyColumns(fk.RefTable)
		if len(want) != len(rel.Columns) {
			return coverage{}, false
		}
	}
	for i, c := range rel.Columns {
		if len(want) > 0 && refs[c] != want[i] {
			return coverage{}, false
		}
	}
	var cov coverage
	for _, c := range extra {
		if c.Type == nil || c.Type.Null {
			cov.nullable = append(cov.nullable, c.Name)
		}
	}
	return cov, true
}

// primaryKeyColumns lists the names of t's primary key columns, in key order.
func primaryKeyColumns(t *schema.Table) []string {
	if t == nil || t.PrimaryKey == nil {
		return nil
	}
	out := make([]string, 0, len(t.PrimaryKey.Parts))
	for _, p := range t.PrimaryKey.Parts {
		if p.C != nil {
			out = append(out, p.C.Name)
		}
	}
	return out
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
