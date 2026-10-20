package rules

import (
	"fmt"
	"strings"

	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlite"
)

// indexShape describes what an index can be used for, independent of its
// name. Two indexes with equal shapes serve exactly the same queries.
type indexShape struct {
	method    string   // access method, lower case; "" is the default B-tree
	predicate string   // partial index predicate, "" when the index is total
	include   string   // non-key INCLUDE columns, "" when none
	parts     []string // one key per key part: column or expression, order, attributes
}

func shapeOf(idx *schema.Index) indexShape {
	s := indexShape{}
	var include []string
	for _, a := range idx.Attrs {
		switch a := a.(type) {
		case *postgres.IndexType:
			s.method = strings.ToLower(a.T)
		case *mysql.IndexType:
			s.method = strings.ToLower(a.T)
		case *postgres.IndexPredicate:
			s.predicate = a.P
		case *sqlite.IndexPredicate:
			s.predicate = a.P
		case *postgres.IndexInclude:
			for _, c := range a.Columns {
				include = append(include, c.Name)
			}
		}
	}
	if s.method == "btree" {
		s.method = ""
	}
	s.include = strings.Join(include, ",")
	for _, p := range idx.Parts {
		s.parts = append(s.parts, partKey(p))
	}
	return s
}

// partName is the column name, or the expression text, a key part indexes.
func partName(p *schema.IndexPart) string {
	if p.C != nil {
		return p.C.Name
	}
	if x, ok := p.X.(*schema.RawExpr); ok {
		return x.X
	}
	return fmt.Sprint(p.X)
}

// partKey identifies one key part: what it indexes, its sort order and any
// attribute that changes how it can be searched (operator class, prefix length).
func partKey(p *schema.IndexPart) string {
	var b strings.Builder
	if p.C != nil {
		b.WriteString("c:")
	} else {
		b.WriteString("x:")
	}
	b.WriteString(partName(p))
	if p.Desc {
		b.WriteString(" desc")
	}
	for _, a := range p.Attrs {
		fmt.Fprintf(&b, " %+v", a)
	}
	return b.String()
}

// indexColumns lists the column names (or expression text) an index is keyed on.
func indexColumns(idx *schema.Index) []string {
	out := make([]string, len(idx.Parts))
	for i, p := range idx.Parts {
		out[i] = partName(p)
	}
	return out
}

// btree reports whether the index is an ordered index, the only kind where a
// shorter index is a prefix of a longer one.
func (s indexShape) btree() bool { return s.method == "" }

// hasPrefix reports whether the key parts of s are the leading key parts of o.
func (s indexShape) hasPrefix(o indexShape) bool {
	if len(s.parts) > len(o.parts) {
		return false
	}
	for i, p := range s.parts {
		if o.parts[i] != p {
			return false
		}
	}
	return true
}

func (s indexShape) equal(o indexShape) bool {
	return s.method == o.method && s.predicate == o.predicate && s.include == o.include &&
		len(s.parts) == len(o.parts) && s.hasPrefix(o)
}

// primaryKeyShape returns the shape of a table's primary key viewed as an
// index, or false when the table has none.
func primaryKeyShape(t *schema.Table) (indexShape, bool) {
	if t.PrimaryKey == nil || len(t.PrimaryKey.Parts) == 0 {
		return indexShape{}, false
	}
	return shapeOf(t.PrimaryKey), true
}
