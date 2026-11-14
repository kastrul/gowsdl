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
// that field into a pointer, and nil for embedded fields and defined types.
type containment struct {
	to    string
	ident *ast.Ident
}

// breakRecursiveStructs makes generated structs that contain themselves by
// value, directly or through other structs, valid Go by turning the named
// field that closes each cycle into a pointer. XSD allows a complex type to
// hold an optional element of its own type, which the templates emit as a
// value field. The source is returned unchanged if it cannot be parsed or has
// no such cycle.
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
		case *ast.StructType:
			result = append(result, structContainments(t, decls)...)
		}
	}
	return result
}
