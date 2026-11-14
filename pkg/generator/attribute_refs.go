// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"github.com/kastrul/gowsdl/pkg/parser"
	"github.com/kastrul/gowsdl/pkg/utils"
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
			attr.Name = utils.LocalName(attr.Ref)
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

// resolveAttributeRefNamespaces maps each attribute declared with ref to the
// namespace URI of the referenced global attribute, under the same rule as
// resolveElementRefNamespaces: when that namespace differs from the target
// namespace of the referencing schema, or when that schema is not the primary
// one. Global attributes are namespace-qualified on the wire whatever the
// attributeFormDefault, as X-Road's id:objectType is. Refs whose prefix the
// schema does not declare are left out.
func resolveAttributeRefNamespaces(schemas []*parser.XSDSchema, primary string) map[*parser.XSDAttribute]string {
	result := make(map[*parser.XSDAttribute]string)

	for _, schema := range schemas {
		visited := make(map[*parser.XSDComplexType]bool)
		imported := schema.TargetNamespace != primary

		record := func(attrs []*parser.XSDAttribute) {
			for _, attr := range attrs {
				if attr.Ref == "" {
					continue
				}
				if ns := refNamespaceURI(schema, attr.Ref); ns != "" && (imported || ns != schema.TargetNamespace) {
					result[attr] = ns
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
			if ct == nil || visited[ct] {
				return
			}
			visited[ct] = true
			record(ct.Attributes)
			record(ct.ComplexContent.Extension.Attributes)
			record(ct.SimpleContent.Extension.Attributes)
			walkElements(ct.Sequence)
			walkElements(ct.Choice)
			walkElements(ct.SequenceChoice)
			walkElements(ct.All)
			walkElements(ct.ComplexContent.Extension.Sequence)
			walkElements(ct.ComplexContent.Extension.Choice)
			walkElements(ct.ComplexContent.Extension.SequenceChoice)
		}

		for _, ct := range schema.ComplexTypes {
			walkComplexType(ct)
		}
		walkElements(schema.Elements)
	}

	return result
}

// attrXMLName returns the name to use in the xml struct tag of an attribute:
// its local name, preceded by the referenced attribute's namespace URI when the
// attribute is declared with ref.
func (g *Generator) attrXMLName(attr *parser.XSDAttribute) string {
	if ns := g.attrNamespaces[attr]; ns != "" {
		return ns + " " + attr.Name
	}
	return attr.Name
}
