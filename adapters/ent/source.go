package entadapter

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"entgo.io/ent/entc/load"
)

// sourceIndex finds where fields and edges are written in the schema
// package's Go source. ent does not record this, so the package is parsed:
// a field is an element of the slice returned by Fields(), a mixed-in field
// an element of the mixin's Fields(), an edge an element of Edges().
type sourceIndex struct {
	cwd     string
	fset    *token.FileSet
	methods map[string]map[string]*ast.FuncDecl // type -> method -> declaration
}

func newSourceIndex(schemaPath string) (*sourceIndex, error) {
	dir, err := filepath.Abs(schemaPath)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("cannot read ent schema package %s: %w", schemaPath, err)
	}
	cwd, _ := os.Getwd()
	s := &sourceIndex{cwd: cwd, fset: token.NewFileSet(), methods: map[string]map[string]*ast.FuncDecl{}}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(s.fset, filepath.Join(dir, name), nil, parser.SkipObjectResolution)
		if err != nil {
			return nil, fmt.Errorf("cannot parse %s: %w", filepath.Join(schemaPath, name), err)
		}
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Recv == nil || len(fd.Recv.List) != 1 {
				continue
			}
			recv := typeName(fd.Recv.List[0].Type)
			if recv == "" {
				continue
			}
			if s.methods[recv] == nil {
				s.methods[recv] = map[string]*ast.FuncDecl{}
			}
			s.methods[recv][fd.Name.Name] = fd
		}
	}
	return s, nil
}

// fieldPos locates the ent field `name` of schema type typeName. p says where
// ent found it: in the type's own Fields() or in one of its mixins.
func (s *sourceIndex) fieldPos(typeName, name string, p *load.Position) *position {
	if p == nil {
		return nil
	}
	if !p.MixedIn {
		return s.pick(s.elems(typeName, "Fields"), p.Index, name)
	}
	mixins := s.elems(typeName, "Mixin")
	if p.MixinIndex < 0 || p.MixinIndex >= len(mixins) {
		return nil
	}
	m := mixins[p.MixinIndex]
	if mt := exprTypeName(m); mt != "" {
		if elems := s.elems(mt, "Fields"); elems != nil {
			if pos := s.pick(elems, p.Index, name); pos != nil {
				return pos
			}
		}
	}
	// The mixin lives outside the package (for example ent's own mixin.Time):
	// the best place left is where the schema mixes it in.
	return s.at(m.Pos())
}

// edgePos locates the edge `name` of schema type typeName, in its Edges() or
// in the Edges() of its mixins.
func (s *sourceIndex) edgePos(typeName, name string) *position {
	if pos := s.find(s.elems(typeName, "Edges"), name); pos != nil {
		return pos
	}
	for _, m := range s.elems(typeName, "Mixin") {
		if mt := exprTypeName(m); mt != "" {
			if pos := s.find(s.elems(mt, "Edges"), name); pos != nil {
				return pos
			}
		}
	}
	return nil
}

// elems returns the elements of the slice literal a method returns.
func (s *sourceIndex) elems(typ, method string) []ast.Expr {
	fd := s.methods[typ][method]
	if fd == nil || fd.Body == nil {
		return nil
	}
	var out []ast.Expr
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		if out != nil {
			return false
		}
		if r, ok := n.(*ast.ReturnStmt); ok && len(r.Results) == 1 {
			if cl, ok := r.Results[0].(*ast.CompositeLit); ok {
				out = cl.Elts
				return false
			}
		}
		return true
	})
	return out
}

// pick finds the element that declares name, preferring a match by name and
// falling back to the index ent reported.
func (s *sourceIndex) pick(elems []ast.Expr, index int, name string) *position {
	if pos := s.find(elems, name); pos != nil {
		return pos
	}
	if index >= 0 && index < len(elems) {
		return s.at(elems[index].Pos())
	}
	return nil
}

func (s *sourceIndex) find(elems []ast.Expr, name string) *position {
	for _, e := range elems {
		if rootArg(e) == name {
			return s.at(e.Pos())
		}
	}
	return nil
}

func (s *sourceIndex) at(p token.Pos) *position {
	tp := s.fset.Position(p)
	if !tp.IsValid() {
		return nil
	}
	return &position{File: s.rel(tp.Filename), Line: tp.Line}
}

// rootArg returns the first string argument of the call a builder chain
// starts with: for field.String("name").NotEmpty() it is "name".
func rootArg(e ast.Expr) string {
	for {
		call, ok := e.(*ast.CallExpr)
		if !ok {
			return ""
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if ok {
			if inner, ok := sel.X.(*ast.CallExpr); ok {
				e = inner
				continue
			}
		}
		if len(call.Args) == 0 {
			return ""
		}
		lit, ok := call.Args[0].(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return ""
		}
		v, err := strconv.Unquote(lit.Value)
		if err != nil {
			return ""
		}
		return v
	}
}

// exprTypeName names the type of a composite literal element such as
// Timestamps{} or &Timestamps{}; selector types (other packages) give "".
func exprTypeName(e ast.Expr) string {
	if u, ok := e.(*ast.UnaryExpr); ok {
		e = u.X
	}
	if cl, ok := e.(*ast.CompositeLit); ok {
		if id, ok := cl.Type.(*ast.Ident); ok {
			return id.Name
		}
	}
	return ""
}

func typeName(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.StarExpr:
		return typeName(t.X)
	case *ast.Ident:
		return t.Name
	}
	return ""
}
