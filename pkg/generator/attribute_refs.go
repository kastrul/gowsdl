// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"github.com/kastrul/gowsdl/pkg/parser"
)

// ResolveAttributeRefs fills in the name and type of attributes declared with
// ref="..." from the referenced global attribute declaration, so templates can
// treat them like locally declared attributes. Unresolvable refs default to
// xsd:string.
func ResolveAttributeRefs(schemas []*parser.XSDSchema) {
	globals := make(map[string]*parser.XSDAttribute)
	for _, schema := range schemas {
		for _, attr := range schema.Attributes {
			if attr.Name != "" {
				if _, exists := globals[attr.Name]; !exists {
					globals[attr.Name] = attr
				}
			}
		}
	}

	resolve := func(attrs []*parser.XSDAttribute) {
		for _, attr := range attrs {
			if attr.Name != "" || attr.Ref == "" {
				continue
			}
			attr.Name = removeNamespacePrefix(attr.Ref)
			if attr.Type != "" {
				continue
			}
			attr.Type = "xsd:string"
			if global, ok := globals[attr.Name]; ok {
				switch {
				case global.Type != "":
					attr.Type = global.Type
				case global.SimpleType != nil && global.SimpleType.Restriction.Base != "":
					attr.Type = global.SimpleType.Restriction.Base
				}
			}
		}
	}

	var walkComplexType func(ct *parser.XSDComplexType)

	walkElements := func(elements []*parser.XSDElement) {
		for _, el := range elements {
			if el.ComplexType != nil {
				walkComplexType(el.ComplexType)
			}
		}
	}

	walkComplexType = func(ct *parser.XSDComplexType) {
		resolve(ct.Attributes)
		resolve(ct.ComplexContent.Extension.Attributes)
		resolve(ct.SimpleContent.Extension.Attributes)
		walkElements(ct.Sequence)
		walkElements(ct.Choice)
		walkElements(ct.SequenceChoice)
		walkElements(ct.All)
		walkElements(ct.ComplexContent.Extension.Sequence)
		walkElements(ct.ComplexContent.Extension.Choice)
		walkElements(ct.ComplexContent.Extension.SequenceChoice)
	}

	for _, schema := range schemas {
		for _, ct := range schema.ComplexTypes {
			walkComplexType(ct)
		}
		walkElements(schema.Elements)
	}
}
