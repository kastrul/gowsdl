// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"strconv"
	"strings"

	"github.com/kastrul/gowsdl/pkg/parser"
	"github.com/kastrul/gowsdl/pkg/utils"
)

// scalarKind classifies the Go type generated for a schema type by what its
// zero value means on the wire.
type scalarKind int

const (
	// notScalar is a struct, slice, pointer or a type that cannot be
	// resolved. Its xml tag keeps omitempty.
	notScalar scalarKind = iota
	// boolOrNumber is a bool, integer or float, or a named type of one.
	boolOrNumber
	// textual is a string, or a named type of one, or one of the soap
	// date and time types.
	textual
)

// sendsZeroValue reports whether a required field of kind k is sent even when
// it holds its zero value, i.e. whether its xml tag drops omitempty.
func sendsZeroValue(k scalarKind) bool {
	return k == boolOrNumber
}

// goScalarKind classifies a Go type produced by the type mapper.
func goScalarKind(goType string) scalarKind {
	switch goType {
	case "bool",
		"int8", "int16", "int32", "int64",
		"uint8", "uint16", "uint32", "uint64",
		"float32", "float64":
		return boolOrNumber
	case "string", "soap.XSDDateTime", "soap.XSDDate", "soap.XSDTime":
		return textual
	}
	return notScalar
}

// elementOmitEmpty returns the ",omitempty" option for the xml tag of the
// field generated for el, or "" when the element is required and its field a
// scalar that must be sent even when it holds its zero value. An element is
// required when it occurs exactly once (maxOccurs absent or 1), its minOccurs
// is absent or at least 1, and no choice or optional sequence or all encloses
// it. Nillable elements are pointers and keep omitempty.
func (g *Generator) elementOmitEmpty(el *parser.XSDElement) string {
	if !elementRequired(el) || !sendsZeroValue(g.elementKind(el)) {
		return ",omitempty"
	}
	return ""
}

// attributeOmitEmpty returns the ",omitempty" option for the xml tag of the
// field generated for attr, or "" when the attribute has use="required" and
// its field a scalar that must be sent even when it holds its zero value.
func (g *Generator) attributeOmitEmpty(attr *parser.XSDAttribute) string {
	if strings.TrimSpace(attr.Use) != "required" || attr.Type == "" || !sendsZeroValue(g.typeKind(attr.Type, nil)) {
		return ",omitempty"
	}
	return ""
}

// elementRequired reports whether el must occur exactly once in every
// instance of the type declaring it.
func elementRequired(el *parser.XSDElement) bool {
	if el.InOptionalGroup || el.Nillable {
		return false
	}
	if maxOccurs := strings.TrimSpace(el.MaxOccurs); maxOccurs != "" && maxOccurs != "1" {
		return false
	}
	if minOccurs := strings.TrimSpace(el.MinOccurs); minOccurs != "" {
		n, err := strconv.Atoi(minOccurs)
		return err == nil && n >= 1
	}
	return true
}

// elementKind classifies the Go type the types template generates for the
// field of el.
func (g *Generator) elementKind(el *parser.XSDElement) scalarKind {
	switch {
	case el.Ref != "":
		return g.globalElementKind(utils.LocalName(el.Ref))
	case el.ComplexType != nil:
		return notScalar
	case el.Type != "":
		return g.typeKind(el.Type, nil)
	case el.SimpleType != nil:
		if base := el.SimpleType.Restriction.Base; base != "" {
			return g.typeKind(base, nil)
		}
		return textual
	}
	return notScalar
}

// globalElementKind classifies the Go type generated for the global element
// name, which is the type of the field of an element referring to it.
func (g *Generator) globalElementKind(name string) scalarKind {
	for _, schema := range g.getSchemas() {
		for _, el := range schema.Elements {
			if el.Name != name {
				continue
			}
			switch {
			case el.Type != "":
				return g.typeKind(el.Type, nil)
			case el.ComplexType != nil:
				return notScalar
			case el.SimpleType != nil:
				if el.SimpleType.Union.MemberTypes == "" && el.SimpleType.Restriction.Base != "" {
					return g.typeKind(el.SimpleType.Restriction.Base, nil)
				}
				return textual
			}
			return notScalar
		}
	}
	return notScalar
}

// typeKind classifies the Go type generated for the XSD type xsdType: a
// built-in type, or a named simple type followed through its restrictions.
// seen guards against restriction cycles.
func (g *Generator) typeKind(xsdType string, seen map[string]bool) scalarKind {
	if g.typeMapper.IsBuiltinType(xsdType) {
		return goScalarKind(g.typeMapper.MapXSDTypeToGoType(xsdType, false))
	}
	local := utils.LocalName(xsdType)
	if seen[local] {
		return notScalar
	}
	if seen == nil {
		seen = map[string]bool{}
	}
	seen[local] = true
	for _, schema := range g.getSchemas() {
		for _, ct := range schema.ComplexTypes {
			if ct.Name == local {
				return notScalar
			}
		}
	}
	for _, schema := range g.getSchemas() {
		for _, st := range schema.SimpleType {
			if st.Name != local {
				continue
			}
			switch {
			case st.List.ItemType != "":
				return notScalar
			case st.Union.MemberTypes != "":
				return textual
			case st.Restriction.Base != "":
				return g.typeKind(st.Restriction.Base, seen)
			}
			return textual
		}
	}
	return notScalar
}
