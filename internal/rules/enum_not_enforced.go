package rules

import (
	"fmt"
	"slices"

	"ariga.io/atlas/sql/schema"

	"github.com/mokevnin/schemalens/internal/dialect"
	"github.com/mokevnin/schemalens/internal/finding"
)

// EnumNotEnforced is the id of the rule below.
const EnumNotEnforced = "enum-not-enforced"

func init() {
	register(Rule{
		ID:       EnumNotEnforced,
		Dialects: dialect.All,
		Explain: Explanation{
			Why:       "The model promises a field holds only a fixed set of values, and code switches over them without a fallback. Unless the database refuses other values, any writer can store a string the application has never heard of and break that code later. The database enforces the set through a native enum type (PostgreSQL enum, MySQL ENUM) or a CHECK constraint listing the values.",
			Fix:       "Keep the enum in the model and regenerate the migration so the column becomes a native enum or gets a CHECK (col IN (...)) listing exactly the model's values. Fix existing rows with other values first.",
			Exception: "A column that intentionally accepts more values than the model while a new value is rolled out, or whose values are policed by a lookup table; write the reason down. A database set that differs from the model's, not only a missing one, is reported too: the finding lists what the database allows.",
		},
		Check: checkEnumNotEnforced,
	})
}

func checkEnumNotEnforced(in Input) []finding.Finding {
	var out []finding.Finding
	for _, mt := range modelTables(in) {
		checks := tableChecks(mt.DB)
		for _, mc := range mt.Model.Columns {
			if len(mc.Enum) == 0 {
				continue
			}
			dc, ok := mt.DB.Column(mc.Name)
			if !ok {
				continue
			}
			detail, enforced := enumEnforcement(dc, checks, mc.Enum)
			if enforced {
				continue
			}
			out = append(out, finding.Finding{
				Rule:   EnumNotEnforced,
				Object: finding.Object{Schema: mt.Schema, Table: mt.DB.Name, Column: dc.Name},
				Message: fmt.Sprintf("model restricts column %q of table %q to the values %s, but the database does not enforce that: %s",
					dc.Name, mt.DB.Name, describeSet(mc.Enum), detail),
				Fix: fmt.Sprintf("Restrict column %q of table %q to the values %s: keep the enum in the model, "+
					"then regenerate the migration so the column becomes a native enum or gets a CHECK listing exactly these values, and apply it.",
					dc.Name, mt.DB.Name, describeSet(mc.Enum)),
				Source: sourceOf(mt.Model, mc),
			})
		}
	}
	return out
}

// enumEnforcement reports whether a native enum type or a CHECK allows
// exactly the model's values. When it does not, detail says what the database
// does allow.
func enumEnforcement(c *schema.Column, checks []*schema.Check, want []string) (detail string, enforced bool) {
	detail = "the column is neither a native enum nor covered by a CHECK listing values"
	want = nameSet(want)
	if et, ok := c.Type.Type.(*schema.EnumType); ok {
		if slices.Equal(nameSet(et.Values), want) {
			return "", true
		}
		return fmt.Sprintf("the column's enum type allows %s", describeSet(et.Values)), false
	}
	for _, ck := range checks {
		if vals, ok := checkEnumValues(ck.Expr, c.Name); ok {
			if slices.Equal(nameSet(vals), want) {
				return "", true
			}
			detail = fmt.Sprintf("a CHECK constraint allows %s", describeSet(vals))
		}
	}
	return detail, false
}
