package rules

import (
	"fmt"
	"slices"

	"ariga.io/atlas/sql/schema"

	"github.com/mokevnin/schemalens/internal/dialect"
	"github.com/mokevnin/schemalens/internal/finding"
)

// RequiredColumn is the id of the rule below.
const RequiredColumn = "required-column"

func init() {
	register(Rule{
		ID: RequiredColumn,
		// Only SQLite exists today; PostgreSQL and MySQL are added with the dialect ticket.
		Dialects: []dialect.Dialect{dialect.SQLite},
		Explain: Explanation{
			Why: "Some columns must be present on every table of a kind, for example a column that scopes each row to its owner. " +
				"The rule is configured with options: column (required, the column name), tables (optional, the tables it applies to; default all) " +
				"and exclude_tables (optional, tables that are skipped). Without the column option the rule does nothing. " +
				"It reports, separately, a table without the column, a column that is nullable, a column without a foreign key, " +
				"a column without an index that starts with it, and each unique index that does not start with it.",
			Fix:       "Add the column to the model that maps to the table, make it required, declare its relation to the referenced table, and index it first; then regenerate the schema.",
			Exception: "Tables that are shared across all owners, such as reference data or the owner table itself, when the reason is written down.",
		},
		Check: checkRequiredColumn,
	})
}

type requiredColumnOptions struct {
	column  string
	tables  []string
	exclude []string
}

func parseRequiredColumnOptions(o map[string]any) (requiredColumnOptions, bool) {
	col, _ := o["column"].(string)
	return requiredColumnOptions{
		column:  col,
		tables:  stringList(o["tables"]),
		exclude: stringList(o["exclude_tables"]),
	}, col != ""
}

func stringList(v any) []string {
	items, _ := v.([]any)
	var out []string
	for _, it := range items {
		if s, ok := it.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func (o requiredColumnOptions) applies(table string) bool {
	if slices.Contains(o.exclude, table) {
		return false
	}
	return len(o.tables) == 0 || slices.Contains(o.tables, table)
}

func hasForeignKey(t *schema.Table, col *schema.Column) bool {
	for _, fk := range t.ForeignKeys {
		if slices.ContainsFunc(fk.Columns, func(c *schema.Column) bool { return c.Name == col.Name }) {
			return true
		}
	}
	return false
}

func leadsWith(idx *schema.Index, column string) bool {
	return idx != nil && len(idx.Parts) > 0 && idx.Parts[0].C != nil && idx.Parts[0].C.Name == column
}

// hasLeadingIndex reports whether an index or the primary key starts with column.
func hasLeadingIndex(t *schema.Table, column string) bool {
	return leadsWith(t.PrimaryKey, column) || slices.ContainsFunc(t.Indexes, func(i *schema.Index) bool {
		return leadsWith(i, column)
	})
}

func checkRequiredColumn(in Input) []finding.Finding {
	opts, ok := parseRequiredColumnOptions(in.Options)
	if !ok {
		return nil
	}
	var out []finding.Finding
	for _, s := range in.Realm.Schemas {
		for _, t := range s.Tables {
			if !opts.applies(t.Name) {
				continue
			}
			obj := finding.Object{Schema: s.Name, Table: t.Name, Column: opts.column}
			col, found := t.Column(opts.column)
			if !found {
				out = append(out, finding.Finding{
					Rule:    RequiredColumn,
					Object:  obj,
					Message: fmt.Sprintf("table %q has no column %q", t.Name, opts.column),
					Fix: fmt.Sprintf("Add a %q field to the model that maps to table %q and regenerate the schema.",
						opts.column, t.Name),
				})
				continue // the other checks need the column
			}
			if col.Type != nil && col.Type.Null {
				out = append(out, finding.Finding{
					Rule:    RequiredColumn,
					Object:  obj,
					Message: fmt.Sprintf("column %q.%q is nullable", t.Name, opts.column),
					Fix: fmt.Sprintf("Make the %q field of the model that maps to table %q required (NOT NULL) and regenerate the schema.",
						opts.column, t.Name),
				})
			}
			if !hasLeadingIndex(t, opts.column) {
				out = append(out, finding.Finding{
					Rule:    RequiredColumn,
					Object:  obj,
					Message: fmt.Sprintf("column %q.%q is not the first column of any index", t.Name, opts.column),
					Fix: fmt.Sprintf("Add an index on table %q that starts with %q (declare it on the model) and regenerate the schema.",
						t.Name, opts.column),
				})
			}
			for _, idx := range t.Indexes {
				if idx.Unique && !leadsWith(idx, opts.column) {
					out = append(out, finding.Finding{
						Rule:    RequiredColumn,
						Object:  finding.Object{Schema: s.Name, Table: t.Name, Index: idx.Name},
						Message: fmt.Sprintf("unique index %q on %q does not start with %q", idx.Name, t.Name, opts.column),
						Fix: fmt.Sprintf("Declare the unique index %q of table %q with %q as its first field, then regenerate the schema.",
							idx.Name, t.Name, opts.column),
					})
				}
			}
			if !hasForeignKey(t, col) {
				out = append(out, finding.Finding{
					Rule:    RequiredColumn,
					Object:  obj,
					Message: fmt.Sprintf("column %q.%q has no foreign key", t.Name, opts.column),
					Fix: fmt.Sprintf("Declare a relation from the %q field of the model that maps to table %q to the table it refers to, and regenerate the schema.",
						opts.column, t.Name),
				})
			}
		}
	}
	return out
}
