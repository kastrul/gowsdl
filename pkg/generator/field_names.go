// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const dedupePackageClause = "package p\n"

type structField struct {
	ident    *ast.Ident
	embedded bool
	derived  bool
}

type identEdit struct {
	pos    token.Pos
	offset int
	old    string
	new    string
}

// dedupeStructFields renames struct fields whose names collide within the same
// struct, which happens when XML names such as "this" and "_this" map to the
// same exported Go identifier. Fields whose names were derived from XML names
// that do not start with a letter are renamed first. The source is returned
// unchanged when it cannot be parsed or contains no collisions.
func dedupeStructFields(src []byte) []byte {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", dedupePackageClause+string(src), parser.SkipObjectResolution)
	if err != nil {
		return src
	}

	var edits []identEdit
	ast.Inspect(file, func(n ast.Node) bool {
		if st, ok := n.(*ast.StructType); ok {
			edits = append(edits, structFieldRenames(st)...)
		}
		return true
	})
	if len(edits) == 0 {
		return src
	}

	base := len(dedupePackageClause)
	for i := range edits {
		edits[i].offset = fset.Position(edits[i].pos).Offset - base
	}
	sort.Slice(edits, func(i, j int) bool { return edits[i].offset < edits[j].offset })

	var out strings.Builder
	out.Grow(len(src) + len(edits)*2)
	last := 0
	for _, e := range edits {
		out.Write(src[last:e.offset])
		out.WriteString(e.new)
		last = e.offset + len(e.old)
	}
	out.Write(src[last:])
	return []byte(out.String())
}

func structFieldRenames(st *ast.StructType) []identEdit {
	if st.Fields == nil {
		return nil
	}

	groups := make(map[string][]structField)
	var order []string
	for _, f := range st.Fields.List {
		derived := isDerivedFieldTag(f.Tag)
		var entries []structField
		if len(f.Names) == 0 {
			if id := embeddedFieldIdent(f.Type); id != nil {
				entries = append(entries, structField{ident: id, embedded: true})
			}
		}
		for _, id := range f.Names {
			if id.Name != "_" {
				entries = append(entries, structField{ident: id, derived: derived})
			}
		}
		for _, e := range entries {
			if _, seen := groups[e.ident.Name]; !seen {
				order = append(order, e.ident.Name)
			}
			groups[e.ident.Name] = append(groups[e.ident.Name], e)
		}
	}

	var edits []identEdit
	for _, name := range order {
		fields := groups[name]
		if len(fields) < 2 {
			continue
		}
		keep := keeperIndex(fields)
		for i, f := range fields {
			if i == keep || f.embedded {
				continue
			}
			newName := uniqueFieldName(name, groups)
			groups[newName] = []structField{f}
			edits = append(edits, identEdit{pos: f.ident.Pos(), old: f.ident.Name, new: newName})
		}
	}
	return edits
}

func keeperIndex(fields []structField) int {
	for i, f := range fields {
		if f.embedded {
			return i
		}
	}
	for i, f := range fields {
		if !f.derived {
			return i
		}
	}
	return 0
}

func uniqueFieldName(name string, taken map[string][]structField) string {
	candidate := name + "_"
	for n := 2; ; n++ {
		if _, exists := taken[candidate]; !exists {
			return candidate
		}
		candidate = name + "_" + strconv.Itoa(n)
	}
}

func embeddedFieldIdent(expr ast.Expr) *ast.Ident {
	switch t := expr.(type) {
	case *ast.Ident:
		return t
	case *ast.StarExpr:
		return embeddedFieldIdent(t.X)
	case *ast.SelectorExpr:
		return t.Sel
	case *ast.IndexExpr:
		return embeddedFieldIdent(t.X)
	case *ast.IndexListExpr:
		return embeddedFieldIdent(t.X)
	}
	return nil
}

// isDerivedFieldTag reports whether the field's xml tag names an element or
// attribute whose name does not start with a letter, meaning the Go field name
// had to be derived rather than taken directly from the XML name.
func isDerivedFieldTag(tag *ast.BasicLit) bool {
	if tag == nil {
		return false
	}
	raw, err := strconv.Unquote(tag.Value)
	if err != nil {
		return false
	}
	xmlName, _, _ := strings.Cut(reflect.StructTag(raw).Get("xml"), ",")
	if i := strings.LastIndex(xmlName, " "); i >= 0 {
		xmlName = xmlName[i+1:]
	}
	if xmlName == "" {
		return false
	}
	r, _ := utf8.DecodeRuneInString(xmlName)
	return !unicode.IsLetter(r)
}
