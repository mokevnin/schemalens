package rules

import (
	"fmt"

	"ariga.io/atlas/sql/schema"

	"github.com/mokevnin/schemalens/internal/dialect"
	"github.com/mokevnin/schemalens/internal/finding"
)

// NullableBoolean is the id of the rule below.
const NullableBoolean = "nullable-boolean"

func init() {
	register(Rule{
		ID: NullableBoolean,
		// PostgreSQL has a boolean type. MySQL's BOOLEAN is TINYINT(1) and is
		// recognised by that declaration. SQLite has no boolean type at all; its
		// columns declared BOOL or BOOLEAN are the ones an ORM maps to bool, so
		// those are checked and every other column is out of reach.
		Dialects: dialect.All,
		Explain: Explanation{
			Why:       "A nullable boolean has three states: true, false and NULL. Code reads it as two, so NULL silently behaves as false in some places and as an error in others, and queries like WHERE flag = false skip the NULL rows.",
			Fix:       "Make the field non-nullable in the model, give it a default, and backfill existing NULL rows before regenerating the schema. If a third state is real, model it explicitly, for example as an enum.",
			Exception: "NULL genuinely means \"not answered yet\" and the code treats it as its own state, for example an optional consent flag.",
		},
		Check: checkNullableBoolean,
	})
}

func checkNullableBoolean(in Input) []finding.Finding {
	var out []finding.Finding
	for _, s := range in.Realm.Schemas {
		for _, t := range s.Tables {
			for _, c := range t.Columns {
				if c.Type == nil || !c.Type.Null || !isBoolean(c.Type.Type) {
					continue
				}
				out = append(out, finding.Finding{
					Rule:    NullableBoolean,
					Object:  finding.Object{Schema: s.Name, Table: t.Name, Column: c.Name},
					Message: fmt.Sprintf("boolean column %q on table %q is nullable, so it has three states: true, false and NULL", c.Name, t.Name),
					Fix: fmt.Sprintf("Make the field mapped to column %q required (non-nullable) with a default, backfill existing NULL rows, then regenerate the schema.",
						c.Name),
				})
			}
		}
	}
	return out
}

func isBoolean(t schema.Type) bool {
	_, ok := t.(*schema.BoolType)
	return ok
}
