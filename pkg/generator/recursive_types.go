// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
)

// containment is a by-value use of a locally declared type inside another type.
// ident is the field's type identifier when the edge can be broken by turning
// that field into a pointer, and nil for embedded fields, defined types and
// optionalPointer fields, which are pointers already.
type containment struct {
	to    string
	ident *ast.Ident
}

// optionalPointer returns the type of a field holding an optional element of
// the complex type goType, as the types template writes it: *(goType). The
// parentheses mark the pointer for breakRecursiveStructs, which counts the
// field as a by-value use, and are removed by unwrapOptionalPointers.
// Counting it so keeps cycles broken at the fields chosen when optional
// elements of complex type were struct values, so that required fields keep
// their types.
func optionalPointer(goType string) string {
	return "*(" + goType + ")"
}

// breakRecursiveStructs makes generated structs that contain themselves by
// value, directly or through other structs, valid Go by turning the named
// field that closes each cycle into a pointer. XSD allows a complex type to
// hold a required element of its own type, which the templates emit as a
// value field. An optionalPointer field counts as a by-value use that is
// already broken. The source is returned unchanged if it cannot be parsed or
// has no such cycle.
func breakRecursiveStructs(src []byte) []byte {
	const prefix = "package p\n"
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", prefix+string(src), parser.SkipObjectResolution)
	if err != nil {
		return src
	}

	edges := make(map[string][]containment)
	for _, d := range file.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, spec := range gd.Specs {
			if ts, ok := spec.(*ast.TypeSpec); ok {
				edges[ts.Name.Name] = nil
			}
		}
	}
	for _, d := range file.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			switch t := ts.Type.(type) {
			case *ast.StructType:
				edges[ts.Name.Name] = structContainments(t, edges)
			case *ast.Ident:
				if _, local := edges[t.Name]; local {
					edges[ts.Name.Name] = []containment{{to: t.Name}}
				}
			}
		}
	}

	names := make([]string, 0, len(edges))
	for name := range edges {
		names = append(names, name)
	}
	sort.Strings(names)

	const (
		unvisited = iota
		inProgress
		done
	)
	state := make(map[string]int, len(edges))
	var pointerize []*ast.Ident
	var visit func(string)
	visit = func(name string) {
		state[name] = inProgress
		for _, e := range edges[name] {
			switch state[e.to] {
			case inProgress:
				if e.ident != nil {
					pointerize = append(pointerize, e.ident)
				}
			case unvisited:
				visit(e.to)
			}
		}
		state[name] = done
	}
	for _, name := range names {
		if state[name] == unvisited {
			visit(name)
		}
	}

	if len(pointerize) == 0 {
		return src
	}

	base := fset.File(file.Pos()).Base()
	offsets := make([]int, 0, len(pointerize))
	for _, ident := range pointerize {
		offsets = append(offsets, int(ident.Pos())-base-len(prefix))
	}
	sort.Sort(sort.Reverse(sort.IntSlice(offsets)))
	out := append([]byte(nil), src...)
	for _, off := range offsets {
		out = append(out[:off], append([]byte("*"), out[off:]...)...)
	}
	return out
}

// structContainments returns the locally declared types that st holds by
// value, including through anonymous struct fields.
func structContainments(st *ast.StructType, decls map[string][]containment) []containment {
	var result []containment
	if st.Fields == nil {
		return nil
	}
	for _, field := range st.Fields.List {
		switch t := field.Type.(type) {
		case *ast.Ident:
			if _, local := decls[t.Name]; !local {
				continue
			}
			c := containment{to: t.Name}
			if len(field.Names) > 0 {
				c.ident = t
			}
			result = append(result, c)
		case *ast.StarExpr:
			if id := optionalPointerIdent(t); id != nil {
				if _, local := decls[id.Name]; local {
					result = append(result, containment{to: id.Name})
				}
			}
		case *ast.StructType:
			result = append(result, structContainments(t, decls)...)
		}
	}
	return result
}

// optionalPointerIdent returns the type identifier of an optionalPointer
// field type, or nil when star is not one.
func optionalPointerIdent(star *ast.StarExpr) *ast.Ident {
	paren, ok := star.X.(*ast.ParenExpr)
	if !ok {
		return nil
	}
	id, _ := paren.X.(*ast.Ident)
	return id
}

// unwrapOptionalPointers removes the parentheses of the optionalPointer field
// types in src. The source is returned unchanged if it cannot be parsed.
func unwrapOptionalPointers(src []byte) []byte {
	const prefix = "package p\n"
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", prefix+string(src), parser.SkipObjectResolution)
	if err != nil {
		return src
	}

	var offsets []int
	base := fset.File(file.Pos()).Base()
	ast.Inspect(file, func(n ast.Node) bool {
		field, ok := n.(*ast.Field)
		if !ok {
			return true
		}
		if star, ok := field.Type.(*ast.StarExpr); ok && optionalPointerIdent(star) != nil {
			paren := star.X.(*ast.ParenExpr)
			offsets = append(offsets, int(paren.Lparen)-base-len(prefix), int(paren.Rparen)-base-len(prefix))
		}
		return true
	})
	if len(offsets) == 0 {
		return src
	}

	sort.Sort(sort.Reverse(sort.IntSlice(offsets)))
	out := append([]byte(nil), src...)
	for _, off := range offsets {
		out = append(out[:off], out[off+1:]...)
	}
	return out
}
