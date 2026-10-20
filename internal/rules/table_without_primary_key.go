package rules

import (
	"fmt"

	"github.com/mokevnin/schemalens/internal/dialect"
	"github.com/mokevnin/schemalens/internal/finding"
)

// TableWithoutPrimaryKey is the id of the rule below.
const TableWithoutPrimaryKey = "table-without-primary-key"

func init() {
	register(Rule{
		ID:       TableWithoutPrimaryKey,
		Dialects: dialect.All,
		Explain: Explanation{
			Why:       "Without a primary key rows cannot be addressed individually, replication and ORMs lose their handle on a row, and duplicates cannot be prevented.",
			Fix:       "Declare a primary key on the model that maps to the table and regenerate the schema.",
			Exception: "Append-only tables that are never updated or referenced, such as a raw event log, when the reason is written down.",
		},
		Check: checkTableWithoutPrimaryKey,
	})
}

func checkTableWithoutPrimaryKey(in Input) []finding.Finding {
	var out []finding.Finding
	for _, s := range in.Realm.Schemas {
		for _, t := range s.Tables {
			if t.PrimaryKey != nil && len(t.PrimaryKey.Parts) > 0 {
				continue
			}
			out = append(out, finding.Finding{
				Rule:    TableWithoutPrimaryKey,
				Object:  finding.Object{Schema: s.Name, Table: t.Name},
				Message: fmt.Sprintf("table %q has no primary key", t.Name),
				Fix: fmt.Sprintf("Declare a primary key for the model that maps to table %q "+
					"(for example an id field marked as the primary key), then regenerate the schema.", t.Name),
			})
		}
	}
	return out
}
