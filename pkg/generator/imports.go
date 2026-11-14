// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
)

// referencedPackages returns the names used as the left-hand side of
// qualified identifiers (such as soap in soap.XSDDateTime) in the given
// generated code fragments. Fragments are declarations without a package
// clause. If they cannot be parsed, context and soap are reported as used so
// the header keeps importing them.
func referencedPackages(code ...[]byte) map[string]bool {
	src := bytes.NewBufferString("package p\n")
	for _, c := range code {
		src.Write(c)
		src.WriteByte('\n')
	}

	file, err := parser.ParseFile(token.NewFileSet(), "", src.Bytes(), parser.SkipObjectResolution)
	if err != nil {
		return map[string]bool{"context": true, "soap": true}
	}

	used := map[string]bool{}
	ast.Inspect(file, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			if id, ok := sel.X.(*ast.Ident); ok {
				used[id.Name] = true
			}
		}
		return true
	})
	return used
}
