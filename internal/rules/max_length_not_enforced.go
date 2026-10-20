package rules

import (
	"fmt"
	"slices"
	"strings"

	"ariga.io/atlas/sql/schema"

	"github.com/mokevnin/schemalens/internal/dialect"
	"github.com/mokevnin/schemalens/internal/finding"
)

// MaxLengthNotEnforced is the id of the rule below.
const MaxLengthNotEnforced = "max-length-not-enforced"

func init() {
	register(Rule{
		ID:       MaxLengthNotEnforced,
		Dialects: dialect.All,
		Explain: Explanation{
			Why:       "The model promises a string never exceeds a length, and validation in the application is the only thing keeping that true unless the database also refuses longer values. A column without a limit (TEXT, unbounded VARCHAR) or with a larger one accepts whatever another writer or a bug stores, and the model's assumptions about size (UI, indexes, downstream limits) break later.",
			Fix:       "Keep the maximum length in the model and regenerate the migration: it should use a length-limited type (VARCHAR(n) or CHAR(n) on PostgreSQL and MySQL) or a CHECK on the length (char_length(col) <= n; length(col) <= n on SQLite). Trim or fix existing longer values first.",
			Exception: "A limit that the model enforces only for input validation while the column is intentionally wider, for example during a staged migration; write the reason down. SQLite ignores the n of VARCHAR(n), so only a CHECK counts there.",
		},
		Check: checkMaxLengthNotEnforced,
	})
}

func checkMaxLengthNotEnforced(in Input) []finding.Finding {
	var out []finding.Finding
	for _, mt := range modelTables(in) {
		checks := tableChecks(mt.DB)
		for _, mc := range mt.Model.Columns {
			if mc.MaxLen <= 0 {
				continue
			}
			dc, ok := mt.DB.Column(mc.Name)
			if !ok {
				continue
			}
			detail, enforced := lengthEnforcement(in.Dialect, dc, checks, mc.MaxLen)
			if enforced {
				continue
			}
			out = append(out, finding.Finding{
				Rule:   MaxLengthNotEnforced,
				Object: finding.Object{Schema: mt.Schema, Table: mt.DB.Name, Column: dc.Name},
				Message: fmt.Sprintf("model limits column %q of table %q to %d characters, but the database does not enforce it: %s",
					dc.Name, mt.DB.Name, mc.MaxLen, detail),
				Fix: fmt.Sprintf("Limit column %q of table %q to %d characters: keep the maximum length in the model, "+
					"then regenerate the migration so the column gets a length-limited type or a CHECK on its length, and apply it.",
					dc.Name, mt.DB.Name, mc.MaxLen),
				Source: sourceOf(mt.Model, mc),
			})
		}
	}
	return out
}

// lengthEnforcement reports whether the column's type or a CHECK caps the
// string at max characters or fewer. When it does not, detail says why.
func lengthEnforcement(d dialect.Dialect, c *schema.Column, checks []*schema.Check, limit int) (detail string, enforced bool) {
	detail = "neither the column type nor a CHECK constraint limits its length"
	// SQLite parses VARCHAR(n) but never enforces n.
	if st, ok := c.Type.Type.(*schema.StringType); ok && d != dialect.SQLite && isCharType(st.T) && st.Size > 0 {
		if st.Size <= limit {
			return "", true
		}
		detail = fmt.Sprintf("the column type allows %d characters", st.Size)
	}
	for _, ck := range checks {
		if n, ok := checkMaxLength(ck.Expr, c.Name); ok {
			if n <= limit {
				return "", true
			}
			detail = fmt.Sprintf("a CHECK constraint allows %d characters", n)
		}
	}
	return detail, false
}

func isCharType(t string) bool {
	return slices.Contains([]string{"varchar", "char", "character varying", "character", "bpchar", "nvarchar", "nchar"}, strings.ToLower(t))
}
