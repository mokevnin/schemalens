package rules

import (
	"fmt"
	"slices"
	"strings"

	"ariga.io/atlas/sql/schema"

	"github.com/mokevnin/schemalens/internal/dialect"
	"github.com/mokevnin/schemalens/internal/finding"
)

// DuplicateIndex is the id of the rule below.
const DuplicateIndex = "duplicate-index"

func init() {
	register(Rule{
		ID:       DuplicateIndex,
		Dialects: dialect.All,
		Explain: Explanation{
			Why:       "Two indexes with the same keys, order, uniqueness and predicate serve exactly the same queries. The copy only adds write cost and storage: every insert and update maintains both. The same holds for an index that repeats the primary key.",
			Fix:       "Remove the repeated index from the model (a duplicated index tag, or an index that restates the primary key) and regenerate the schema. The finding names the index to drop and the one that stays.",
			Exception: "Rare: an index deliberately duplicated while another is built under it for a zero-downtime rebuild; remove the exclusion once the old index is dropped.",
		},
		Check: checkDuplicateIndex,
	})
}

func checkDuplicateIndex(in Input) []finding.Finding {
	var out []finding.Finding
	for _, s := range in.Realm.Schemas {
		for _, t := range s.Tables {
			out = append(out, duplicateIndexes(s, t)...)
		}
	}
	return out
}

func duplicateIndexes(s *schema.Schema, t *schema.Table) []finding.Finding {
	var out []finding.Finding
	report := func(idx *schema.Index, keeps string) {
		kind := "index"
		if idx.Unique {
			kind = "unique index"
		}
		out = append(out, finding.Finding{
			Rule:   DuplicateIndex,
			Object: finding.Object{Schema: s.Name, Table: t.Name, Index: idx.Name},
			Message: fmt.Sprintf("%s %q on table %q duplicates %s: both are keyed on (%s)",
				kind, idx.Name, t.Name, keeps, strings.Join(indexColumns(idx), ", ")),
			Fix: fmt.Sprintf("Remove %q from the model (it adds write cost and no query benefit), then regenerate the schema.", idx.Name),
		})
	}

	// Name order, so the alphabetically first index of a group is the one kept.
	idxs := slices.Clone(t.Indexes)
	slices.SortStableFunc(idxs, func(a, b *schema.Index) int { return strings.Compare(a.Name, b.Name) })

	pk, hasPK := primaryKeyShape(t)
	var kept []*schema.Index
	for _, idx := range idxs {
		shape := shapeOf(idx)
		if hasPK && shape.equal(pk) {
			report(idx, "the primary key")
			continue
		}
		if i := slices.IndexFunc(kept, func(k *schema.Index) bool {
			return k.Unique == idx.Unique && shapeOf(k).equal(shape)
		}); i >= 0 {
			report(idx, fmt.Sprintf("index %q", kept[i].Name))
			continue
		}
		kept = append(kept, idx)
	}
	return out
}
