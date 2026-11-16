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
	// notScalar is a slice, soap.AnyType, one of the other struct types
	// the type mapper produces, or a type that cannot be resolved. Its xml
	// tag keeps omitempty.
	notScalar scalarKind = iota
	// boolOrNumber is a bool, integer or float, or a named type of one.
	boolOrNumber
	// textual is a string, or a named type of one, or one of the soap
	// date and time types.
	textual
	// complexStruct is the struct generated for a complex type, named or
	// anonymous. Its xml tag keeps omitempty.
	complexStruct
)

// sendsZeroValue reports whether a required field of kind k is sent even when
// it holds its zero value, i.e. whether its xml tag drops omitempty. A zero
// string is sent as an empty element or attribute. The soap date and time
// types write nothing for their zero value whatever the tag says.
func sendsZeroValue(k scalarKind) bool {
	return k == boolOrNumber || k == textual
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
// required when it occurs at most once (see maxOccursRepeats), its minOccurs
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

// elementFieldType returns goType, the type of the single, non-slice field
// generated for el, as a pointer when el is optional and of complex type, so
// that an unset field is left out of a request and an element absent from a
// response decodes to nil. A required element of complex type stays a struct
// value and is always sent. goType is returned unchanged when it is already a
// pointer, as it is for a nillable element.
//
// An element with an anonymous type and minOccurs="0" has always had a
// pointer field. Any other pointer is written as an optionalPointer marker,
// which breakRecursiveStructs reads as a by-value use. A repeated element
// gets a slice field from the types template and is left as it is here.
func (g *Generator) elementFieldType(el *parser.XSDElement, goType string) string {
	switch {
	case strings.HasPrefix(goType, "*"):
		return goType
	case el.ComplexType != nil && el.MinOccurs == "0":
		return "*" + goType
	case elementOptional(el) && !elementRepeated(el) && g.elementKind(el) == complexStruct:
		return optionalPointer(goType)
	}
	return goType
}

// elementRequired reports whether el must occur exactly once in every
// instance of the type declaring it.
func elementRequired(el *parser.XSDElement) bool {
	return !el.Nillable && !elementOptional(el) && !elementRepeated(el)
}

// elementRepeated reports whether el may occur more than once, so that its
// field is a slice: see maxOccursRepeats.
func elementRepeated(el *parser.XSDElement) bool {
	return maxOccursRepeats(el.MaxOccurs)
}

// maxOccursRepeats reports whether a particle with the given maxOccurs may
// occur more than once: maxOccurs is "unbounded" or a number greater than 1.
// An absent maxOccurs means 1. A value that is neither a number nor
// "unbounded" is taken to repeat, so that no occurrence is dropped.
func maxOccursRepeats(maxOccurs string) bool {
	maxOccurs = strings.TrimSpace(maxOccurs)
	if maxOccurs == "" {
		return false
	}
	n, err := strconv.Atoi(maxOccurs)
	return err != nil || n > 1
}

// elementOptional reports whether an instance of the type declaring el may
// leave el out: its minOccurs is 0, or a choice or an optional sequence or all
// encloses it.
func elementOptional(el *parser.XSDElement) bool {
	return particleOptional(el.MinOccurs, el.InOptionalGroup)
}

// particleOptional reports whether a particle, an element or a group
// reference, with the given minOccurs may be left out: its minOccurs is 0,
// or inOptionalGroup reports that a choice or an optional sequence or all
// encloses it.
func particleOptional(minOccurs string, inOptionalGroup bool) bool {
	if inOptionalGroup {
		return true
	}
	if minOccurs = strings.TrimSpace(minOccurs); minOccurs != "" {
		n, err := strconv.Atoi(minOccurs)
		return err != nil || n < 1
	}
	return false
}

// elementKind classifies the Go type the types template generates for the
// field of el.
func (g *Generator) elementKind(el *parser.XSDElement) scalarKind {
	switch {
	case el.Ref != "":
		return g.globalElementKind(utils.LocalName(el.Ref))
	case el.ComplexType != nil:
		return complexStruct
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
				return complexStruct
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
				return complexStruct
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
