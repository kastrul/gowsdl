// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// sequenceMember is an element of a repeated sequence that a struct holds:
// the xml tag name of its field and where that field is, as an index path
// from the occurrence type of the outermost sequence.
type sequenceMember struct {
	tag        string
	occurrence string
	path       []int
}

// addRepeatedSequenceUnmarshalers adds an UnmarshalXML method to every
// struct type in src that holds the occurrences of repeated sequences, as a
// field whose type is a slice of an occurrence type, directly or in a struct
// it embeds. Occurrence types are those whose MarshalXML calls
// soap.MarshalModelGroup.
//
// The method decodes through soap.UnmarshalModelGroups with one member field
// per element of the sequences, tagged as the field of that element in its
// occurrence type, so the names come from the struct tags the types
// template wrote. It runs after the passes that rename, pointerize or retag
// fields, on the code they produce. The source is returned unchanged if it
// cannot be parsed or holds no occurrence type.
func addRepeatedSequenceUnmarshalers(src []byte) []byte {
	const prefix = "package p\n"
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", prefix+string(src), parser.SkipObjectResolution)
	if err != nil {
		return src
	}

	structs := make(map[string]*ast.StructType)
	ends := make(map[string]token.Pos)
	var order []string
	occurrences := make(map[string]bool)
	unmarshalers := make(map[string]bool)
	for _, d := range file.Decls {
		switch d := d.(type) {
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok {
					if st, ok := ts.Type.(*ast.StructType); ok {
						structs[ts.Name.Name] = st
						ends[ts.Name.Name] = d.End()
						order = append(order, ts.Name.Name)
					}
				}
			}
		case *ast.FuncDecl:
			recv := receiverTypeName(d)
			switch {
			case d.Name.Name == "MarshalXML" && callsMarshalModelGroup(d):
				occurrences[recv] = true
			case d.Name.Name == "UnmarshalXML":
				unmarshalers[recv] = true
			}
		}
	}
	if len(occurrences) == 0 {
		return src
	}

	type insertion struct {
		offset int
		text   string
	}
	var inserts []insertion
	base := fset.File(file.Pos()).Base()
	for _, name := range order {
		if occurrences[name] || unmarshalers[name] {
			continue
		}
		members := heldSequenceMembers(name, structs, occurrences, map[string]bool{})
		if len(members) == 0 {
			continue
		}
		inserts = append(inserts, insertion{
			offset: int(ends[name]) - base - len(prefix),
			text:   repeatedSequenceUnmarshaler(name, members),
		})
	}

	sort.Slice(inserts, func(i, j int) bool { return inserts[i].offset > inserts[j].offset })
	out := append([]byte(nil), src...)
	for _, in := range inserts {
		out = append(out[:in.offset], append([]byte(in.text), out[in.offset:]...)...)
	}
	return out
}

// heldSequenceMembers returns the members of the repeated sequences the
// struct name holds, in field order, those of structs it embeds included.
// Of several members with the same tag only the first is kept, since a
// struct can name an element only once.
func heldSequenceMembers(name string, structs map[string]*ast.StructType, occurrences, visited map[string]bool) []sequenceMember {
	if visited[name] {
		return nil
	}
	visited[name] = true
	st := structs[name]
	if st == nil || st.Fields == nil {
		return nil
	}
	var members []sequenceMember
	for _, field := range st.Fields.List {
		if len(field.Names) == 0 {
			if id := embeddedFieldIdent(field.Type); id != nil && !occurrences[id.Name] {
				members = append(members, heldSequenceMembers(id.Name, structs, occurrences, visited)...)
			}
			continue
		}
		if occ := sliceElemIdent(field.Type); occ != "" && occurrences[occ] {
			members = append(members, occurrenceMembers(occ, occ, nil, structs, occurrences)...)
		}
	}
	seen := make(map[string]bool)
	var unique []sequenceMember
	for _, m := range members {
		if !seen[m.tag] {
			seen[m.tag] = true
			unique = append(unique, m)
		}
	}
	return unique
}

// occurrenceMembers returns the members of the occurrence type occ, nested
// in the occurrences of root at path.
func occurrenceMembers(root, occ string, path []int, structs map[string]*ast.StructType, occurrences map[string]bool) []sequenceMember {
	st := structs[occ]
	if st == nil || st.Fields == nil {
		return nil
	}
	var members []sequenceMember
	index := 0
	for _, field := range st.Fields.List {
		n := max(len(field.Names), 1)
		for range n {
			at := append(append([]int(nil), path...), index)
			if inner := sliceElemIdent(field.Type); inner != "" && occurrences[inner] {
				members = append(members, occurrenceMembers(root, inner, at, structs, occurrences)...)
			} else if tag := xmlTagName(field.Tag); tag != "" {
				members = append(members, sequenceMember{tag: tag, occurrence: root, path: at})
			}
			index++
		}
	}
	return members
}

// repeatedSequenceUnmarshaler returns the UnmarshalXML method of the struct
// name holding members.
func repeatedSequenceUnmarshaler(name string, members []sequenceMember) string {
	fieldPrefix := "M"
	for clash := true; clash; {
		clash = false
		for i := range members {
			if fmt.Sprintf("%s%d", fieldPrefix, i) == name {
				fieldPrefix += "M"
				clash = true
				break
			}
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "\n\n// UnmarshalXML implements xml.Unmarshaler. It decodes the elements of the\n")
	fmt.Fprintf(&b, "// repeated sequences %s holds into their occurrences: see\n", name)
	fmt.Fprintf(&b, "// soap.UnmarshalModelGroups.\n")
	fmt.Fprintf(&b, "func (t *%s) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {\n", name)
	fmt.Fprintf(&b, "\treturn soap.UnmarshalModelGroups(d, start, &struct {\n\t\t*%s\n", name)
	if name != "UnmarshalXML" {
		// The field hides the UnmarshalXML methods of the embedded struct.
		b.WriteString("\t\tUnmarshalXML struct{} `xml:\"-\"`\n")
	}
	for i, m := range members {
		fmt.Fprintf(&b, "\t\t%s%d soap.ModelGroupMember `xml:%s`\n", fieldPrefix, i, strconv.Quote(m.tag))
	}
	fmt.Fprintf(&b, "\t}{\n\t\t%s: t,\n", name)
	for i, m := range members {
		path := make([]string, len(m.path))
		for j, p := range m.path {
			path[j] = strconv.Itoa(p)
		}
		fmt.Fprintf(&b, "\t\t%s%d: soap.GroupMember[%s](%s),\n", fieldPrefix, i, m.occurrence, strings.Join(path, ", "))
	}
	b.WriteString("\t})\n}")
	return b.String()
}

// callsMarshalModelGroup reports whether fn calls soap.MarshalModelGroup.
func callsMarshalModelGroup(fn *ast.FuncDecl) bool {
	found := false
	ast.Inspect(fn, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			if x, ok := sel.X.(*ast.Ident); ok && x.Name == "soap" && sel.Sel.Name == "MarshalModelGroup" {
				found = true
			}
		}
		return !found
	})
	return found
}

// receiverTypeName returns the name of the type of fn's receiver, or "".
func receiverTypeName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return ""
	}
	if id := embeddedFieldIdent(fn.Recv.List[0].Type); id != nil {
		return id.Name
	}
	return ""
}

// sliceElemIdent returns the element type name of a field of type []Name,
// or "".
func sliceElemIdent(expr ast.Expr) string {
	arr, ok := expr.(*ast.ArrayType)
	if !ok || arr.Len != nil {
		return ""
	}
	if id, ok := arr.Elt.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

// xmlTagName returns the name part of the xml struct tag of a field,
// namespace included, or "" when it has none or is "-".
func xmlTagName(tag *ast.BasicLit) string {
	if tag == nil {
		return ""
	}
	raw, err := strconv.Unquote(tag.Value)
	if err != nil {
		return ""
	}
	name, _, _ := strings.Cut(reflect.StructTag(raw).Get("xml"), ",")
	if name == "-" {
		return ""
	}
	return name
}
