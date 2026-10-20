package rules

import (
	"fmt"
	"slices"

	"ariga.io/atlas/sql/schema"

	"github.com/mokevnin/schemalens/internal/dialect"
	"github.com/mokevnin/schemalens/internal/finding"
)

// RequiredColumn is the id of the rule that owns the options of the
// required-column family. It reports nothing itself: each violation is its own
// rule below, so that each can be excluded, baselined or disabled on its own.
const RequiredColumn = "required-column"

// The violations of the required-column rule, one rule id each.
const (
	RequiredColumnMissing      = "required-column-missing"
	RequiredColumnNullable     = "required-column-nullable"
	RequiredColumnNoForeignKey = "required-column-no-foreign-key"
	RequiredColumnNoIndex      = "required-column-no-leading-index"
	RequiredColumnUniqueIndex  = "required-column-unique-index"
)

const requiredColumnOptionsText = "The required-column rule is configured under rules.required-column.options: column (required, the column name), " +
	"tables (optional, the tables it applies to; default all) and exclude_tables (optional, tables that are skipped). " +
	"Without the column option it does nothing. Each violation is reported under its own rule id, so an exclusion silences one violation only. "

func init() {
	register(Rule{
		ID:       RequiredColumn,
		Dialects: dialect.All,
		Explain: Explanation{
			Why: "Some columns must be present on every table of a kind, for example a column that scopes each row to its owner. " +
				requiredColumnOptionsText +
				"The violations are: " + RequiredColumnMissing + " (a table without the column), " + RequiredColumnNullable + " (a nullable column), " +
				RequiredColumnNoForeignKey + " (a column without a foreign key), " + RequiredColumnNoIndex + " (a column no index starts with) and " +
				RequiredColumnUniqueIndex + " (a unique index that does not start with the column). " +
				"Disabling required-column disables all of them.",
			Fix:       "Add the column to the model that maps to the table, make it required, declare its relation to the referenced table, and index it first; then regenerate the schema.",
			Exception: "Tables that are shared across all owners, such as reference data or the owner table itself, when the reason is written down.",
		},
	})
	for _, v := range []struct {
		id    string
		what  string
		fix   string
		check func(requiredColumnOptions, string, *schema.Table) []finding.Finding
	}{
		{RequiredColumnMissing, "a table that has no such column", "Add the column to the model that maps to the table and regenerate the schema.", checkRCMissing},
		{RequiredColumnNullable, "a required column that is nullable", "Make the field of the model that maps to the table required (NOT NULL) and regenerate the schema.", checkRCNullable},
		{RequiredColumnNoForeignKey, "a required column that has no foreign key", "Declare a relation from the field of the model to the table it refers to, and regenerate the schema.", checkRCNoForeignKey},
		{RequiredColumnNoIndex, "a required column that is not the first column of any index", "Declare an index on the model that starts with the column, and regenerate the schema.", checkRCNoIndex},
		{RequiredColumnUniqueIndex, "a unique index that does not start with the required column", "Declare the unique index on the model with the column as its first field, and regenerate the schema.", checkRCUniqueIndex},
	} {
		register(Rule{
			ID:       v.id,
			Parent:   RequiredColumn,
			Dialects: dialect.All,
			Explain: Explanation{
				Why: "This is the " + v.id + " violation of the required-column rule: " + v.what + ". " +
					"Some columns must be present on every table of a kind, for example a column that scopes each row to its owner. " +
					requiredColumnOptionsText + "The options are read from rules.required-column.options.",
				Fix:       v.fix,
				Exception: "Tables that are shared across all owners, such as reference data or the owner table itself, when the reason is written down.",
			},
			Check: requiredColumnCheck(v.check),
		})
	}
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

func requiredColumnCheck(check func(requiredColumnOptions, string, *schema.Table) []finding.Finding) func(Input) []finding.Finding {
	return func(in Input) []finding.Finding {
		var raw map[string]any
		if in.RuleOptions != nil {
			raw = in.RuleOptions(RequiredColumn)
		}
		opts, ok := parseRequiredColumnOptions(raw)
		if !ok {
			return nil
		}
		var out []finding.Finding
		for _, s := range in.Realm.Schemas {
			for _, t := range s.Tables {
				if opts.applies(t.Name) {
					out = append(out, check(opts, s.Name, t)...)
				}
			}
		}
		return out
	}
}

func checkRCMissing(o requiredColumnOptions, sch string, t *schema.Table) []finding.Finding {
	if _, found := t.Column(o.column); found {
		return nil
	}
	return []finding.Finding{{
		Rule:    RequiredColumnMissing,
		Object:  finding.Object{Schema: sch, Table: t.Name, Column: o.column},
		Message: fmt.Sprintf("table %q has no column %q", t.Name, o.column),
		Fix: fmt.Sprintf("Add a %q field to the model that maps to table %q and regenerate the schema.",
			o.column, t.Name),
	}}
}

func checkRCNullable(o requiredColumnOptions, sch string, t *schema.Table) []finding.Finding {
	col, found := t.Column(o.column)
	if !found || col.Type == nil || !col.Type.Null {
		return nil
	}
	return []finding.Finding{{
		Rule:    RequiredColumnNullable,
		Object:  finding.Object{Schema: sch, Table: t.Name, Column: o.column},
		Message: fmt.Sprintf("column %q.%q is nullable", t.Name, o.column),
		Fix: fmt.Sprintf("Make the %q field of the model that maps to table %q required (NOT NULL) and regenerate the schema.",
			o.column, t.Name),
	}}
}

func checkRCNoIndex(o requiredColumnOptions, sch string, t *schema.Table) []finding.Finding {
	if _, found := t.Column(o.column); !found || hasLeadingIndex(t, o.column) {
		return nil
	}
	return []finding.Finding{{
		Rule:    RequiredColumnNoIndex,
		Object:  finding.Object{Schema: sch, Table: t.Name, Column: o.column},
		Message: fmt.Sprintf("column %q.%q is not the first column of any index", t.Name, o.column),
		Fix: fmt.Sprintf("Add an index on table %q that starts with %q (declare it on the model) and regenerate the schema.",
			t.Name, o.column),
	}}
}

func checkRCUniqueIndex(o requiredColumnOptions, sch string, t *schema.Table) []finding.Finding {
	if _, found := t.Column(o.column); !found {
		return nil
	}
	var out []finding.Finding
	for _, idx := range t.Indexes {
		if idx.Unique && !leadsWith(idx, o.column) {
			out = append(out, finding.Finding{
				Rule:    RequiredColumnUniqueIndex,
				Object:  finding.Object{Schema: sch, Table: t.Name, Index: idx.Name},
				Message: fmt.Sprintf("unique index %q on %q does not start with %q", idx.Name, t.Name, o.column),
				Fix: fmt.Sprintf("Declare the unique index %q of table %q with %q as its first field, then regenerate the schema.",
					idx.Name, t.Name, o.column),
			})
		}
	}
	return out
}

func checkRCNoForeignKey(o requiredColumnOptions, sch string, t *schema.Table) []finding.Finding {
	col, found := t.Column(o.column)
	if !found || hasForeignKey(t, col) {
		return nil
	}
	return []finding.Finding{{
		Rule:    RequiredColumnNoForeignKey,
		Object:  finding.Object{Schema: sch, Table: t.Name, Column: o.column},
		Message: fmt.Sprintf("column %q.%q has no foreign key", t.Name, o.column),
		Fix: fmt.Sprintf("Declare a relation from the %q field of the model that maps to table %q to the table it refers to, and regenerate the schema.",
			o.column, t.Name),
	}}
}
