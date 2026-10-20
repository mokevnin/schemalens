// Package staticmodel is the shared static-analysis layer of the schemalens
// adapters (gorm, bun, ...). It loads a project's Go packages with
// golang.org/x/tools/go/packages and exposes the named struct types in them
// with their fields, struct tags, types and source positions. It never runs
// project code and knows nothing about any ORM: adapters decide which structs
// are models and what the tags mean.
//
// It is a module of its own, outside internal/, so every adapter module can
// import it.
package staticmodel

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
)

// Position is a place in the project's source; File is relative to the
// directory the program was loaded from, with forward slashes.
type Position struct {
	File string
	Line int
}

// Field is one struct field.
type Field struct {
	Name     string
	Type     types.Type
	Tag      reflect.StructTag
	Embedded bool // anonymous (embedded) field
	Exported bool
	// Pos is where the field is declared. For a field declared outside the
	// project (an embedded gorm.Model, say) it is the position of the field
	// of the project's struct that embeds it, so findings always point at
	// code the user owns.
	Pos Position
	// Path lists the embedding fields leading to this field, outermost
	// first; empty for a direct field. Set by Flatten.
	Path []Field

	inProject bool
}

// Struct is a named struct type declared in a loaded package.
type Struct struct {
	Name    string
	PkgPath string
	Pos     Position
	Fields  []Field

	named   *types.Named
	pkg     *packages.Package
	program *Program
}

// Program is a loaded project.
type Program struct {
	dir     string
	fset    *token.FileSet
	structs []*Struct
}

// Load type-checks the packages matching patterns (default ./...) in dir.
// It returns an error naming every load or type error: a model that does not
// compile cannot be analysed reliably.
func Load(dir string, patterns ...string) (*Program, error) {
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	cfg := &packages.Config{
		Dir: abs,
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedImports |
			packages.NeedDeps | packages.NeedTypes | packages.NeedTypesInfo,
	}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, fmt.Errorf("cannot load %s: %w", strings.Join(patterns, " "), err)
	}
	var problems []string
	for _, p := range pkgs {
		for _, e := range p.Errors {
			problems = append(problems, e.Error())
		}
	}
	if len(problems) > 0 {
		return nil, fmt.Errorf("cannot load %s:\n  %s", strings.Join(patterns, " "), strings.Join(problems, "\n  "))
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("cannot load %s: no packages matched", strings.Join(patterns, " "))
	}
	p := &Program{dir: abs}
	for _, pkg := range pkgs {
		p.fset = pkg.Fset
		p.collect(pkg)
	}
	slices.SortFunc(p.structs, func(a, b *Struct) int {
		if c := strings.Compare(a.Pos.File, b.Pos.File); c != 0 {
			return c
		}
		return a.Pos.Line - b.Pos.Line
	})
	return p, nil
}

// Structs returns the named struct types declared in the loaded packages, in
// source order.
func (p *Program) Structs() []*Struct { return p.structs }

func (p *Program) collect(pkg *packages.Package) {
	scope := pkg.Types.Scope()
	for _, name := range scope.Names() {
		tn, ok := scope.Lookup(name).(*types.TypeName)
		if !ok || tn.IsAlias() {
			continue
		}
		named, ok := tn.Type().(*types.Named)
		if !ok {
			continue
		}
		st, ok := named.Underlying().(*types.Struct)
		if !ok {
			continue
		}
		pos, inProject := p.position(tn.Pos())
		if !inProject {
			continue
		}
		p.structs = append(p.structs, &Struct{
			Name: name, PkgPath: pkg.PkgPath, Pos: pos, Fields: p.fieldsOf(st), named: named, pkg: pkg, program: p,
		})
	}
}

// position converts a token position to a project-relative one; the bool is
// false for positions outside the project directory.
func (p *Program) position(pos token.Pos) (Position, bool) {
	tp := p.fset.Position(pos)
	rel, err := filepath.Rel(p.dir, tp.Filename)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return Position{File: tp.Filename, Line: tp.Line}, false
	}
	return Position{File: filepath.ToSlash(rel), Line: tp.Line}, true
}

func (p *Program) fieldsOf(st *types.Struct) []Field {
	out := make([]Field, 0, st.NumFields())
	for i := range st.NumFields() {
		f := st.Field(i)
		pos, in := p.position(f.Pos())
		out = append(out, Field{
			Name:      f.Name(),
			Type:      f.Type(),
			Tag:       reflect.StructTag(st.Tag(i)),
			Embedded:  f.Anonymous(),
			Exported:  f.Exported(),
			Pos:       pos,
			inProject: in,
		})
	}
	return out
}

const maxDepth = 8

// Flatten returns the struct's fields with embedded struct fields replaced by
// their own fields, recursively. expand decides, per embedding field, whether
// to expand it (an ORM decides what counts as embedding); a field it declines
// stays as a field. Expanded fields carry the embedding chain in Path, and
// those declared outside the project take the position of the outermost
// embedding field.
func (s *Struct) Flatten(expand func(Field) bool) []Field {
	return s.program.flatten(s.Fields, nil, expand, 0)
}

func (p *Program) flatten(fields, path []Field, expand func(Field) bool, depth int) []Field {
	var out []Field
	for _, f := range fields {
		if st, ok := structOf(f.Type); ok && depth < maxDepth && expand(f) {
			chain := append(slices.Clone(path), f)
			children := p.fieldsOf(st)
			for i := range children {
				if !children[i].inProject {
					children[i].Pos = chain[0].Pos
				}
			}
			out = append(out, p.flatten(children, chain, expand, depth+1)...)
			continue
		}
		f.Path = slices.Clone(path)
		out = append(out, f)
	}
	return out
}

// structOf returns the struct underlying t, looking through one pointer.
func structOf(t types.Type) (*types.Struct, bool) {
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	st, ok := t.Underlying().(*types.Struct)
	return st, ok
}

// Deref strips one level of pointer from t.
func Deref(t types.Type) types.Type {
	if ptr, ok := t.(*types.Pointer); ok {
		return ptr.Elem()
	}
	return t
}

// NamedType returns "importpath.Name" for t (looking through a pointer), or
// "" when t is not a named type. Adapters compare it with names such as
// "gorm.io/gorm.Model" or "time.Time".
func NamedType(t types.Type) string {
	n, ok := types.Unalias(Deref(t)).(*types.Named)
	if !ok || n.Obj().Pkg() == nil {
		return ""
	}
	return n.Obj().Pkg().Path() + "." + n.Obj().Name()
}

// HasMethod reports whether t, or a pointer to it, has the named exported
// method.
func HasMethod(t types.Type, name string) bool {
	if _, ok := t.(*types.Pointer); !ok {
		t = types.NewPointer(t)
	}
	return types.NewMethodSet(t).Lookup(nil, name) != nil
}

// HasMethod reports whether the struct, or a pointer to it, has the method.
func (s *Struct) HasMethod(name string) bool { return HasMethod(s.named, name) }

// StringMethod reads a method of the form
//
//	func (T) Name() string { return "literal" }
//
// and returns the literal. ok is false when the struct has no such method or
// its body is anything else (the value is only known at run time).
func (s *Struct) StringMethod(name string) (value string, ok bool) {
	for _, file := range s.pkg.Syntax {
		for _, d := range file.Decls {
			fd, isFn := d.(*ast.FuncDecl)
			if !isFn || fd.Recv == nil || fd.Name.Name != name || fd.Body == nil || len(fd.Body.List) != 1 {
				continue
			}
			if recvName(fd.Recv.List[0].Type) != s.Name {
				continue
			}
			ret, isRet := fd.Body.List[0].(*ast.ReturnStmt)
			if !isRet || len(ret.Results) != 1 {
				return "", false
			}
			lit, isLit := ret.Results[0].(*ast.BasicLit)
			if !isLit || lit.Kind != token.STRING {
				return "", false
			}
			v, err := strconv.Unquote(lit.Value)
			return v, err == nil
		}
	}
	return "", false
}

func recvName(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.StarExpr:
		return recvName(t.X)
	case *ast.Ident:
		return t.Name
	}
	return ""
}
