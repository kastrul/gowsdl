// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"sort"
	"strings"
)

// externalStructTypes lists struct types from other packages that generated
// fields may use by value.
var externalStructTypes = map[string]bool{
	"soap.XSDDateTime": true,
	"soap.XSDDate":     true,
	"soap.XSDTime":     true,
	"soap.AnyType":     true,
	"xml.Name":         true,
}

var jsonTagPattern = regexp.MustCompile("(^|[\\s`])json:\"([^\"]*)\"")

// fixStructOmitempty rewrites the json "omitempty" option to "omitzero" on
// fields whose type is a struct value. encoding/json ignores omitempty for
// structs, while omitzero (Go 1.24+) omits their zero value. The source is
// returned unchanged if it cannot be parsed.
func fixStructOmitempty(src []byte) []byte {
	const prefix = "package p\n"
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", prefix+string(src), parser.SkipObjectResolution)
	if err != nil {
		return src
	}

	structTypes := collectStructTypes(file)

	type edit struct {
		start, end int
		text       string
	}
	var edits []edit
	base := fset.File(file.Pos()).Base()

	ast.Inspect(file, func(n ast.Node) bool {
		st, ok := n.(*ast.StructType)
		if !ok || st.Fields == nil {
			return true
		}
		for _, field := range st.Fields.List {
			if field.Tag == nil || !isStructExpr(field.Type, structTypes) {
				continue
			}
			newTag, changed := rewriteJSONOmitempty(field.Tag.Value)
			if !changed {
				continue
			}
			start := int(field.Tag.Pos()) - base - len(prefix)
			edits = append(edits, edit{start: start, end: start + len(field.Tag.Value), text: newTag})
		}
		return true
	})

	if len(edits) == 0 {
		return src
	}

	sort.Slice(edits, func(i, j int) bool { return edits[i].start > edits[j].start })
	out := append([]byte(nil), src...)
	for _, e := range edits {
		out = append(out[:e.start], append([]byte(e.text), out[e.end:]...)...)
	}
	return out
}

// collectStructTypes returns the names of types declared in file whose
// underlying type is a struct, following defined types and aliases.
func collectStructTypes(file *ast.File) map[string]bool {
	decls := make(map[string]ast.Expr)
	for _, d := range file.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, spec := range gd.Specs {
			if ts, ok := spec.(*ast.TypeSpec); ok {
				decls[ts.Name.Name] = ts.Type
			}
		}
	}

	result := make(map[string]bool)
	for name := range decls {
		seen := make(map[string]bool)
		expr := decls[name]
		for {
			if _, ok := expr.(*ast.StructType); ok {
				result[name] = true
				break
			}
			if sel, ok := expr.(*ast.SelectorExpr); ok {
				result[name] = isExternalStruct(sel)
				break
			}
			ident, ok := expr.(*ast.Ident)
			if !ok || seen[ident.Name] {
				break
			}
			seen[ident.Name] = true
			next, ok := decls[ident.Name]
			if !ok {
				break
			}
			expr = next
		}
	}
	return result
}

func isStructExpr(expr ast.Expr, structTypes map[string]bool) bool {
	switch t := expr.(type) {
	case *ast.StructType:
		return true
	case *ast.Ident:
		return structTypes[t.Name]
	case *ast.SelectorExpr:
		return isExternalStruct(t)
	}
	return false
}

func isExternalStruct(sel *ast.SelectorExpr) bool {
	pkg, ok := sel.X.(*ast.Ident)
	return ok && externalStructTypes[pkg.Name+"."+sel.Sel.Name]
}

// rewriteJSONOmitempty replaces the omitempty option in the json key of a
// raw struct tag literal with omitzero.
func rewriteJSONOmitempty(tag string) (string, bool) {
	changed := false
	result := jsonTagPattern.ReplaceAllStringFunc(tag, func(m string) string {
		sub := jsonTagPattern.FindStringSubmatch(m)
		parts := strings.Split(sub[2], ",")
		hasZero := false
		for _, p := range parts[1:] {
			if p == "omitzero" {
				hasZero = true
			}
		}
		opts := parts[:1]
		for _, p := range parts[1:] {
			if p == "omitempty" {
				changed = true
				if hasZero {
					continue
				}
				p = "omitzero"
			}
			opts = append(opts, p)
		}
		return sub[1] + `json:"` + strings.Join(opts, ",") + `"`
	})
	return result, changed
}
