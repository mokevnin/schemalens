package rules

import (
	"fmt"

	"ariga.io/atlas/sql/schema"

	"github.com/mokevnin/schemalens/internal/dialect"
	"github.com/mokevnin/schemalens/internal/finding"
)

// RequiredFieldNullableColumn is the id of the rule below.
const RequiredFieldNullableColumn = "required-field-nullable-column"

func init() {
	register(Rule{
		ID:       RequiredFieldNullableColumn,
		Dialects: dialect.All,
		Explain: Explanation{
			Why:       "The model promises a value is always present, so the application skips null handling. Only a NOT NULL column makes the database enforce that promise; otherwise another writer or a bug can store null and break the assumption later.",
			Fix:       "Keep the field required in the model and make the column NOT NULL: regenerate the migration from the model, backfill existing nulls, and apply it.",
			Exception: "A column that is intentionally filled in a second step, for example by a backfill job, while the model is already stricter; write the reason down and remove the exclusion afterwards.",
		},
		Check: checkRequiredFieldNullableColumn,
	})
}

// inPrimaryKey reports whether c is part of the table's primary key. Such a
// column cannot hold null even when the engine reports it as nullable (SQLite
// reports INTEGER PRIMARY KEY, the rowid alias, that way).
func inPrimaryKey(mt modelTable, c *schema.Column) bool {
	if mt.DB.PrimaryKey == nil {
		return false
	}
	for _, p := range mt.DB.PrimaryKey.Parts {
		if p.C != nil && p.C.Name == c.Name {
			return true
		}
	}
	return false
}

func checkRequiredFieldNullableColumn(in Input) []finding.Finding {
	var out []finding.Finding
	for _, mt := range modelTables(in) {
		for _, mc := range mt.Model.Columns {
			if !mc.Required {
				continue
			}
			dc, ok := mt.DB.Column(mc.Name)
			if !ok || !dc.Type.Null || inPrimaryKey(mt, dc) {
				continue
			}
			out = append(out, finding.Finding{
				Rule:    RequiredFieldNullableColumn,
				Object:  finding.Object{Schema: mt.Schema, Table: mt.DB.Name, Column: dc.Name},
				Message: fmt.Sprintf("model field for column %q of table %q is required, but the column is nullable", dc.Name, mt.DB.Name),
				Fix: fmt.Sprintf("Make column %q of table %q NOT NULL: keep the field required in the model, "+
					"then regenerate the migration and apply it so the database enforces what the model promises.", dc.Name, mt.DB.Name),
				Source: sourceOf(mt.Model, mc),
			})
		}
	}
	return out
}
