// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import "github.com/kastrul/gowsdl/pkg/parser"

// declarationNamespaces records the targetNamespace and elementFormDefault of
// the schema that declares each global element and complex type, so that
// XMLName tags use the namespace the declaration actually lives in rather than
// the namespace of the first schema in the WSDL, and so that one qualified
// schema does not make the declarations of an unqualified one qualified.
type declarationNamespaces struct {
	elements          map[*parser.XSDElement]string
	complexTypes      map[*parser.XSDComplexType]string
	qualified         map[any]bool
	fallback          string
	fallbackQualified bool
}

// resolveDeclarationNamespaces indexes the global elements and complex types
// of each schema by the schema's targetNamespace and elementFormDefault.
// Declarations in a schema without a targetNamespace are left out and resolve
// to fallback, and are qualified when any schema is.
func resolveDeclarationNamespaces(schemas []*parser.XSDSchema, fallback string) *declarationNamespaces {
	d := &declarationNamespaces{
		elements:     make(map[*parser.XSDElement]string),
		complexTypes: make(map[*parser.XSDComplexType]string),
		qualified:    make(map[any]bool),
		fallback:     fallback,
	}

	for _, schema := range schemas {
		qualified := schema.ElementFormDefault == "qualified"
		if qualified {
			d.fallbackQualified = true
		}
		ns := schema.TargetNamespace
		if ns == "" {
			continue
		}
		for _, el := range schema.Elements {
			d.elements[el] = ns
			d.qualified[el] = qualified
			if el.ComplexType != nil {
				d.complexTypes[el.ComplexType] = ns
				d.qualified[el.ComplexType] = qualified
			}
		}
		for _, ct := range schema.ComplexTypes {
			d.complexTypes[ct] = ns
			d.qualified[ct] = qualified
		}
	}

	return d
}

// isQualified reports whether the schema declaring decl, a global element or
// complex type, has elementFormDefault="qualified".
func (d *declarationNamespaces) isQualified(decl any) bool {
	if q, ok := d.qualified[decl]; ok {
		return q
	}
	return d.fallbackQualified
}

// element returns the targetNamespace of the schema declaring el.
func (d *declarationNamespaces) element(el *parser.XSDElement) string {
	if ns, ok := d.elements[el]; ok {
		return ns
	}
	return d.fallback
}

// complexType returns the targetNamespace of the schema declaring ct.
func (d *declarationNamespaces) complexType(ct *parser.XSDComplexType) string {
	if ns, ok := d.complexTypes[ct]; ok {
		return ns
	}
	return d.fallback
}
