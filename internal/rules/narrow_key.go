package rules

import (
	"fmt"
	"strings"

	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/schema"

	"github.com/mokevnin/schemalens/internal/dialect"
	"github.com/mokevnin/schemalens/internal/finding"
)

// NarrowKey is the id of the rule below.
const NarrowKey = "narrow-key"

func init() {
	register(Rule{
		ID: NarrowKey,
		// SQLite stores every integer in up to 8 bytes whatever its declared
		// type (INT, SMALLINT and BIGINT are the same), so there is no narrow
		// integer key there and the rule does not run.
		Dialects: []dialect.Dialect{dialect.Postgres, dialect.MySQL},
		Explain: Explanation{
			Why:       "A single-column integer primary key or foreign key narrower than 64 bits runs out of values: 2^31 for a 32-bit key, far fewer for 8, 16 or 24 bits. Widening a key later rewrites the table and every table that references it, under a long lock. Composite keys are not checked; their space is the product of their columns.",
			Fix:       "Declare the key field as a 64-bit integer in the model (bigint, int64) on the primary key and on every foreign key that references it, then regenerate the schema.",
			Exception: "A small, bounded lookup table (statuses, countries) whose row count can never approach the limit, and the references to it.",
		},
		Check: checkNarrowKey,
	})
}

func checkNarrowKey(in Input) []finding.Finding {
	var out []finding.Finding
	for _, s := range in.Realm.Schemas {
		for _, t := range s.Tables {
			if t.PrimaryKey != nil && len(t.PrimaryKey.Parts) == 1 && t.PrimaryKey.Parts[0].C != nil {
				c := t.PrimaryKey.Parts[0].C
				if bits, name, ok := narrowInteger(c); ok {
					out = append(out, finding.Finding{
						Rule:   NarrowKey,
						Object: finding.Object{Schema: s.Name, Table: t.Name, Column: c.Name},
						Message: fmt.Sprintf("primary key column %q on table %q is %s (%d bits), narrower than 64 bits",
							c.Name, t.Name, name, bits),
						Fix: fmt.Sprintf("Declare the primary key field mapped to column %q as a 64-bit integer (bigint), also on the foreign keys that reference it, then regenerate the schema.", c.Name),
					})
				}
			}
			for _, fk := range t.ForeignKeys {
				if len(fk.Columns) != 1 {
					continue
				}
				c := fk.Columns[0]
				if bits, name, ok := narrowInteger(c); ok {
					out = append(out, finding.Finding{
						Rule:   NarrowKey,
						Object: finding.Object{Schema: s.Name, Table: t.Name, Column: c.Name, ForeignKey: fk.Symbol},
						Message: fmt.Sprintf("foreign key %q column %q on table %q is %s (%d bits), narrower than 64 bits",
							fk.Symbol, c.Name, t.Name, name, bits),
						Fix: fmt.Sprintf("Declare the field mapped to column %q as a 64-bit integer (bigint), matching the key it references, then regenerate the schema.", c.Name),
					})
				}
			}
		}
	}
	return out
}

// narrowInteger reports the width in bits and the declared type name of an
// integer column narrower than 64 bits.
func narrowInteger(c *schema.Column) (bits int, name string, ok bool) {
	if c.Type == nil {
		return 0, "", false
	}
	var t *schema.IntegerType
	switch typ := c.Type.Type.(type) {
	case *schema.IntegerType:
		t = typ
	case *postgres.SerialType:
		t = typ.IntegerType()
	default:
		return 0, "", false
	}
	switch strings.ToLower(t.T) {
	case "tinyint":
		bits = 8
	case "smallint", "int2":
		bits = 16
	case "mediumint":
		bits = 24
	case "int", "integer", "int4":
		bits = 32
	default:
		return 0, "", false // bigint, int8 and anything unknown
	}
	name = strings.ToLower(t.T)
	if s, isSerial := c.Type.Type.(*postgres.SerialType); isSerial {
		name = s.T
	}
	return bits, name, true
}
