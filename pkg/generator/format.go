// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"sort"
	"strings"
)

// FormatSource formats generated Go source like gofmt and additionally
// removes the template artifacts gofmt keeps: blank lines between struct
// fields, interface methods, grouped const/var/type specs and composite
// literal elements, and comment lines that contain only "//".
func FormatSource(src []byte) ([]byte, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse generated code: %w", err)
	}

	tokFile := fset.File(file.Pos())
	dropped := map[int]bool{}
	for _, c := range removeEmptyComments(file) {
		line := tokFile.Line(c.Pos())
		if line > 1 && lineText(tokFile, src, line) == c.Text {
			dropped[line] = true
		}
	}
	collapseBlankLines(tokFile, file, src, dropped)

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return nil, fmt.Errorf("failed to print generated code: %w", err)
	}

	out, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to format generated code: %w", err)
	}
	return out, nil
}

func isEmptyComment(c *ast.Comment) bool {
	return strings.TrimSpace(strings.TrimPrefix(c.Text, "//")) == "" && strings.HasPrefix(c.Text, "//")
}

func trimEmptyComments(cg *ast.CommentGroup) []*ast.Comment {
	var removed []*ast.Comment
	list := cg.List
	for len(list) > 0 && isEmptyComment(list[0]) {
		removed = append(removed, list[0])
		list = list[1:]
	}
	for len(list) > 0 && isEmptyComment(list[len(list)-1]) {
		removed = append(removed, list[len(list)-1])
		list = list[:len(list)-1]
	}
	cg.List = list
	return removed
}

// removeEmptyComments drops leading and trailing "//" lines from every
// comment group, removes groups that end up empty and returns the
// comments it dropped.
func removeEmptyComments(file *ast.File) []*ast.Comment {
	var removed []*ast.Comment
	kept := file.Comments[:0]
	for _, cg := range file.Comments {
		removed = append(removed, trimEmptyComments(cg)...)
		if len(cg.List) > 0 {
			kept = append(kept, cg)
		}
	}
	file.Comments = kept

	dropEmpty := func(cg **ast.CommentGroup) {
		if *cg != nil && len((*cg).List) == 0 {
			*cg = nil
		}
	}
	ast.Inspect(file, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.Field:
			dropEmpty(&n.Doc)
			dropEmpty(&n.Comment)
		case *ast.GenDecl:
			dropEmpty(&n.Doc)
		case *ast.FuncDecl:
			dropEmpty(&n.Doc)
		case *ast.ValueSpec:
			dropEmpty(&n.Doc)
			dropEmpty(&n.Comment)
		case *ast.TypeSpec:
			dropEmpty(&n.Doc)
			dropEmpty(&n.Comment)
		case *ast.ImportSpec:
			dropEmpty(&n.Doc)
			dropEmpty(&n.Comment)
		}
		return true
	})
	dropEmpty(&file.Doc)
	return removed
}

// collapseBlankLines merges blank lines found in the gaps between the
// elements of lists into the preceding line so the printer emits the
// elements on consecutive lines. Only whitespace-only lines inside gaps
// are touched, so element contents (func literals, raw strings) and kept
// comments are left alone. Lines that held only a dropped comment are
// always merged.
func collapseBlankLines(tokFile *token.File, file *ast.File, src []byte, dropped map[int]bool) {
	remove := map[int]bool{}
	for line := range dropped {
		remove[line] = true
	}

	gap := func(from, to token.Pos) {
		if !from.IsValid() || !to.IsValid() {
			return
		}
		start := tokFile.Line(from) + 1
		end := tokFile.Line(to) - 1
		for line := start; line <= end; line++ {
			if lineText(tokFile, src, line) == "" {
				remove[line] = true
			}
		}
	}

	list := func(open, close token.Pos, nodes []ast.Node) {
		prev := open
		for _, n := range nodes {
			gap(prev, n.Pos())
			prev = n.End()
		}
		gap(prev, close)
	}

	ast.Inspect(file, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.StructType:
			list(n.Fields.Opening, n.Fields.Closing, fieldNodes(n.Fields, file.Comments))
		case *ast.InterfaceType:
			list(n.Methods.Opening, n.Methods.Closing, fieldNodes(n.Methods, file.Comments))
		case *ast.GenDecl:
			if n.Tok == token.IMPORT || !n.Lparen.IsValid() {
				return true
			}
			nodes := make([]ast.Node, 0, len(n.Specs))
			for _, s := range n.Specs {
				nodes = append(nodes, withComments(s, specDoc(s), specComment(s)))
			}
			list(n.Lparen, n.Rparen, nodes)
		case *ast.CompositeLit:
			nodes := make([]ast.Node, 0, len(n.Elts))
			for _, e := range n.Elts {
				nodes = append(nodes, e)
			}
			list(n.Lbrace, n.Rbrace, nodes)
		}
		return true
	})

	if len(remove) == 0 {
		return
	}

	lines := tokFile.Lines()
	newLines := make([]int, 0, len(lines))
	for i, off := range lines {
		if !remove[i+1] {
			newLines = append(newLines, off)
		}
	}
	tokFile.SetLines(newLines)
}

// lineText returns the source of the given line without surrounding
// whitespace, or "\x00" for lines outside the file.
func lineText(tokFile *token.File, src []byte, line int) string {
	if line < 1 || line > tokFile.LineCount() {
		return "\x00"
	}
	start := tokFile.Offset(tokFile.LineStart(line))
	end := len(src)
	if line < tokFile.LineCount() {
		end = tokFile.Offset(tokFile.LineStart(line + 1))
	}
	return string(bytes.TrimSpace(src[start:end]))
}

// span is an ast.Node covering a node together with its doc and line comments.
type span struct{ pos, end token.Pos }

func (s span) Pos() token.Pos { return s.pos }
func (s span) End() token.Pos { return s.end }

func withComments(n ast.Node, doc, comment *ast.CommentGroup) ast.Node {
	s := span{pos: n.Pos(), end: n.End()}
	if doc != nil && doc.Pos() < s.pos {
		s.pos = doc.Pos()
	}
	if comment != nil && comment.End() > s.end {
		s.end = comment.End()
	}
	return s
}

// fieldNodes returns the fields of fl, each extended by its comments.
// Free-floating comments inside the list are included as their own nodes
// so blank lines around them stay intact relative to the comment text.
func fieldNodes(fl *ast.FieldList, comments []*ast.CommentGroup) []ast.Node {
	nodes := make([]ast.Node, 0, len(fl.List))
	for _, f := range fl.List {
		nodes = append(nodes, withComments(f, f.Doc, f.Comment))
	}
	for _, cg := range comments {
		if cg.Pos() > fl.Opening && cg.End() < fl.Closing && !attached(cg, fl) {
			nodes = append(nodes, cg)
		}
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].Pos() < nodes[j].Pos() })
	return nodes
}

func attached(cg *ast.CommentGroup, fl *ast.FieldList) bool {
	for _, f := range fl.List {
		if f.Doc == cg || f.Comment == cg {
			return true
		}
		if cg.Pos() >= f.Pos() && cg.End() <= f.End() {
			return true
		}
	}
	return false
}

func specDoc(s ast.Spec) *ast.CommentGroup {
	switch s := s.(type) {
	case *ast.ValueSpec:
		return s.Doc
	case *ast.TypeSpec:
		return s.Doc
	}
	return nil
}

func specComment(s ast.Spec) *ast.CommentGroup {
	switch s := s.(type) {
	case *ast.ValueSpec:
		return s.Comment
	case *ast.TypeSpec:
		return s.Comment
	}
	return nil
}
