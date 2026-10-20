package rules

import (
	"fmt"
	"strings"

	"ariga.io/atlas/sql/schema"

	"github.com/mokevnin/schemalens/internal/dialect"
	"github.com/mokevnin/schemalens/internal/finding"
)

// PrefixRedundantIndex is the id of the rule below.
const PrefixRedundantIndex = "prefix-redundant-index"

func init() {
	register(Rule{
		ID:       PrefixRedundantIndex,
		Dialects: dialect.All,
		Explain: Explanation{
			Why:       "A B-tree index on (a) is answered by any index or primary key that starts with (a), such as (a, b), so the shorter index adds write cost and storage and no query benefit. Only plain non-unique indexes are reported: a unique index also enforces a constraint and is never redundant, and a partial, expression, covering (INCLUDE) or non-B-tree index is not interchangeable with a longer one.",
			Fix:       "Remove the shorter index from the model and regenerate the schema; the finding names the index to drop and the longer one that serves its queries.",
			Exception: "The shorter index is much smaller and a hot query is index-only on it, or it is kept during a migration; document the measurement or the plan.",
		},
		Check: checkPrefixRedundantIndex,
	})
}

func checkPrefixRedundantIndex(in Input) []finding.Finding {
	var out []finding.Finding
	for _, s := range in.Realm.Schemas {
		for _, t := range s.Tables {
			out = append(out, prefixRedundantIndexes(s, t)...)
		}
	}
	return out
}

type coverer struct {
	idx    *schema.Index // nil for the primary key
	desc   string
	shape  indexShape
	unique bool
}

func prefixRedundantIndexes(s *schema.Schema, t *schema.Table) []finding.Finding {
	var coverers []coverer
	pk, hasPK := primaryKeyShape(t)
	if hasPK {
		coverers = append(coverers, coverer{nil, "the primary key", pk, true})
	}
	for _, idx := range t.Indexes {
		coverers = append(coverers, coverer{idx, fmt.Sprintf("index %q", idx.Name), shapeOf(idx), idx.Unique})
	}

	var out []finding.Finding
	for _, idx := range t.Indexes {
		shape := shapeOf(idx)
		if idx.Unique || !shape.btree() || shape.include != "" || len(shape.parts) == 0 {
			continue
		}
		for _, c := range coverers {
			if c.idx == idx || !c.shape.btree() || c.shape.predicate != shape.predicate {
				continue
			}
			if !shape.hasPrefix(c.shape) {
				continue
			}
			// Same keys and equally non-unique: that is a duplicate, reported by
			// duplicate-index, not a prefix.
			if len(shape.parts) == len(c.shape.parts) && !c.unique && shape.include == c.shape.include {
				continue
			}
			// A primary key of the same keys is a duplicate too.
			if len(shape.parts) == len(c.shape.parts) && c.idx == nil {
				continue
			}
			out = append(out, finding.Finding{
				Rule:   PrefixRedundantIndex,
				Object: finding.Object{Schema: s.Name, Table: t.Name, Index: idx.Name},
				Message: fmt.Sprintf("index %q on table %q is redundant: its keys (%s) are the leading keys of %s",
					idx.Name, t.Name, strings.Join(indexColumns(idx), ", "), c.desc),
				Fix: fmt.Sprintf("Remove %q from the model (%s already serves its queries), then regenerate the schema.", idx.Name, c.desc),
			})
			break
		}
	}
	return out
}
