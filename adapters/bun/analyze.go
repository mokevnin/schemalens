package main

import (
	"fmt"
	"go/types"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/jinzhu/inflection"

	"github.com/mokevnin/schemalens/adapters/staticmodel"
)

const baseModel = "github.com/uptrace/bun/schema.BaseModel"

// analyze turns the project's bun structs into a manifest.
func analyze(prog *staticmodel.Program, cfg *config) (*staticmodel.Manifest, error) {
	embeddable := embeddedOnly(prog)
	m := &staticmodel.Manifest{Version: 1, Tables: []staticmodel.Table{}}
	var all []*built
	for _, s := range prog.Structs() {
		if slices.Contains(cfg.Exclude, s.Name) || slices.Contains(cfg.Exclude, s.PkgPath+"."+s.Name) {
			continue
		}
		if !isModel(s, cfg, embeddable) {
			continue
		}
		all = append(all, buildTable(s, cfg))
	}
	if err := addRelations(all); err != nil {
		return nil, err
	}
	for _, b := range all {
		m.Tables = append(m.Tables, b.table)
	}
	return m, nil
}

// built is a table under construction, with what relations need to find it.
type built struct {
	table     staticmodel.Table
	strukt    *staticmodel.Struct
	cols      map[string]bool // column names
	relFields []staticmodel.Field
}

// embeddedOnly collects structs that are embedded into another struct;
// unless they say otherwise they are fragments, not tables.
func embeddedOnly(prog *staticmodel.Program) map[string]bool {
	out := map[string]bool{}
	for _, s := range prog.Structs() {
		for _, f := range s.Fields {
			if isEmbedding(f) {
				out[staticmodel.NamedType(f.Type)] = true
			}
		}
	}
	return out
}

// isModel decides whether a struct maps to a table: it embeds bun.BaseModel,
// has a config entry, or carries bun tags and is not merely embedded
// elsewhere.
func isModel(s *staticmodel.Struct, cfg *config, embeddable map[string]bool) bool {
	if _, ok := lookupTable(cfg, s); ok {
		return true
	}
	hasTag := false
	for _, f := range s.Fields {
		if f.Embedded && staticmodel.NamedType(f.Type) == baseModel {
			return true
		}
		if f.Tag.Get("bun") != "" {
			hasTag = true
		}
	}
	return hasTag && !embeddable[s.PkgPath+"."+s.Name]
}

func lookupTable(cfg *config, s *staticmodel.Struct) (string, bool) {
	if v, ok := cfg.Tables[s.PkgPath+"."+s.Name]; ok {
		return v, true
	}
	v, ok := cfg.Tables[s.Name]
	return v, ok
}

// tableName follows bun: the config override, then the table (or bare name)
// in the bun.BaseModel tag, then the struct name underscored and pluralized.
func tableName(s *staticmodel.Struct, cfg *config) string {
	if v, ok := lookupTable(cfg, s); ok {
		return v
	}
	for _, f := range s.Fields {
		if !f.Embedded || staticmodel.NamedType(f.Type) != baseModel {
			continue
		}
		t := parseTag(f.Tag.Get("bun"))
		if v, ok := t.option("table"); ok {
			return v
		}
		if t.Name != "" {
			return t.Name
		}
	}
	name := underscore(s.Name)
	if cfg.Naming.SingularTable {
		return name
	}
	return inflection.Plural(name)
}

// underscore converts "CamelCasedString" to "camel_cased_string" exactly as
// bun does (its own helper is internal to the bun module).
func underscore(s string) string {
	isUpper := func(c byte) bool { return c >= 'A' && c <= 'Z' }
	isLower := func(c byte) bool { return c >= 'a' && c <= 'z' }
	r := make([]byte, 0, len(s)+5)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if isUpper(c) {
			if i > 0 && i+1 < len(s) && (isLower(s[i-1]) || isLower(s[i+1])) {
				r = append(r, '_')
			}
			r = append(r, c+32)
			continue
		}
		r = append(r, c)
	}
	return string(r)
}

// isEmbedding reports whether bun flattens the field into the parent table:
// an anonymous struct field (other than bun.BaseModel), or a struct field
// tagged "embed".
func isEmbedding(f staticmodel.Field) bool {
	t := parseTag(f.Tag.Get("bun"))
	if t.Name == "-" && t.Options == nil {
		return false
	}
	if _, ok := f.Type.Underlying().(*types.Struct); !ok {
		p, isPtr := f.Type.(*types.Pointer)
		if !isPtr {
			return false
		}
		if _, ok := p.Elem().Underlying().(*types.Struct); !ok {
			return false
		}
	}
	if staticmodel.NamedType(f.Type) == baseModel {
		return false
	}
	return f.Embedded || t.has("embed")
}

// embedPrefix is the column prefix contributed by the embedding chain.
func embedPrefix(f staticmodel.Field) string {
	var b strings.Builder
	for _, p := range f.Path {
		v, _ := parseTag(p.Tag.Get("bun")).option("embed")
		b.WriteString(v)
	}
	return b.String()
}

func buildTable(s *staticmodel.Struct, cfg *config) *built {
	b := &built{strukt: s, cols: map[string]bool{}}
	t := &b.table
	*t = staticmodel.Table{Source: staticmodel.ToSource(s.Pos), Columns: []staticmodel.Column{}}
	t.SplitSchema(tableName(s, cfg))

	groups := map[string][]string{} // named unique groups: name -> columns
	var groupOrder []string
	depth := map[string]int{}
	for _, f := range s.Flatten(isEmbedding) {
		if !f.Exported || staticmodel.NamedType(f.Type) == baseModel {
			continue
		}
		tg := parseTag(f.Tag.Get("bun"))
		if tg.Name == "-" && tg.Options == nil {
			continue
		}
		if tg.has("rel") || tg.has("m2m") {
			b.relFields = append(b.relFields, f)
			continue
		}
		if tg.has("scanonly") {
			continue
		}
		prefix := embedPrefix(f)
		col := underscore(f.Name)
		if tg.Name != "" {
			col = tg.Name
		}
		if v, ok := tg.option("column"); ok {
			col = v
		}
		col = prefix + col

		// A field declared directly on the struct wins over an embedded one
		// with the same column, as in bun.
		if d, seen := depth[col]; seen {
			if len(f.Path) >= d {
				continue
			}
			t.Columns = slices.DeleteFunc(t.Columns, func(c staticmodel.Column) bool { return c.Name == col })
			t.PrimaryKey = slices.DeleteFunc(t.PrimaryKey, func(c string) bool { return c == col })
			for name, cols := range groups {
				groups[name] = slices.DeleteFunc(cols, func(c string) bool { return c == col })
			}
		}
		depth[col] = len(f.Path)

		isPK := tg.has("pk")
		c := staticmodel.Column{
			Name:     col,
			Required: tg.has("notnull") || isPK || tg.has("autoincrement"),
			Source:   staticmodel.ToSource(f.Pos),
		}
		if typ, ok := tg.option("type"); ok {
			c.MaxLen = charLength(typ)
		}
		for _, name := range uniqueGroups(tg) {
			if name == "" {
				c.Unique = true
				continue
			}
			name = prefix + name
			if _, seen := groups[name]; !seen {
				groupOrder = append(groupOrder, name)
			}
			groups[name] = append(groups[name], col)
		}
		t.Columns = append(t.Columns, c)
		b.cols[col] = true
		if isPK {
			t.PrimaryKey = append(t.PrimaryKey, col)
		}
	}
	for _, name := range groupOrder {
		switch cols := groups[name]; len(cols) {
		case 0:
		case 1:
			for i := range t.Columns {
				if t.Columns[i].Name == cols[0] {
					t.Columns[i].Unique = true
				}
			}
		default:
			t.Uniques = append(t.Uniques, staticmodel.Unique{Columns: cols})
		}
	}
	return b
}

// uniqueGroups lists the groups of a `unique` option: "" for a plain unique
// column, names for shared unique constraints. Like bun, a single value may
// hold several comma-separated names.
func uniqueGroups(t tag) []string {
	vs, ok := t.Options["unique"]
	if !ok {
		return nil
	}
	if len(vs) == 1 {
		return strings.Split(vs[0], ",")
	}
	return vs
}

var charType = regexp.MustCompile(`(?i)^\s*(?:n?var)?char(?:acter)?(?:\s+varying)?(?:2)?\s*\(\s*(\d+)\s*\)`)

// charLength reads the length of a character column type: varchar(255),
// char(2), character varying(40).
func charLength(sqlType string) int {
	m := charType.FindStringSubmatch(sqlType)
	if m == nil {
		return 0
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return 0
	}
	return n
}

// addRelations records the foreign keys bun's rel tags declare: belongs-to
// puts the key on the declaring table, has-one and has-many on the associated
// one. m2m and polymorphic relations are not covered.
func addRelations(all []*built) error {
	byType := map[string]*built{}
	for _, b := range all {
		byType[b.strukt.PkgPath+"."+b.strukt.Name] = b
	}
	for _, owner := range all {
		for _, f := range owner.relFields {
			tg := parseTag(f.Tag.Get("bun"))
			kind, _ := tg.option("rel")
			if kind == "" || tg.has("polymorphic") {
				continue
			}
			elem := staticmodel.Deref(f.Type)
			if sl, ok := elem.Underlying().(*types.Slice); ok {
				elem = staticmodel.Deref(sl.Elem())
			}
			target := byType[staticmodel.NamedType(elem)]
			if target == nil {
				continue
			}
			fail := func(format string, args ...any) error {
				return fmt.Errorf("%s:%d: %s.%s %s: %s", f.Pos.File, f.Pos.Line, owner.strukt.Name, f.Name, kind, fmt.Sprintf(format, args...))
			}
			switch kind {
			case "belongs-to":
				cols, err := belongsToKey(owner, target, f, tg)
				if err != nil {
					return fail("%v", err)
				}
				if len(cols) > 0 {
					owner.table.Relations = append(owner.table.Relations, staticmodel.Relation{Columns: cols, References: target.table.Ref()})
				}
			case "has-one", "has-many":
				cols, err := hasKey(owner, target, tg)
				if err != nil {
					return fail("%v", err)
				}
				if len(cols) > 0 {
					target.table.Relations = append(target.table.Relations, staticmodel.Relation{Columns: cols, References: owner.table.Ref()})
				}
			}
		}
	}
	return nil
}

// joinPairs reads the "join:base=joined" option(s) into (base, joined) pairs.
func joinPairs(t tag) ([][2]string, bool, error) {
	vs, ok := t.Options["join"]
	if !ok {
		return nil, false, nil
	}
	if len(vs) == 1 {
		vs = strings.Split(vs[0], ",")
	}
	var out [][2]string
	for _, v := range vs {
		base, joined, ok := strings.Cut(strings.TrimSpace(v), "=")
		if !ok || strings.Contains(joined, "=") {
			return nil, true, fmt.Errorf("cannot parse join:%s", strings.Join(vs, ","))
		}
		out = append(out, [2]string{base, joined})
	}
	return out, true, nil
}

// belongsToKey returns the columns of owner that reference target.
func belongsToKey(owner, target *built, f staticmodel.Field, t tag) ([]string, error) {
	pairs, joined, err := joinPairs(t)
	if err != nil {
		return nil, err
	}
	var cols []string
	if joined {
		for _, p := range pairs {
			if !owner.cols[p[0]] {
				return nil, fmt.Errorf("%s must have column %s", owner.strukt.Name, p[0])
			}
			cols = append(cols, p[0])
		}
		return cols, nil
	}
	prefix := underscore(f.Name) + "_"
	for _, pk := range target.table.PrimaryKey {
		switch {
		case owner.cols[prefix+pk]:
			cols = append(cols, prefix+pk)
		case owner.cols[pk]:
			cols = append(cols, pk)
		default:
			return nil, fmt.Errorf("%s must have column %s (to override, use join:base_column=join_column on the field)", owner.strukt.Name, prefix+pk)
		}
	}
	return cols, nil
}

// hasKey returns the columns of target that reference owner.
func hasKey(owner, target *built, t tag) ([]string, error) {
	pairs, joined, err := joinPairs(t)
	if err != nil {
		return nil, err
	}
	var cols []string
	if joined {
		for _, p := range pairs {
			if !target.cols[p[1]] {
				return nil, fmt.Errorf("%s must have column %s", target.strukt.Name, p[1])
			}
			cols = append(cols, p[1])
		}
		return cols, nil
	}
	prefix := underscore(owner.strukt.Name) + "_"
	for _, pk := range owner.table.PrimaryKey {
		switch {
		case target.cols[prefix+pk]:
			cols = append(cols, prefix+pk)
		case target.cols[pk]:
			cols = append(cols, pk)
		default:
			return nil, fmt.Errorf("%s must have column %s (to override, use join:base_column=join_column on the field)", target.strukt.Name, prefix+pk)
		}
	}
	return cols, nil
}
