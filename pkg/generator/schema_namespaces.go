// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import "github.com/kastrul/gowsdl/pkg/parser"

// declarationNamespaces records the targetNamespace of the schema that
// declares each global element and complex type, so that XMLName tags use the
// namespace the declaration actually lives in rather than the namespace of the
// first schema in the WSDL.
type declarationNamespaces struct {
	elements     map[*parser.XSDElement]string
	complexTypes map[*parser.XSDComplexType]string
	fallback     string
}

// resolveDeclarationNamespaces indexes the global elements and complex types
// of each schema by the schema's targetNamespace. Declarations in a schema
// without a targetNamespace are left out and resolve to fallback.
func resolveDeclarationNamespaces(schemas []*parser.XSDSchema, fallback string) *declarationNamespaces {
	d := &declarationNamespaces{
		elements:     make(map[*parser.XSDElement]string),
		complexTypes: make(map[*parser.XSDComplexType]string),
		fallback:     fallback,
	}

	for _, schema := range schemas {
		ns := schema.TargetNamespace
		if ns == "" {
			continue
		}
		for _, el := range schema.Elements {
			d.elements[el] = ns
			if el.ComplexType != nil {
				d.complexTypes[el.ComplexType] = ns
			}
		}
		for _, ct := range schema.ComplexTypes {
			d.complexTypes[ct] = ns
		}
	}

	return d
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
