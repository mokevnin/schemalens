package gormadapter

import (
	"bytes"
	"encoding/json"
	"go/types"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/mokevnin/schemalens/adapters/staticmodel"
)

// joinFor maps a join table to the tables it joins.
type joinFor map[staticmodel.TableRef][]staticmodel.TableRef

// keyField is a field of a model that a many2many relation keys on.
type keyField struct{ field, col string }

// addJoinTables adds the join table of every many2many field, as gorm's
// AutoMigrate creates it: the name from the tag (through the naming strategy,
// or verbatim from the "tables" override), one column per key field of each
// side, and a primary key over all of them. The join table is declared for the
// two tables it joins. Both sides of a relation usually declare the same join
// table; it is added once.
func addJoinTables(all []*built, cfg *config) ([]staticmodel.Table, joinFor) {
	byType := map[string]*built{}
	byName := map[staticmodel.TableRef]bool{}
	for _, b := range all {
		byType[b.strukt.PkgPath+"."+b.strukt.Name] = b
		byName[b.table.Ref()] = true
	}
	namer := cfg.namer()
	var tables []staticmodel.Table
	joins := joinFor{}
	for _, owner := range all {
		for _, f := range owner.relFields {
			tag := parseTag(f.Tag.Get("gorm"))
			tagged := tag["MANY2MANY"]
			if tagged == "" {
				continue
			}
			related := byType[staticmodel.NamedType(elemType(f.Type))]
			if related == nil {
				continue
			}
			ownKeys, ok1 := keyFields(owner, splitList(tag["FOREIGNKEY"]))
			refKeys, ok2 := keyFields(related, splitList(tag["REFERENCES"]))
			if !ok1 || !ok2 || len(ownKeys) == 0 || len(refKeys) == 0 {
				continue
			}
			name := namer.JoinTableName(tagged)
			if v, ok := cfg.Tables[tagged]; ok {
				name = v
			}
			join := staticmodel.Table{Columns: []staticmodel.Column{}}
			join.SplitSchema(name)
			ref := join.Ref()

			ownCols := map[string]bool{}
			joinKeys := splitList(tag["JOINFOREIGNKEY"])
			var cols []string
			for i, k := range ownKeys {
				fieldName := upperFirst(owner.strukt.Name) + k.field
				if i < len(joinKeys) {
					fieldName = upperFirst(joinKeys[i])
				}
				ownCols[fieldName] = true
				cols = append(cols, namer.ColumnName("", fieldName))
			}
			refNames := splitList(tag["JOINREFERENCES"])
			for i, k := range refKeys {
				fieldName := upperFirst(related.strukt.Name) + k.field
				if i < len(refNames) {
					fieldName = upperFirst(refNames[i])
				}
				if ownCols[fieldName] {
					// A table joined to itself: gorm tells the two sides apart.
					if f.Name != related.strukt.Name {
						fieldName = upperFirst(f.Name) + k.field
					} else {
						fieldName += "Reference"
					}
				}
				cols = append(cols, namer.ColumnName("", fieldName))
			}

			if _, seen := joins[ref]; !seen {
				joins[ref] = []staticmodel.TableRef{owner.table.Ref(), related.table.Ref()}
				if !byName[ref] {
					join.Source = staticmodel.ToSource(f.Pos)
					join.PrimaryKey = cols
					for _, c := range cols {
						join.Columns = append(join.Columns, staticmodel.Column{Name: c, Required: true, Source: join.Source})
					}
					tables = append(tables, join)
				}
			}
		}
	}
	return tables, joins
}

// keyFields resolves the fields a relation keys on: the named ones, or the
// primary key. A name may be a Go field or its column.
func keyFields(b *built, names []string) ([]keyField, bool) {
	var out []keyField
	if len(names) == 0 {
		for _, col := range b.table.PrimaryKey {
			found := false
			for field, c := range b.fieldCols {
				if c == col {
					out = append(out, keyField{field, col})
					found = true
					break
				}
			}
			if !found {
				return nil, false
			}
		}
		return out, true
	}
	for _, n := range names {
		if col, ok := b.fieldCols[n]; ok {
			out = append(out, keyField{n, col})
			continue
		}
		found := false
		for field, c := range b.fieldCols {
			if c == n {
				out = append(out, keyField{field, c})
				found = true
				break
			}
		}
		if !found {
			return nil, false
		}
	}
	return out, true
}

// elemType is the associated struct type of a relation field: the element of
// a slice, through pointers.
func elemType(t types.Type) types.Type {
	t = staticmodel.Deref(t)
	if sl, ok := t.Underlying().(*types.Slice); ok {
		return staticmodel.Deref(sl.Elem())
	}
	return t
}

func splitList(v string) []string {
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func upperFirst(s string) string {
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[n:]
}

// withJoinTableFor rewrites the manifest document Run printed, adding
// "join_table_for" to the tables that are join tables. staticmodel's manifest
// type has no such field in the tagged version this module requires.
func withJoinTableFor(out io.Writer, doc []byte, joins joinFor) error {
	var m struct {
		Version int `json:"version"`
		Tables  []struct {
			staticmodel.Table
			JoinTableFor []staticmodel.TableRef `json:"join_table_for,omitempty"`
		} `json:"tables"`
	}
	if err := json.Unmarshal(doc, &m); err != nil {
		return err
	}
	for i := range m.Tables {
		m.Tables[i].JoinTableFor = joins[m.Tables[i].Ref()]
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(m); err != nil {
		return err
	}
	_, err := out.Write(buf.Bytes())
	return err
}
